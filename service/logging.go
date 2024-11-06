package service

import (
	"github.com/bytedance/sonic"
	"gopkg.in/telebot.v3"
	"log/slog"
)

func LoggingHandlerFunc(next telebot.HandlerFunc, log *slog.Logger) telebot.HandlerFunc {
	return func(ctx telebot.Context) error {
		data, _ := sonic.Marshal(ctx.Update())
		log.Debug(string(data))
		return next(ctx)
	}
}
