package chats

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/client-sdk-go/api"
	clientAwkApiReader "github.com/awakari/client-sdk-go/api/grpc/reader"
	"github.com/awakari/client-sdk-go/model"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"go.uber.org/ratelimit"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"log/slog"
	"sync"
	"time"
)

type Reader interface {
	Run(ctx context.Context, log *slog.Logger)
}

const ReaderTtl = 24 * time.Hour
const runtimeReaderCountLimit = 65_536
const readBatchSize = 16
const releaseChatsConcurrencyMax = 16
const msgFmtReadOnceFailed = "unexpected failure: %s\ndon't worry, retrying in %s..."
const msgFmtRunFatal = "fatal: %s,\nto recover: try to select a subscription again"
const backOffInit = 1 * time.Second
const backOffFactor = 3
const backOffMax = 24 * time.Hour
const msgExpired = "⚠ The subscription has been expired."
const msgExpiresSoon = "⏳ The subscription expires in %s."
const msgFmtExtendSteps = ` Please consider the following steps to extend it:
1. Go to @AwakariBot.
2. Tap the "Subscriptions" reply keyboard button.
3. Select the subscription "%s".
4. Tap the "▲ Extend" button.`
const rateLimit = 20

var optRateLimitPer = ratelimit.Per(time.Minute)
var runtimeReaders = make(map[int64]*reader)
var runtimeReadersLock = &sync.Mutex{}

func ResumeAllReaders(ctx context.Context, log *slog.Logger, chatStor Storage, tgBot *telebot.Bot, clientAwk api.Client, format messages.Format) (count uint32, err error) {
	var resumingDone bool
	var c Chat
	var nextErr error
	for !resumingDone && count < runtimeReaderCountLimit {
		c, nextErr = chatStor.ActivateNext(ctx, time.Now().UTC().Add(ReaderTtl))
		switch {
		case nextErr == nil:
			u := telebot.Update{
				Message: &telebot.Message{
					Chat: &telebot.Chat{
						ID: c.Key.Id,
					},
				},
			}
			r := NewReader(tgBot.NewContext(u), clientAwk, chatStor, c.Key, c.GroupId, c.UserId, format)
			go r.Run(context.Background(), log)
			count++
		case errors.Is(nextErr, ErrNotFound):
			resumingDone = true
		default:
			err = errors.Join(err, nextErr)
		}
	}
	return
}

func ReleaseAllChats(ctx context.Context, log *slog.Logger) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(releaseChatsConcurrencyMax)
	log.Info(fmt.Sprintf("Release %d readers", len(runtimeReaders)))
	for _, r := range runtimeReaders {
		r := r // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			c := Chat{
				Key:     r.chatKey,
				State:   StateInactive,
				Expires: time.Now().UTC(),
			}
			log.Info(fmt.Sprintf("Release reader: %+v", r.chatKey))
			err := r.chatStor.UpdateSubscriptionLink(gCtx, c)
			if err != nil {
				log.Error(fmt.Sprintf("Failed to release chat %d: %s", c.Key.Id, err))
			}
			return nil
		})
	}
	_ = g.Wait()
}

func StopChatReader(chatId int64) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	r, ok := runtimeReaders[chatId]
	if ok {
		r.stop = true
	}
}

func NewReader(tgCtx telebot.Context, clientAwk api.Client, chatStor Storage, chatKey Key, groupId, userId string, format messages.Format) Reader {
	return &reader{
		tgCtx:     tgCtx,
		clientAwk: clientAwk,
		chatStor:  chatStor,
		chatKey:   chatKey,
		groupId:   groupId,
		userId:    userId,
		format:    format,
		rl:        ratelimit.New(rateLimit, optRateLimitPer),
	}
}

type reader struct {
	tgCtx     telebot.Context
	clientAwk api.Client
	chatStor  Storage
	chatKey   Key
	groupId   string
	userId    string
	stop      bool
	format    messages.Format
	rl        ratelimit.Limiter
}

func (r *reader) Run(ctx context.Context, log *slog.Logger) {
	//
	r.runtimeRegister(ctx)
	defer r.runtimeUnregister(ctx)
	//
	var err error
	for err == nil && !r.stop {
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = backOffInit
		b.Multiplier = backOffFactor
		b.MaxInterval, b.MaxElapsedTime = backOffMax, backOffMax
		err = backoff.RetryNotify(r.runOnce, b, func(err error, d time.Duration) {
			log.Warn(fmt.Sprintf(msgFmtReadOnceFailed, err, d))
		})
		if errors.Is(err, context.DeadlineExceeded) {
			err = nil
		}
	}
	//
	if err != nil {
		err = r.tgCtx.Send(fmt.Sprintf(msgFmtRunFatal, err))
		_ = r.chatStor.UnlinkSubscription(ctx, r.chatKey)
		_ = r.tgCtx.Bot().Leave(r.tgCtx.Chat())
	}
}

