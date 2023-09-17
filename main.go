package main

import (
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	//
	slog.Info("starting...")
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		slog.Error("failed to load the config", err)
	}
	opts := slog.HandlerOptions{
		Level: slog.Level(cfg.Log.Level),
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	//
	bot, err := tgbotapi.NewBotAPI(cfg.Api.Token)
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	log.Info(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))
	certFile := tgbotapi.FileBytes{
		Name:  "server-cert",
		Bytes: []byte(cfg.Api.Cert),
	}
	wh, _ := tgbotapi.NewWebhookWithCert(fmt.Sprintf("https://%s%s", cfg.Api.Host, cfg.Api.Path), certFile)
	_, err = bot.Request(wh)
	if err != nil {
		panic(err)
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		panic(err)
	}
	if info.LastErrorDate != 0 {
		panic(err)
	}
	chUpdates := bot.ListenForWebhook(cfg.Api.Path)
	go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", cfg.Api.Port), nil)
	for update := range chUpdates {
		log.Info(fmt.Sprintf("%+v\n", update))
	}
}
