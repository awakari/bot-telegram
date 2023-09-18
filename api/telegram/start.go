package telegram

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StartHandler struct {
}

func NewStartHandler() Handler {
	return StartHandler{}
}

func (s StartHandler) Handle(req *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	resp = tgbotapi.NewMessage(req.Chat.ID, fmt.Sprintf("Let's create a new subscription here for \"%s\"", req.Chat.Title))
	menu := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Create Subscription", "button1")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Send Custom Message", "button2")),
	)
	resp.ReplyMarkup = menu
	return
}
