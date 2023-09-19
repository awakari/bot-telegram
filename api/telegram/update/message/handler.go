package message

import (
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram/update/message/command"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	Handle(msg *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error)
}

type handler struct {
	handlerCmd command.Handler
}

func NewHandler(handlerCmd command.Handler) Handler {
	return handler{
		handlerCmd: handlerCmd,
	}
}

func (h handler) Handle(msg *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	cmd := msg.Command()
	switch cmd {
	case "":
		fmt.Printf("TODO: handle non-command message \"%s\"\n", msg.Text)
	default:
		resp, err = h.handlerCmd.Handle(msg.Chat, msg.From, cmd)
	}
	return
}
