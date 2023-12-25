package support

import (
	"fmt"
	"gopkg.in/telebot.v3"
)

type Handler struct {
	SupportChatId int64
}

func (sh Handler) Request(tgCtx telebot.Context, args ...string) (err error) {
	tgCtxSupport := tgCtx.Bot().NewContext(telebot.Update{
		Message: &telebot.Message{
			Chat: &telebot.Chat{
				ID: sh.SupportChatId,
			},
		},
	})
	err = tgCtxSupport.Send(fmt.Sprintf("Support request from @%s:\n%s", tgCtx.Sender().Username, args[len(args)-1]))
	return
}
