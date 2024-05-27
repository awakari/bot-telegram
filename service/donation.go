package service

import (
	"gopkg.in/telebot.v3"
	"strconv"
)

func DonationHandler(ctx telebot.Context) (err error) {
	customerId := "tg___user_id=" + strconv.FormatInt(ctx.Sender().ID, 10) + "%40awakari.com"
	link := "https://donate.stripe.com/14k7uCaYq5befN65kk?prefilled_email=" + customerId
	return ctx.Send("Donate", telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "Donate",
					URL:  link,
				},
			},
		},
	})
}
