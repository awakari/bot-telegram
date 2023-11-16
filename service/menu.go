package service

import (
	"gopkg.in/telebot.v3"
)

const LabelMainMenu = "< Main Menu"
const LabelPub = "📢 Publish"
const LabelPubs = "Publishing >"
const LabelSub = "🔍 Subscribe"
const LabelSubs = "Subscriptions >"

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
