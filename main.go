package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/bot-telegram/config"
	"gopkg.in/telebot.v3"
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
	s := telebot.Settings{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		ParseMode: telebot.ModeMarkdownV2,
		Poller: &telebot.Webhook{
			Endpoint: &telebot.WebhookEndpoint{
				PublicURL: fmt.Sprintf("https://%s%s", cfg.Api.Host, cfg.Api.Path),
				Cert:      "/etc/server-cert/tls.crt",
			},
			HasCustomCert: true,
			Listen:        fmt.Sprintf(":%d", cfg.Api.Port),
		},
		Token: cfg.Api.Token,
	}
	var b *telebot.Bot
	b, err = telebot.NewBot(s)
	if err != nil {
		panic(err)
	}
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return telegram.LoggingHandlerFunc(next, log)
	})
	b.Handle("/start", telegram.Start)
	b.Handle(telegram.CmdPrefixSubCreateSimplePrefix, telegram.CreateTextSubscription)
	b.Handle(telebot.OnCallback, telegram.Callback)
	b.Handle(telebot.OnText, telegram.SubmitText)
	b.Start()
}
