package subscriptions

import (
	"context"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"gopkg.in/telebot.v3"
)

const CmdStop = "sub_stop"

func Stop(chatStor chats.Storage) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		k := chats.Key{
			Id:    tgCtx.Chat().ID,
			SubId: args[0],
		}
		err = chatStor.UnlinkSubscription(context.Background(), k)
		if err == nil {
			_ = tgCtx.Send("Unlinked the subscription from this chat")
		}
		return
	}
}
