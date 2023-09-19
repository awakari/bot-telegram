package command

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	Handle(chat *tgbotapi.Chat, user *tgbotapi.User, cmd string) (resp tgbotapi.MessageConfig, err error)
}

var ErrUnrecognizedCommand = errors.New("unrecognized command")

type handler struct {
	handlerByCmd map[string]Handler
}

func NewHandler(handlerByCmd map[string]Handler) Handler {
	return handler{
		handlerByCmd: handlerByCmd,
	}
}

func (h handler) Handle(chat *tgbotapi.Chat, user *tgbotapi.User, cmd string) (resp tgbotapi.MessageConfig, err error) {
	hCmd, ok := h.handlerByCmd[cmd]
	if ok {
		resp, err = hCmd.Handle(chat, user, cmd)
	} else {
		err = fmt.Errorf("%w: %s", ErrUnrecognizedCommand, cmd)
	}
	return
}
