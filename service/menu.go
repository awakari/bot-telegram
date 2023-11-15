package service

import (
	"gopkg.in/telebot.v3"
)

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelUsageSub = "Usage ˢ"
const LabelPublishing = "Publishing"
const LabelPubMsgBasic = "▷ Basic"
const LabelPubMsgCustom = "▷ Custom"
const LabelUsagePub = "Usage ᵖ"
const LabelMainMenu = "< Main Menu"

const LabelPub = "📢 Publish"
const LabelPubs = "Publishing >"
const LabelSub = "🔍 Subscribe"
const LabelSubs = "Subscriptions >"

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

var btnPub = telebot.Btn{
	Text: LabelPub,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
}

var btnPubs = telebot.Btn{
	Text: LabelPubs,
}

var btnSub = telebot.Btn{
	Text: LabelSub,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnSubs = telebot.Btn{
	Text: LabelSubs,
}

var BtnMainMenu = telebot.Btn{
	Text: LabelMainMenu,
}

func MakeReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnUsageSub),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnUsagePub),
	)
	return
}

func MakeMainMenu() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}
	kbd.Reply(
		kbd.Row(btnPub, btnSub),
		kbd.Row(btnPubs, btnSubs),
	)
	return
}
