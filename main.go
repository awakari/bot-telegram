package main

import (
	"crypto/tls"
	"fmt"
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
		Token: cfg.Api.Token,
		Poller: &telebot.Webhook{
			Listen:        fmt.Sprintf(":%d", cfg.Api.Port),
			HasCustomCert: true,
			Endpoint: &telebot.WebhookEndpoint{
				PublicURL: fmt.Sprintf("https://%s%s", cfg.Api.Host, cfg.Api.Path),
				Cert:      "/etc/server-cert/tls.crt",
			},
		},
		Verbose: true,
	}
	var b *telebot.Bot
	b, err = telebot.NewBot(s)
	if err != nil {
		panic(err)
	}
	b.Handle(telebot.OnText, func(ctx telebot.Context) error {
		log.Info(fmt.Sprintf("update: %+v", ctx.Update()))
		return ctx.Send(ctx.Text())
	})
	b.Start()
}
