package service

import (
	"gopkg.in/telebot.v3"
)

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelSubUsage = "Sub Usage"
const LabelMsgDetails = "Publishing"
const LabelMsgSendBasic = "▷ Basic"
const LabelMsgSendCustom = "▷ Custom"
const LabelPubUsage = "Pub Usage"

var btnSubList = telebot.Btn{
	Text: LabelSubList,
}

var btnSubNewBasic = telebot.Btn{
	Text: LabelSubCreateBasic,
}

var btnSubNewCustom = telebot.Btn{
	Text: LabelSubCreateCustom,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnSubUsage = telebot.Btn{
	Text: LabelSubUsage,
	//WebApp: &telebot.WebApp{
	//	URL: "https://awakari.app/price-calc-subs.html",
	//},
}

var btnMsgs = telebot.Btn{
	Text: LabelMsgDetails,
}

var btnMsgNewBasic = telebot.Btn{
	Text: LabelMsgSendBasic,
}

var btnMsgNewCustom = telebot.Btn{
	Text: LabelMsgSendCustom,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
}

var btnPubUsage = telebot.Btn{
	Text: LabelPubUsage,
}

func MakeReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnSubUsage),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnPubUsage),
	)
	return
}
