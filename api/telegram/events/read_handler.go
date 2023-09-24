package events

import (
	"context"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/bot-telegram/chats"
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const CmdSubRead = "readsub"

func SubscriptionReadHandlerFunc(awakariClient api.Client, chatStor chats.Storage, groupId string) telegram.ArgHandlerFunc {
	return func(ctx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		subId := args[0]
		chat := chats.Chat{
			Key: chats.Key{
				Id:    ctx.Chat().ID,
				SubId: subId,
			},
			GroupId: groupId,
			UserId:  userId,
			State:   chats.StateActive,
			Expires: time.Now().Add(ReaderTtl),
		}
		err = chatStor.Create(context.TODO(), chat)
		if err == nil {
			r := NewReader(ctx, awakariClient, chatStor, chat.Key, groupId, userId)
			go r.Run(context.Background())
			_ = ctx.Send("Started, new messages by the subscription will appear here...")
		}
		return
	}
}
