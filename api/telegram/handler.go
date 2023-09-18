package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Handler interface {
	Handle(req *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error)
}
