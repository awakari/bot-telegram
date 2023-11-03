package service

import (
	"gopkg.in/telebot.v3"
)

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelUsage = "Usage"
const LabelPublishing = "Publishing"
const LabelPubMsgBasic = "▷ Basic"
const LabelPubMsgCustom = "▷ Custom"
const LabelPubAddSource = "+ Source"

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

var btnUsage = telebot.Btn{
	Text: LabelUsage,
}

var btnMsgs = telebot.Btn{
	Text: LabelPublishing,
}

var btnMsgNewBasic = telebot.Btn{
	Text: LabelPubMsgBasic,
}

var btnMsgNewCustom = telebot.Btn{
	Text: LabelPubMsgCustom,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
}

var btnPubAddSource = telebot.Btn{
	Text: LabelPubAddSource,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/source-add.html",
	},
}

func MakeReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnUsage),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnUsage),
		kbd.Row(btnPubAddSource),
	)
	return
}
