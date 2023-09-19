package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

var ErrChatType = errors.New("unsupported chat type (supported options: \"group\", \"private\")")

func Start(ctx telebot.Context) (err error) {
	chat := ctx.Chat()
	switch chat.Type {
	case telebot.ChatGroup:
		err = startSubscription(ctx, chat.ID)
	case telebot.ChatPrivate:
		err = startMessage(ctx, chat.ID)
	default:
		err = fmt.Errorf("%w: %s", ErrChatType, chat.Type)
	}
	return
}

func startSubscription(ctx telebot.Context, chatId int64) (err error) {
	m := &telebot.ReplyMarkup{}
	m.Inline(
		m.Row(
			m.WebApp(
				"Setup new subscription",
				&telebot.WebApp{
					URL: "https://google.com",
				},
			),
		),
	)
	err = ctx.Send("Set up new subscription", m)
	return
}

func startMessage(ctx telebot.Context, chatId int64) (err error) {
	m := &telebot.ReplyMarkup{}
	m.Inline(
		m.Row(
			m.WebApp(
				"Setup new subscription",
				&telebot.WebApp{
					URL: "https://google.com",
				},
			),
		),
	)
	err = ctx.Send("Set up new subscription", m)
	return
}
