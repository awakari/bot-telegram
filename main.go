package main

import (
	"crypto/tls"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/bot-telegram/api/telegram/events"
	"github.com/awakari/bot-telegram/api/telegram/subscriptions"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/client-sdk-go/api"
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
	awakariClient, err := api.
		NewClientBuilder().
		ApiUri(cfg.Api.Uri).
		Build()
	defer awakariClient.Close()
	//
	s := telebot.Settings{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		Poller: &telebot.Webhook{
			Endpoint: &telebot.WebhookEndpoint{
				PublicURL: fmt.Sprintf("https://%s%s", cfg.Api.Telegram.Webhook.Host, cfg.Api.Telegram.Webhook.Path),
				Cert:      "/etc/server-cert/tls.crt",
			},
			HasCustomCert: true,
			Listen:        fmt.Sprintf(":%d", cfg.Api.Telegram.Webhook.Port),
		},
		Token: cfg.Api.Telegram.Token,
	}
	log.Debug(fmt.Sprintf("Telegram bot settigs: %+v", s))
	var b *telebot.Bot
	b, err = telebot.NewBot(s)
	if err != nil {
		panic(err)
	}
	//
	createSimpleSubHandlerFunc := subscriptions.CreateSimpleHandlerFunc(awakariClient, cfg.Api.GroupId)
	listSubsHandlerFunc := subscriptions.ListHandlerFunc(awakariClient, cfg.Api.GroupId)
	viewInboxHandlerFunc := events.ViewInboxHandlerFunc(awakariClient, cfg.Api.GroupId)
	argHandlers := map[string]telegram.ArgHandlerFunc{
		"viewsub": viewInboxHandlerFunc,
	}
	callbackHandlerFunc := telegram.Callback(argHandlers)
	//
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return telegram.LoggingHandlerFunc(next, log)
	})
	b.Handle("/start", telegram.ErrorHandlerFunc(telegram.StartHandlerFunc(listSubsHandlerFunc)))
	b.Handle(subscriptions.CmdPrefixSubCreateSimplePrefix, telegram.ErrorHandlerFunc(createSimpleSubHandlerFunc))
	b.Handle(telebot.OnCallback, telegram.ErrorHandlerFunc(callbackHandlerFunc))
	b.Handle(telebot.OnText, telegram.ErrorHandlerFunc(telegram.SubmitText))
	b.Start()
}
