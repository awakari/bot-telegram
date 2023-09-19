package update

import (
	"github.com/awakari/bot-telegram/api/telegram/update/message"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	Handle(req tgbotapi.Update) (err error)
}

type handler struct {
	bot        *tgbotapi.BotAPI
	handlerMsg message.Handler
}

func NewHandler(bot *tgbotapi.BotAPI, handlerMsg message.Handler) Handler {
	return handler{
		bot:        bot,
		handlerMsg: handlerMsg,
	}
}

func (h handler) Handle(req tgbotapi.Update) (err error) {
	msg := req.Message
	switch msg {
	case nil:
	default:
		reply, _ := h.handlerMsg.Handle(msg)
		_, err = h.bot.Send(reply)
	}
	return
}
