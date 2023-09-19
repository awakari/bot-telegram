package command

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type startCommandHandler struct {
}

func NewStartCommandHandler() Handler {
	return startCommandHandler{}
}

func (s startCommandHandler) Handle(chat *tgbotapi.Chat, user *tgbotapi.User, cmd string) (resp tgbotapi.MessageConfig, err error) {
	return
}
