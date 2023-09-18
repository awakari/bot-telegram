package telegram

import (
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
)

type LoggingHandler struct {
	h   Handler
	log *slog.Logger
}

func NewLoggingHandler(h Handler, log *slog.Logger) Handler {
	return LoggingHandler{
		h:   h,
		log: log,
	}
}

func (l LoggingHandler) Handle(req *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	resp, err = l.h.Handle(req)
	switch err {
	case nil:
		l.log.Debug(fmt.Sprintf("Handler request id=%d text=%s, text=%s", req.MessageID, req.Text, resp.Text))
	default:
		l.log.Error(fmt.Sprintf("Handler request id=%d text=%s, text=%s, err=%s", req.MessageID, req.Text, resp.Text, err))
	}
	return
}
