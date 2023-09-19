package message

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

func (e errorHandler) Handle(msg *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	resp, err = e.h.Handle(msg)
	if err != nil {
		resp.Text += err.Error()
	}
	return
}
