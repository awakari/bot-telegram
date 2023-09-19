package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram/update"
	"github.com/awakari/bot-telegram/api/telegram/update/message"
	"github.com/awakari/bot-telegram/api/telegram/update/message/command"
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
	bot, err := tgbotapi.NewBotAPIWithClient(cfg.Api.Token, tgbotapi.APIEndpoint, &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	log.Info(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))
	certData, err := os.ReadFile("/etc/server-cert/tls.crt")
	if err != nil {
		panic(err)
	}
	certFile := tgbotapi.FileBytes{
		Name:  "server-cert",
		Bytes: certData,
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
	log.Info(fmt.Sprintf("Webhook listen path: %s", cfg.Api.Path))
	chUpdates := bot.ListenForWebhook(cfg.Api.Path)
	//
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	//
	go http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", cfg.Api.Port), nil)
	//
	startCmdHandler := command.NewStartCommandHandler()
	cmdHandlerByCmd := map[string]command.Handler{
		"start": startCmdHandler,
	}
	cmdHandler := command.NewHandler(cmdHandlerByCmd)
	cmdHandler = command.NewLoggingHandler(cmdHandler, log)
	msgHandler := message.NewHandler(cmdHandler)
	msgHandler = message.NewLoggingHandler(msgHandler, log)
	msgHandler = message.NewErrorHandler(msgHandler)
	updHandler := update.NewHandler(bot, msgHandler)
	updHandler = update.NewLoggingHandler(updHandler, log)
	log.Info("Start processing updates...")
	for u := range chUpdates {
		_ = updHandler.Handle(u)
	}
}
