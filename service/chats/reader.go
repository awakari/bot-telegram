package chats

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/client-sdk-go/api"
	clientAwkApiReader "github.com/awakari/client-sdk-go/api/grpc/reader"
	"github.com/awakari/client-sdk-go/model"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"go.uber.org/ratelimit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"log/slog"
	"math"
	"reflect"
	"sync"
	"time"
)

type Reader interface {
	Run(ctx context.Context, log *slog.Logger)
}

const ReaderTtl = 24 * time.Hour
const readBatchSize = 16
const msgFmtReadOnceFailed = "unexpected failure: %s\ndon't worry, retrying in %s..."
const backOffInit = 1 * time.Second
const backOffFactor = 3
const backOffMax = 24 * time.Hour
const msgExpired = "⚠ The subscription has been expired."
const msgExpiresSoon = "⏳ The subscription expires in %s."
const msgFmtExtendSteps = ` Please consider the following steps to extend it:
1. Go to your private chat with @AwakariBot.
2. Tap the "Subscriptions" reply keyboard button.
3. Select the subscription "%s".
4. Tap the "▲ Extend" button.`
const rateLimit = 20
const resumeBatchSize = 16

var optRateLimitPer = ratelimit.Per(time.Minute)
var runtimeReaders = make(map[string]*reader)
var runtimeReadersLock = &sync.Mutex{}

func ResumeAllReaders(
	ctx context.Context,
	log *slog.Logger,
	chatStor Storage,
	tgBot *telebot.Bot,
	clientAwk api.Client,
	format messages.Format,
	replicaIndex uint32,
	replicaRange uint32,
) (count uint32, err error) {
	cursor := int64(math.MinInt64)
	var page []Chat
	for {
		page, err = chatStor.GetBatch(ctx, replicaIndex, replicaRange, resumeBatchSize, cursor)
		if err != nil || len(page) == 0 {
			break
		}
		cursor = page[len(page)-1].Id
		for _, c := range page {
			u := telebot.Update{
				Message: &telebot.Message{
					Chat: &telebot.Chat{
						ID: c.Id,
					},
				},
			}
			r := NewReader(tgBot.NewContext(u), clientAwk, chatStor, c.Id, c.SubId, c.GroupId, c.UserId, format)
			go r.Run(context.Background(), log)
			count++
		}
	}
	return
}

func StopChatReaders(chatId int64) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	for _, r := range runtimeReaders {
		if r.chatId == chatId {
			r.stop = true
		}
	}
}

func StopChatReader(subId string) (found bool) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	var r *reader
	r, found = runtimeReaders[subId]
	if found {
		r.stop = true
	}
	return
}

func NewReader(tgCtx telebot.Context, clientAwk api.Client, chatStor Storage, chatId int64, subId, groupId, userId string, format messages.Format) Reader {
	return &reader{
		tgCtx:     tgCtx,
		clientAwk: clientAwk,
		chatStor:  chatStor,
		chatId:    chatId,
		subId:     subId,
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
	chatId    int64
	subId     string
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
		panic(err)
	}
}

func (r *reader) runOnce() (err error) {
	// prepare the context with a certain timeout
	ctx := context.Background()
	groupIdCtx := metadata.AppendToOutgoingContext(ctx, service.KeyGroupId, r.groupId)
	r.checkExpiration(groupIdCtx)
	groupIdCtx, cancel := context.WithTimeout(groupIdCtx, ReaderTtl)
	defer cancel()
	// get subscription info
	var sd subscription.Data
	sd, err = r.clientAwk.ReadSubscription(groupIdCtx, r.userId, r.subId)
	// open the events reader
	var readerAwk model.AckReader[[]*pb.CloudEvent]
	var subDescr string
	if err == nil {
		subDescr = sd.Description
		readerAwk, err = r.clientAwk.OpenMessagesAckReader(groupIdCtx, r.userId, r.subId, readBatchSize)
	}
	if err == nil {
		defer readerAwk.Close()
		err = r.deliverEventsReadLoop(ctx, readerAwk, subDescr)
	}
	switch {
	case errors.Is(err, clientAwkApiReader.ErrNotFound):
		_ = r.tgCtx.Send(fmt.Sprintf("subscription %s doesn't exist, stopping", r.subId))
		_ = r.chatStor.UnlinkSubscription(ctx, r.subId)
		r.stop = true
		err = nil
	}
	return
}

func (r *reader) checkExpiration(groupIdCtx context.Context) {
	sd, err := r.clientAwk.ReadSubscription(groupIdCtx, r.userId, r.subId)
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
		_ = r.tgCtx.Send(fmt.Sprintf("subscription %s doesn't exist, stopping", r.subId))
		_ = r.chatStor.UnlinkSubscription(ctx, r.subId)
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
				errTb := &telebot.Error{}
				if errors.As(err, &errTb) && errTb.Code == 403 {
					fmt.Printf("Bot blocked: %s, removing the chat from the storage", err)
					_, _ = r.chatStor.Delete(context.TODO(), r.tgCtx.Chat().ID)
					r.stop = true
					return
				}
				fmt.Printf("Failed to send message %+v to chat %d in HTML mode, cause: %s (%s)\n", tgMsg, r.tgCtx.Chat().ID, err, reflect.TypeOf(err))
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
	runtimeReaders[r.subId] = r
}

func (r *reader) runtimeUnregister(ctx context.Context) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	delete(runtimeReaders, r.subId)
}
