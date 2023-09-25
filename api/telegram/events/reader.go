package events

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/chats"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"sync"
	"time"
)

type Reader interface {
	Run(ctx context.Context)
}

const ReaderTtl = 24 * time.Hour
const readBatchSize = 16
const readerBackoff = 1 * time.Minute

var runtimeReaders = make(map[int64]*reader)
var runtimeReadersLock = &sync.Mutex{}

func ResumeAllReaders(ctx context.Context, chatStor chats.Storage, tgBot *telebot.Bot, awakariClient api.Client, format Format) (count uint32, err error) {
	var resumingDone bool
	var c chats.Chat
	var nextErr error
	for !resumingDone {
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
			r := NewReader(tgBot.NewContext(u), awakariClient, chatStor, c.Key, c.GroupId, c.UserId, format)
			go r.Run(context.Background())
			count++
		case errors.Is(nextErr, chats.ErrNotFound):
			resumingDone = true
		default:
			err = errors.Join(err, nextErr)
		}
	}
	return
}

func ReleaseAllChats(ctx context.Context) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	for _, r := range runtimeReaders {
		c := chats.Chat{
			Key:   r.chatKey,
			State: chats.StateInactive,
		}
		_ = r.chatStor.Update(ctx, c)
	}
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

func (r *reader) Run(ctx context.Context) {
	r.runtimeRegister(ctx)
	defer r.runtimeUnregister(ctx)
	//
	for !r.stop {
		err := r.runOnce(ctx)
		if err != nil {
			_ = r.tgCtx.Send(
				fmt.Sprintf(
					`unexpected failure: %s,
to recover: try to create a new chat and select the same subscription`,
					err,
				),
			)
			_ = r.chatStor.Delete(ctx, r.chatKey.Id)
			break
		}
	}
}

func (r *reader) runOnce(ctx context.Context) (err error) {
	groupIdCtx, cancel := context.WithTimeout(ctx, ReaderTtl)
	defer cancel()
	groupIdCtx = metadata.AppendToOutgoingContext(groupIdCtx, "x-awakari-group-id", r.groupId)
	var awakariReader model.Reader[[]*pb.CloudEvent]
	awakariReader, err = r.client.OpenMessagesReader(groupIdCtx, r.userId, r.chatKey.SubId, readBatchSize)
	switch err {
	case nil:
		defer awakariReader.Close()
		err = r.deliverEventsReadLoop(ctx, awakariReader)
	default:
		_ = r.tgCtx.Send(fmt.Sprintf("unexpected failure: %s,\nretrying after %s", err, readerBackoff))
		time.Sleep(readerBackoff)
		err = nil
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
		err = r.tgCtx.Send(r.format.Html(evt), telebot.ModeHTML)
		if err != nil {
			fmt.Printf("Failed to send events to chat %d: %s\n", r.chatKey.Id, err)
			break
		}
		if r.stop {
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
	_ = r.chatStor.Update(ctx, chats.Chat{
		Key:   r.chatKey,
		State: chats.StateInactive,
	})
}
