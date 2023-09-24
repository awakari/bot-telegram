package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/bot-telegram/api/telegram/events"
	"github.com/awakari/bot-telegram/api/telegram/subscriptions"
	"github.com/awakari/bot-telegram/chats"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {

	ctx := context.TODO()

	// init config and logger
	slog.Info("starting...")
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		slog.Error("failed to load the config", err)
	}
	opts := slog.HandlerOptions{
		Level: slog.Level(cfg.Log.Level),
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &opts))

	// init Awakari client
	awakariClient, err := api.
		NewClientBuilder().
		ApiUri(cfg.Api.Uri).
		Build()
	defer awakariClient.Close()

	// init Telegram bot
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

	// init chat storage
	var chatStor chats.Storage
	chatStor, err = chats.NewStorage(ctx, cfg.Chats.Db)
	if err != nil {
		panic(err)
	}
	defer chatStor.Close()

	// init handlers
	createSimpleSubHandlerFunc := subscriptions.CreateSimpleHandlerFunc(awakariClient, cfg.Api.GroupId)
	listSubsHandlerFunc := subscriptions.ListHandlerFunc(awakariClient, cfg.Api.GroupId)
	readSubHandlerFunc := events.SubscriptionReadHandlerFunc(awakariClient, chatStor, cfg.Api.GroupId)
	argHandlers := map[string]telegram.ArgHandlerFunc{
		events.CmdSubRead: readSubHandlerFunc,
	}
	callbackHandlerFunc := telegram.Callback(argHandlers)

	// assign handlers
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return telegram.LoggingHandlerFunc(next, log)
	})
	b.Handle("/start", telegram.ErrorHandlerFunc(telegram.StartHandlerFunc(listSubsHandlerFunc)))
	b.Handle(subscriptions.CmdPrefixSubCreateSimplePrefix, telegram.ErrorHandlerFunc(createSimpleSubHandlerFunc))
	b.Handle(telebot.OnCallback, telegram.ErrorHandlerFunc(callbackHandlerFunc))
	b.Handle(telebot.OnUserLeft, telegram.ErrorHandlerFunc(telegram.UserLeft(chatStor)))
	b.Handle(telebot.OnText, telegram.ErrorHandlerFunc(telegram.SubmitText))

	log.Info("Resume previously existing inactive/expried chats...")
	var resumingDone bool
	for !resumingDone {
		c, err := chatStor.ActivateNext(ctx, time.Now().Add(events.ReaderTtl))
		switch {
		case err == nil:
			u := telebot.Update{
				Message: &telebot.Message{
					Chat: &telebot.Chat{
						ID: c.Key.Id,
					},
				},
			}
			r := events.NewReader(b.NewContext(u), awakariClient, chatStor, c.Key, c.GroupId, c.UserId)
			go r.Run(context.Background())
			log.Debug(fmt.Sprintf("Resumed the chat %d for the subscription %s", c.Key.Id, c.Key.SubId))
		case errors.Is(err, chats.ErrNotFound):
			log.Info(fmt.Sprintf("Resuming chats done"))
			resumingDone = true
		default:
			log.Error(fmt.Sprintf("failed to resume a chat %+v: %s", c.Key, err))
		}
	}
	defer events.StopAllReaders()

	b.Start()
}
