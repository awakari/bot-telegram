package message

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
		l.log.Debug(fmt.Sprintf("Message %d processing done, response text: %s", req.MessageID, resp.Text))
	default:
		l.log.Warn(fmt.Sprintf("Message %d processing failed, err=%s, response text: %s", req.MessageID, err, resp.Text))
	}
	return
}
