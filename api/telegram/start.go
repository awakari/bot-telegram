package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

var ErrChatType = errors.New("unsupported chat type (supported options: \"group\", \"private\")")

var setSub = &telebot.ReplyMarkup{
	ReplyKeyboard: [][]telebot.ReplyButton{
		{
			telebot.ReplyButton{
				WebApp: &telebot.WebApp{
					URL: "https://google.com",
				},
			},
		},
	},
}

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
	err = ctx.Send("Setup New Subscription", setSub)
	return
}
func startMessage(ctx telebot.Context, chatId int64) (err error) {
	fmt.Printf("TODO: start message")
	return
}
