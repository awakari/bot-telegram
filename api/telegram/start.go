package telegram

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StartHandler struct {
}

func NewStartHandler() Handler {
	return StartHandler{}
}

func (s StartHandler) Handle(req *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	resp = tgbotapi.NewMessage(req.Chat.ID, "Welcome to the Awakari bot!")
	menu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("New Subscription", "button1")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Edit Subscription", "button2")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Delete Subscription", "button3")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("New Custom Message", "button4")),
	)
	resp.ReplyMarkup = menu
	return
}
