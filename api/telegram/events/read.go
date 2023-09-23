package events

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/bot-telegram/chats"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const CmdSubRead = "readsub"
const readBatchSize = 16
const readerBackoff = 1 * time.Minute
const ReaderTtl = 24 * time.Hour

func SubscriptionReadHandlerFunc(awakariClient api.Client, chatStor chats.Storage, groupId string) telegram.ArgHandlerFunc {
	return func(ctx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		subId := args[0]
		chat := chats.Chat{
			Key: chats.Key{
				Id:    ctx.Chat().ID,
				SubId: subId,
			},
			State:   chats.StateActive,
			Expires: time.Now().Add(ReaderTtl),
		}
		err = chatStor.Create(context.TODO(), chat)
		if err == nil {
			go ChatEvents{
				TgCtx:    ctx,
				Client:   awakariClient,
				ChatStor: chatStor,
				ChatKey:  chat.Key,
				GroupId:  groupId,
				UserId:   userId,
			}.DeliveryLoop(context.Background())
		}
		return
	}
}

type ChatEvents struct {
	TgCtx    telebot.Context
	Client   api.Client
	ChatStor chats.Storage
	ChatKey  chats.Key
	GroupId  string
	UserId   string
}

func (ce ChatEvents) DeliveryLoop(ctx context.Context) {
	for {
		err := ce.deliverOnce(ctx)
		if err != nil {
			_ = ce.TgCtx.Send(
				fmt.Sprintf(
					`unexpected failure: %s,
to recover: try to create a new chat and select the same subscription`,
					err,
				),
			)
			_ = ce.ChatStor.Delete(ctx, ce.ChatKey)
			break
		}
	}
}

func (ce ChatEvents) deliverOnce(ctx context.Context) (err error) {
	groupIdCtx, cancel := context.WithTimeout(ctx, ReaderTtl)
	defer cancel()
	groupIdCtx = metadata.AppendToOutgoingContext(groupIdCtx, "x-awakari-group-id", ce.GroupId)
	var r model.Reader[[]*pb.CloudEvent]
	r, err = ce.Client.OpenMessagesReader(groupIdCtx, ce.UserId, ce.ChatKey.SubId, readBatchSize)
	switch err {
	case nil:
		defer r.Close()
		err = ce.deliverEventsReadLoop(ctx, r)
	default:
		_ = ce.TgCtx.Send(fmt.Sprintf("unexpected failure: %s,\nretrying after %s", err, readerBackoff))
		time.Sleep(readerBackoff)
		err = nil
	}
	if err == nil {
		nextChatState := chats.Chat{
			Key:     ce.ChatKey,
			Expires: time.Now().Add(ReaderTtl),
			State:   chats.StateActive,
		}
		err = ce.ChatStor.Update(ctx, nextChatState)
	}
	return
}

func (ce ChatEvents) deliverEventsReadLoop(ctx context.Context, r model.Reader[[]*pb.CloudEvent]) (err error) {
	for {
		err = ce.deliverEventsRead(ctx, r)
		if err != nil {
			break
		}
	}
	return
}

func (ce ChatEvents) deliverEventsRead(ctx context.Context, r model.Reader[[]*pb.CloudEvent]) (err error) {
	//
	var evts []*pb.CloudEvent
	evts, err = r.Read()
	//
	switch status.Code(err) {
	case codes.NotFound:
		_ = ce.ChatStor.Delete(ctx, ce.ChatKey)
	}
	//
	if len(evts) > 0 {
		_ = ce.deliverEvents(evts)
	}
	//
	return err
}

func (ce ChatEvents) deliverEvents(evts []*pb.CloudEvent) (err error) {
	for _, evt := range evts {
		err = ce.TgCtx.Send(evt.String())
		if err != nil {
			break
		}
	}
	return
}
