package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

var ErrChatType = errors.New("unsupported chat type (supported options: \"group\", \"private\")")

var btnSubNew = telebot.Btn{
	Text: "Setup New Subscription",
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/web/sub-new-tg.html",
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

	return
}

func startPrivate(ctx telebot.Context, chatId int64) (err error) {
	m := &telebot.ReplyMarkup{}
	m.Reply(m.Row(btnSubNew))
	err = ctx.Send("", m)
	return
}
