package command

import (
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram/update/query"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type startCommandHandler struct {
}

type ChatType string

const (
	ChatTypeGroup   ChatType = "group"
	ChatTypePrivate ChatType = "private"
)

var ErrNoUser = errors.New("start command: user is missing")
var ErrChatType = errors.New("unsupported chat type (supported options: \"group\", \"private\")")

func NewStartCommandHandler() Handler {
	return startCommandHandler{}
}

func (s startCommandHandler) Handle(chat *tgbotapi.Chat, user *tgbotapi.User, _ string, resp *tgbotapi.MessageConfig) (err error) {
	if user == nil {
		err = ErrNoUser
	}
	var userId int64
	if err == nil {
		userId = user.ID
		switch ChatType(chat.Type) {
		case ChatTypeGroup:
			err = s.menuSubscription(chat, userId, resp)
		case ChatTypePrivate:
			err = s.menuMessage(chat, userId, resp)
		default:
			err = fmt.Errorf("%w: %s", ErrChatType, chat.Type)
		}
	}
	return
}

func (s startCommandHandler) menuSubscription(chat *tgbotapi.Chat, userId int64, resp *tgbotapi.MessageConfig) (err error) {
	resp.Text = "Set up a new subscription to receive matching messages here. To complete, press the button below."
	cbd := query.SetSubscription
	url := "https://google.com"
	resp.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.InlineKeyboardButton{
				Text:         "Setup Subscription",
				URL:          &url,
				CallbackData: &cbd,
			},
		),
	)
	return
}

func (s startCommandHandler) menuMessage(chat *tgbotapi.Chat, userId int64, resp *tgbotapi.MessageConfig) (err error) {
	resp.Text = "Type a text to send a simple text message to Awakari. Press the button below to send a message with arbitrary attributes."
	resp.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("New Custom Message", query.NewMessage),
		),
	)
	return
}
