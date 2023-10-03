package service

import "gopkg.in/telebot.v3"

func ErrorHandlerFunc(h telebot.HandlerFunc, kbd *telebot.ReplyMarkup) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		err = h(ctx)
		if err != nil {
			_ = ctx.Send(err.Error(), kbd)
		}
		return
	}
}
