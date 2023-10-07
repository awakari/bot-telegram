package messages

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
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

var runtimeReaders = make(map[int64]*reader)
var runtimeReadersLock = &sync.Mutex{}

func ResumeAllReaders(ctx context.Context, log *slog.Logger, chatStor chats.Storage, tgBot *telebot.Bot, clientAwk api.Client, format Format) (count uint32, err error) {
	var resumingDone bool
	var c chats.Chat
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
		case errors.Is(nextErr, chats.ErrNotFound):
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
	for _, r := range runtimeReaders {
		r := r // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			c := chats.Chat{
				Key:     r.chatKey,
				State:   chats.StateInactive,
				Expires: time.Now().UTC(),
			}
			err := r.chatStor.Update(gCtx, c)
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

func NewReader(tgCtx telebot.Context, client api.Client, chatStor chats.Storage, chatKey chats.Key, groupId, userId string, format Format) Reader {
	return &reader{
		tgCtx:    tgCtx,
		client:   client,
		chatStor: chatStor,
		chatKey:  chatKey,
		groupId:  groupId,
		userId:   userId,
		format:   format,
	}
}

type reader struct {
	tgCtx    telebot.Context
	client   api.Client
	chatStor chats.Storage
	chatKey  chats.Key
	groupId  string
	userId   string
	stop     bool
	format   Format
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
		_ = r.chatStor.Delete(ctx, r.chatKey.Id)
	}
}

func (r *reader) runOnce() (err error) {
	ctx := context.Background()
	groupIdCtx := metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", r.groupId)
	r.checkExpiration(groupIdCtx)
	groupIdCtx, cancel := context.WithTimeout(groupIdCtx, ReaderTtl)
	defer cancel()
	var awakariReader model.Reader[[]*pb.CloudEvent]
	awakariReader, err = r.client.OpenMessagesReader(groupIdCtx, r.userId, r.chatKey.SubId, readBatchSize)
	if err == nil {
		defer awakariReader.Close()
		err = r.deliverEventsReadLoop(ctx, awakariReader)
	}
	if err == nil {
		nextChatState := chats.Chat{
			Key:     r.chatKey,
			Expires: time.Now().UTC().Add(ReaderTtl),
			State:   chats.StateActive,
		}
		err = r.chatStor.Update(ctx, nextChatState)
	}
	return
}

func (r *reader) checkExpiration(groupIdCtx context.Context) {
	sd, err := r.client.ReadSubscription(groupIdCtx, r.userId, r.chatKey.SubId)
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

func (r *reader) deliverEventsReadLoop(ctx context.Context, awakariReader model.Reader[[]*pb.CloudEvent]) (err error) {
	for !r.stop {
		err = r.deliverEventsRead(ctx, awakariReader)
		if err != nil {
			break
		}
	}
	return
}

func (r *reader) deliverEventsRead(ctx context.Context, awakariReader model.Reader[[]*pb.CloudEvent]) (err error) {
	//
	var evts []*pb.CloudEvent
	evts, err = awakariReader.Read()
	//
	switch status.Code(err) {
	case codes.NotFound:
		_ = r.tgCtx.Send(fmt.Sprintf("subscription doesn't exist: %s", r.chatKey.SubId))
		_ = r.chatStor.Delete(ctx, r.chatKey.Id)
		r.stop = true
		err = nil
	}
	//
	if len(evts) > 0 {
		_ = r.deliverEvents(evts)
	}
	//
	return err
}

func (r *reader) deliverEvents(evts []*pb.CloudEvent) (err error) {
	for _, evt := range evts {
		tgMsg := r.format.Convert(evt)
		err = r.tgCtx.Send(tgMsg, telebot.ModeHTML)
		if err != nil {
			fmt.Printf("Failed to send message in HTML mode, cause: %s\n", err)
			// fallback: try to re-send as a raw text
			err = r.tgCtx.Send(r.format.Plain(evt))
			if err != nil {
				fmt.Printf("Failed to send event to chat %d: %s\n%s\n", r.chatKey.Id, err, r.format.Plain(evt))
			}
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
	_ = r.chatStor.Update(ctx, chats.Chat{
		Key:   r.chatKey,
		State: chats.StateInactive,
	})
}
