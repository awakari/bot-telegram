package service

import "gopkg.in/telebot.v3"

func ErrorHandlerFunc(h telebot.HandlerFunc, restoreKbd *telebot.ReplyMarkup) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		err = h(ctx)
		if err != nil {
			switch restoreKbd {
			case nil:
				err = ctx.Send(err.Error())
			default:
				err = ctx.Send(err.Error(), restoreKbd)
			}
		}
		return
	}
}
