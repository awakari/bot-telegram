package messages

import (
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
)

func DetailsHandlerFunc(clientAwk api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		m := &telebot.ReplyMarkup{}
		m.Row(telebot.Btn{
			Text: "Common Sources",
		})
		m.Row(telebot.Btn{
			Text: "Own Sources",
		})
		err = tgCtx.Send("", m)
		return
	}
}
