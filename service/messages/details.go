package messages

import (
	"github.com/awakari/bot-telegram/service/sources"
	"gopkg.in/telebot.v3"
)

const LabelMainMenu = "< Main Menu"
const LabelPubAddSource = "+ Own Source"

var btnMainMenu = telebot.Btn{
	Text: LabelMainMenu,
}
var btnPubAddSource = telebot.Btn{
	Text: LabelPubAddSource,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/source-add.html",
	},
}

func Details(tgCtx telebot.Context) (err error) {
	if err == nil {
		m := &telebot.ReplyMarkup{}
		m.Inline(
			m.Row(
				telebot.Btn{
					Text: "All",
					Data: sources.CmdFeedListAll,
				},
				telebot.Btn{
					Text: "Own",
					Data: sources.CmdFeedListOwn,
				},
			),
		)
		err = tgCtx.Send("Source Feeds:", m)
	}
	if err == nil {
		m := &telebot.ReplyMarkup{}
		m.Inline(
			m.Row(
				telebot.Btn{
					Text: "All",
					Data: sources.CmdTgChListAll,
				},
				telebot.Btn{
					Text: "Own",
					Data: sources.CmdTgChListOwn,
				},
			),
		)
		err = tgCtx.Send("Source Telegram Channels:", m)
	}
	if err == nil {
		m := &telebot.ReplyMarkup{}
		m.Reply(m.Row(btnMainMenu, btnPubAddSource))
		err = tgCtx.Send("To add own source, use the corresponding reply keyboard button.", m)
	}
	return
}
