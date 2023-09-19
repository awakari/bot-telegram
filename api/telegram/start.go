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
	markup := &telebot.ReplyMarkup{}
	markup.Inline(
		telebot.Row{
			telebot.Btn{Text: "Set up new subscription", WebApp: &telebot.WebApp{URL: "https://google.com"}},
		},
	)
	return ctx.Send("Open this app!", markup)
}

func startMessage(ctx telebot.Context, chatId int64) (err error) {
	fmt.Printf("TODO: start message")
	return
}
