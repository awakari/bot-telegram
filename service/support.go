package service

import (
	"fmt"
	"gopkg.in/telebot.v3"
)

type SupportHandler struct {
	SupportChatId int64
	RestoreKbd    *telebot.ReplyMarkup
}

func (sh SupportHandler) Support(tgCtx telebot.Context, args ...string) (err error) {
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
