package telegram

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
)

func TextHandlerFunc(
	awakariClient api.Client,
	groupId string,
	txtHandlers map[string]telebot.HandlerFunc,
	replyHandlers map[string]func(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) error,
) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		switch tgCtx.Message().IsReply() {
		case false:
			txt := tgCtx.Text()
			h, hOk := txtHandlers[txt]
			switch hOk {
			case true:
				err = h(tgCtx)
			default:
				err = errors.New(fmt.Sprintf("unrecognized command, use the reply keyboard"))
			}
		default:
			err = HandleReply(tgCtx, awakariClient, groupId, replyHandlers)
		}
		return
	}
}
