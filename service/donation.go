package service

import (
	"gopkg.in/telebot.v3"
)

const donationDefaultMsgTxt = "Help Awakari to be free"

func DonationHandler(ctx telebot.Context) (err error) {
	_, err = DonationMessage(ctx, donationDefaultMsgTxt)
	return
}

func DonationMessage(ctx telebot.Context, msgTxt string) (msg *telebot.Message, err error) {
	//customerId := "tg___user_id_" + strconv.FormatInt(ctx.Sender().ID, 10) + "%40awakari.com"
	link := "https://t.me/tribute/app?startapp=dcS8"
	msg, err = ctx.Bot().Send(ctx.Chat(), msgTxt, &telebot.ReplyMarkup{
		InlineKeyboard: [][]telebot.InlineButton{
			{
				telebot.InlineButton{
					Text: "Donate",
					URL:  link,
				},
			},
		},
	})
	return
}

func DonationMessagePin(ctx telebot.Context) (err error) {
	var msg *telebot.Message
	msg, err = DonationMessage(ctx, donationDefaultMsgTxt)
	if err == nil {
		err = ctx.Bot().Pin(msg)
	}
	return
}
