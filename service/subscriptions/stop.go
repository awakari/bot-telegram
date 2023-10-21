package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"gopkg.in/telebot.v3"
)

const CmdStop = "sub_stop"

func Stop(chatStor chats.Storage) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		k := chats.Key{
			Id:    tgCtx.Chat().ID,
			SubId: subId,
		}
		err = chatStor.UnlinkSubscription(context.Background(), k)
		if err == nil {
			if chats.StopChatReader(subId) {
				_ = tgCtx.Send("Unlinked the subscription from this chat")
			} else {
				_ = tgCtx.Send(fmt.Sprintf("Unlinked the subscription from this chat. Note: don't delete this group for the next %s. Some new messages may appear here.", chats.ReaderTtl))
			}
		}
		return
	}
}
