package command

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

func (l LoggingHandler) Handle(chat *tgbotapi.Chat, user *tgbotapi.User, cmd string) (resp tgbotapi.MessageConfig, err error) {
	resp, err = l.h.Handle(chat, user, cmd)
	switch err {
	case nil:
		l.log.Debug(fmt.Sprintf("Command \"%s\", user %+v, chat %+v: processing done, response text: %s", cmd, user, chat, resp.Text))
	default:
		l.log.Warn(fmt.Sprintf("Command \"%s\", user %+v, chat %+v: processing failed, err=%s, response text: %s", cmd, user, chat, err, resp.Text))
	}
	return
}
