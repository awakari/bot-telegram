package telegram

import (
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
)

func HandleReply(tgCtx telebot.Context, awakariClient api.Client, groupId string) (err error) {
	reqMsg := tgCtx.Message()
	respMsg := reqMsg.ReplyTo
	err = tgCtx.Send("Request: %s\nResponse: %s\n", reqMsg.Text, respMsg.Text)
	return
}
