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
const LabelSub = "🔍 Subscribe"

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
}

var btnSub = telebot.Btn{
	Text: LabelSub,
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
		kbd.Row(
			btnPub,
			telebot.Btn{
				Text: "Publishing >",
			},
		),
		kbd.Row(
			btnSub,
			telebot.Btn{
				Text: "Subscriptions >",
			},
		),
	)
	return
}
