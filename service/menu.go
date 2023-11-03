package service

import (
	"gopkg.in/telebot.v3"
)

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelUsageSub = "Usageˢ"
const LabelPublishing = "Publishing"
const LabelPubMsgBasic = "▷ Basic"
const LabelPubMsgCustom = "▷ Custom"
const LabelUsagePub = "Usageᴾ"
const LabelMainMenu = "< Main Menu"

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

var btnUsageSub = telebot.Btn{
	Text: LabelUsageSub,
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

var btnUsagePub = telebot.Btn{
	Text: LabelUsagePub,
}

var BtnMainMenu = telebot.Btn{
	Text: LabelMainMenu,
}

func MakeReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnUsageSub),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnUsagePub),
	)
	return
}
