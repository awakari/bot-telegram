package service

import "gopkg.in/telebot.v3"

func ErrorHandlerFunc(h telebot.HandlerFunc, restoreKbd *telebot.ReplyMarkup) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		err = h(ctx)
		if err != nil {
			switch ctx.Chat().Type {
			case telebot.ChatPrivate:
				err = ctx.Send(err.Error(), restoreKbd)
			default:
				err = ctx.Send(err.Error())
			}
		}
		return
	}
}
