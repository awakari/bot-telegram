package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
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
	routes := map[string]telegram.Handler{
		"start": telegram.NewStartHandler(),
	}
	h := telegram.NewRouteHandler(routes)
	h = telegram.NewLoggingHandler(h, log)
	h = telegram.NewErrorHandler(h)
	log.Info("Start processing updates...")
	for update := range chUpdates {
		msg := update.Message
		if msg != nil {
			reply, _ := h.Handle(msg)
			_, err = bot.Send(reply)
			if err != nil {
				log.Error("failed to send reply", err)
			}
		}
	}
}
