package service

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

func TextHandlerFunc(
	txtHandlers map[string]telebot.HandlerFunc,
	replyHandlers map[string]func(tgCtx telebot.Context, args ...string) error,
) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		switch tgCtx.Message().IsReply() {
		case true:
			err = HandleReply(tgCtx, replyHandlers)
		default:
			txt := tgCtx.Text()
			h, hOk := txtHandlers[txt]
			switch hOk {
			case true:
				err = h(tgCtx)
			default:
				err = errors.New(fmt.Sprintf("unrecognized command, use the reply keyboard"))
			}
		}
		return
	}
}