func (r *reader) runOnce() (err error) {
	// prepare the context with a certain timeout
	ctx := context.Background()
	groupIdCtx := metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", r.groupId)
	r.checkExpiration(groupIdCtx)
	groupIdCtx, cancel := context.WithTimeout(groupIdCtx, ReaderTtl)
	defer cancel()
	// get subscription info
	var sd subscription.Data
	sd, err = r.clientAwk.ReadSubscription(groupIdCtx, r.userId, r.chatKey.SubId)
	// open the events reader
	var readerAwk model.AckReader[[]*pb.CloudEvent]
	var subDescr string
	if err == nil {
		subDescr = sd.Description
		readerAwk, err = r.clientAwk.OpenMessagesAckReader(groupIdCtx, r.userId, r.chatKey.SubId, readBatchSize)
	}
	if err == nil {
		defer readerAwk.Close()
		err = r.deliverEventsReadLoop(ctx, readerAwk, subDescr)
	}
	if err == nil {
		nextChatState := Chat{
			Key:     r.chatKey,
			Expires: time.Now().UTC().Add(ReaderTtl),
			State:   StateActive,
		}
		err = r.chatStor.UpdateSubscriptionLink(ctx, nextChatState)
	}
	switch {
	case errors.Is(err, clientAwkApiReader.ErrNotFound):
		_ = r.tgCtx.Send(fmt.Sprintf("subscription %s doesn't exist, stopping", r.chatKey.SubId))
		_ = r.chatStor.UnlinkSubscription(ctx, r.chatKey)
		r.stop = true
		err = nil
	}
	return
}

func (r *reader) checkExpiration(groupIdCtx context.Context) {
	sd, err := r.clientAwk.ReadSubscription(groupIdCtx, r.userId, r.chatKey.SubId)
	if err == nil {
		switch {
		case sd.Expires.IsZero(): // never expires
		case sd.Expires.Before(time.Now().UTC()):
			_ = r.tgCtx.Send(msgExpired + fmt.Sprintf(msgFmtExtendSteps, sd.Description))
		case sd.Expires.Sub(time.Now().UTC()) < 168*time.Hour: // expires earlier than in 1 week
			_ = r.tgCtx.Send(fmt.Sprintf(msgExpiresSoon, sd.Expires.Sub(time.Now().UTC()).Round(time.Minute)) + fmt.Sprintf(msgFmtExtendSteps, sd.Description))
		}
	}
}

func (r *reader) deliverEventsReadLoop(
	ctx context.Context,
	readerAwk model.AckReader[[]*pb.CloudEvent],
	subDescr string,
) (err error) {
	for !r.stop {
		err = r.deliverEventsRead(ctx, readerAwk, subDescr)
		if err != nil {
			break
		}
	}
	return
}

func (r *reader) deliverEventsRead(
	ctx context.Context,
	readerAwk model.AckReader[[]*pb.CloudEvent],
	subDescr string,
) (err error) {
	var evts []*pb.CloudEvent
	evts, err = readerAwk.Read()
	switch status.Code(err) {
	case codes.OK:
		var countAck uint32
		if len(evts) > 0 {
			countAck, err = r.deliverEvents(evts, subDescr)
		}
		_ = readerAwk.Ack(countAck)
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
				d := time.Second * time.Duration(err.(telebot.FloodError).RetryAfter)
				fmt.Printf("Flood error, retry in %s\n", d)
				time.Sleep(d)
				err = nil
			}
		}
	case codes.NotFound:
		_ = r.tgCtx.Send(fmt.Sprintf("subscription %s doesn't exist, stopping", r.chatKey.SubId))
		_ = r.chatStor.UnlinkSubscription(ctx, r.chatKey)
		r.stop = true
		err = nil
	}
	return err
}

func (r *reader) deliverEvents(evts []*pb.CloudEvent, subDescr string) (countAck uint32, err error) {
	for _, evt := range evts {
		r.rl.Take()
		tgMsg := r.format.Convert(evt, subDescr, messages.FormatModeHtml)
		err = r.tgCtx.Send(tgMsg, telebot.ModeHTML)
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
			default:
				fmt.Printf("Failed to send message %+v in HTML mode, cause: %s\n", tgMsg, err)
				tgMsg = r.format.Convert(evt, subDescr, messages.FormatModePlain)
				err = r.tgCtx.Send(tgMsg) // fallback: try to re-send as a plain text
			}
		}
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
			default:
				fmt.Printf("Failed to send message %+v in plain text mode, cause: %s\n", tgMsg, err)
				tgMsg = r.format.Convert(evt, subDescr, messages.FormatModeRaw)
				err = r.tgCtx.Send(tgMsg) // fallback: try to re-send as a raw text w/o file attachments
			}
		}
		//
		if err == nil {
			countAck++
		}
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
			default:
				fmt.Printf("FATAL: failed to send message %+v in raw text mode, cause: %s\n", tgMsg, err)
				countAck++ // to skip
			}
			break
		}
	}
	return
}

func (r *reader) runtimeRegister(_ context.Context) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	runtimeReaders[r.chatKey.Id] = r
}

func (r *reader) runtimeUnregister(ctx context.Context) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	delete(runtimeReaders, r.chatKey.Id)
	// try to do the best effort to mark chat as inactive in the chats DB
	_ = r.chatStor.UpdateSubscriptionLink(ctx, Chat{
		Key:   r.chatKey,
		State: StateInactive,
	})
}
