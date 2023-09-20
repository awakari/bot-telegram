package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

const msgStartGroup = "Select a subscription from the list"
const msgStartPrivate = `
* Just type a text to send a simple text message to Awakari.
* Send "/sub keyword1 keyword2 ..." command to create a basic text matching subscription.
`

var ErrChatType = errors.New("unsupported chat type (supported options: \"group\", \"private\")")

var btnSubNewCustom = telebot.Btn{
	Text: "Setup New Subscription",
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/web/sub-new-tg.html",
	},
}

var btnMsgNewCustom = telebot.Btn{
	Text: "Send Custom Message",
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/web/msg-new-tg.html",
	},
}

func Start(ctx telebot.Context) (err error) {
	chat := ctx.Chat()
	switch chat.Type {
	case telebot.ChatGroup:
		err = startGroup(ctx, chat.ID)
	case telebot.ChatPrivate:
		err = startPrivate(ctx, chat.ID)
	default:
		err = fmt.Errorf("%w: %s", ErrChatType, chat.Type)
	}
	return
}

func startGroup(ctx telebot.Context, chatId int64) (err error) {
	m := &telebot.ReplyMarkup{}
	m.Inline(
		m.Row(telebot.Btn{
			Unique: "sub0 unique",
			Text:   "sub0 text",
			Data:   "sub0 data",
		}),
		m.Row(telebot.Btn{
			Unique: "sub1 unique",
			Text:   "sub1 text",
			Data:   "sub1 data",
		}),
		m.Row(telebot.Btn{
			Unique: "sub2 unique",
			Text:   "sub2 text",
			Data:   "sub2 data",
		}),
	)
	err = ctx.Send(msgStartGroup, m)
	return
}

func startPrivate(ctx telebot.Context, chatId int64) (err error) {
	m := &telebot.ReplyMarkup{}
	m.Reply(m.Row(btnSubNewCustom, btnMsgNewCustom))
	err = ctx.Send(msgStartPrivate, m)
	return
}
