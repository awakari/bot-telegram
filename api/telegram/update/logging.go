package update

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

func (l LoggingHandler) Handle(req tgbotapi.Update) (err error) {
	err = l.h.Handle(req)
	switch err {
	case nil:
		l.log.Debug(fmt.Sprintf("Update processing done: %+v", req))
	default:
		l.log.Warn(fmt.Sprintf("Update processing failed: %+v, err=%s", req, err))
	}
	return
}
