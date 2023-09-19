package message

import (
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram/update/message/command"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	Handle(msg *tgbotapi.Message, resp *tgbotapi.MessageConfig) (err error)
}

type handler struct {
	handlerCmd command.Handler
}

func NewHandler(handlerCmd command.Handler) Handler {
	return handler{
		handlerCmd: handlerCmd,
	}
}

func (h handler) Handle(msg *tgbotapi.Message, resp *tgbotapi.MessageConfig) (err error) {
	cmd := msg.Command()
	switch cmd {
	case "":
		err = h.handleMessage(msg, resp)
	default:
		err = h.handlerCmd.Handle(msg.Chat, msg.From, cmd, resp)
	}
	return
}

func (h handler) handleMessage(msg *tgbotapi.Message, resp *tgbotapi.MessageConfig) (err error) {
	userLeft := msg.LeftChatMember
	switch {
	case userLeft != nil:
		fmt.Printf("User id=%d left, TODO: check if it's the same as subscription owner id and if so, delete the subscription\n", userLeft.ID)
	default:
		fmt.Printf("Handle message: %+v\n", msg)
	}
	return
}
