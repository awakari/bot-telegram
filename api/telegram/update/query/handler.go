package query

import (
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler interface {
	Handle(q *tgbotapi.CallbackQuery, resp *tgbotapi.MessageConfig) (err error)
}

const (
	SetSubscription = "set-sub"
	NewMessage      = "new-msg"
)

var ErrUnrecognizedQuery = errors.New("unrecognized query")

type handler struct {
}

func NewHandler() Handler {
	return handler{}
}

func (h handler) Handle(q *tgbotapi.CallbackQuery, resp *tgbotapi.MessageConfig) (err error) {
	switch q.ID {
	case SetSubscription:
	case NewMessage:
	default:
		err = fmt.Errorf("%w: %s", ErrUnrecognizedQuery, q.ID)
	}
	return
}
