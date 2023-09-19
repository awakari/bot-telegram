package update

import (
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram/update/message"
	"github.com/awakari/bot-telegram/api/telegram/update/query"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	Handle(req tgbotapi.Update) (err error)
}

type handler struct {
	bot          *tgbotapi.BotAPI
	handlerMsg   message.Handler
	handlerQuery query.Handler
}

func NewHandler(bot *tgbotapi.BotAPI, handlerMsg message.Handler, handlerQuery query.Handler) Handler {
	return handler{
		bot:          bot,
		handlerMsg:   handlerMsg,
		handlerQuery: handlerQuery,
	}
}

func (h handler) Handle(req tgbotapi.Update) (err error) {
	var reply tgbotapi.MessageConfig
	cbq, msg := req.CallbackQuery, req.Message
	switch {
	case cbq != nil:
		_ = h.handlerQuery.Handle(cbq, &reply)
	case msg != nil:
		_ = h.handlerMsg.Handle(msg, &reply)
	default:
		fmt.Printf("TODO: handle non-message update %+v\n", req)
	}
	if reply.Text != "" {
		reply.ChatID = msg.Chat.ID
		_, err = h.bot.Send(reply)
	}
	return
}
