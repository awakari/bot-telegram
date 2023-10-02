package service

import (
	"encoding/json"
	"gopkg.in/telebot.v3"
	"log/slog"
)

func LoggingHandlerFunc(next telebot.HandlerFunc, log *slog.Logger) telebot.HandlerFunc {
	return func(ctx telebot.Context) error {
		data, _ := json.Marshal(ctx.Update())
		log.Debug(string(data))
		return next(ctx)
	}
}
