package query

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type errorHandler struct {
	h Handler
}

func NewErrorHandler(h Handler) Handler {
	return errorHandler{
		h: h,
	}
}

func (e errorHandler) Handle(q *tgbotapi.CallbackQuery, resp *tgbotapi.MessageConfig) (err error) {
	err = e.h.Handle(q, resp)
	if err != nil {
		resp.Text += err.Error()
	}
	return
}
