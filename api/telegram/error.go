package telegram

import (
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type ErrorHandler struct {
	h Handler
}

func NewErrorHandler(h Handler) Handler {
	return ErrorHandler{
		h: h,
	}
}

func (e ErrorHandler) Handle(req *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	resp, err = e.h.Handle(req)
	if err != nil {
		resp.Text += err.Error()
	}
	return
}
