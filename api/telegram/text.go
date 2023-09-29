package telegram

import (
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
)

func TextHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		switch tgCtx.Message().IsReply() {
		case false:
			err = SubmitText(tgCtx, awakariClient, groupId)
		default:
			err = HandleReply(tgCtx, awakariClient, groupId)
		}
		return
	}
}
