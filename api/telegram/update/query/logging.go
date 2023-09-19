package query

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

func (l LoggingHandler) Handle(q *tgbotapi.CallbackQuery, resp *tgbotapi.MessageConfig) (err error) {
	err = l.h.Handle(q, resp)
	switch err {
	case nil:
		l.log.Debug(fmt.Sprintf("Query %+v processing done, response text: %s", q, resp.Text))
	default:
		l.log.Warn(fmt.Sprintf("Query %+v processing failed, err=%s, response text: %s", q, err, resp.Text))
	}
	return
}
