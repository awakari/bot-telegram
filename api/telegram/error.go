package telegram

import "gopkg.in/telebot.v3"

func ErrorHandlerFunc(h telebot.HandlerFunc) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		err = h(ctx)
		if err != nil {
			err = ctx.Send(err.Error())
		}
		return
	}
}
