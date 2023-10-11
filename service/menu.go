package service

import (
	"gopkg.in/telebot.v3"
)

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelMsgDetails = "Messages Publishing"
const LabelMsgSendBasic = "▸ Basic"
const LabelMsgSendCustom = "▸ Custom"
const LabelLimitIncrease = "▲ Limit"

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

var btnLimitIncrSubs = telebot.Btn{
	Text: LabelLimitIncrease,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/price-calc-subs.html",
	},
}

var btnLimitIncrMsgs = telebot.Btn{
	Text: LabelLimitIncrease,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/price-calc-msgs.html",
	},
}

func MakeReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnLimitIncrSubs),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnLimitIncrMsgs),
	)
	return
}