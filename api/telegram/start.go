package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

type startHandler struct {
	bot *telebot.Bot
}

var ErrChatType = errors.New("unsupported chat type (supported options: \"group\", \"private\")")

func NewStartHandler(bot *telebot.Bot) Handler {
	return startHandler{
		bot: bot,
	}
}

func (h startHandler) Handler() telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		chat := ctx.Chat()
		switch chat.Type {
		case telebot.ChatGroup:
			err = h.startSubscription(ctx, chat.ID)
		case telebot.ChatPrivate:
			err = h.startMessage(ctx, chat.ID)
		default:
			err = fmt.Errorf("%w: %s", ErrChatType, chat.Type)
		}
		return
	}
}

func (h startHandler) startSubscription(ctx telebot.Context, chatId int64) (err error) {
	markup := h.bot.NewMarkup()
	markup.Inline(telebot.Row{
		markup.WebApp("Open", &telebot.WebApp{
			URL: "https://google.com",
		}),
	})
	return ctx.Send("Open this app!", markup)
}

func (h startHandler) startMessage(ctx telebot.Context, chatId int64) (err error) {
	fmt.Printf("TODO: start message")
	return
}
