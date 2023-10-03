package main

import (
	"crypto/tls"
	"fmt"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/bot-telegram/service/subscriptions"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"os"
)

func main() {

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
	clientAwk, err := api.
		NewClientBuilder().
		ApiUri(cfg.Api.Uri).
		Build()
	defer clientAwk.Close()

	// init Awakari admin client
	connAdmin, err := grpc.Dial(cfg.Api.Admin.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the limits admin service")
		defer connAdmin.Close()
	} else {
		log.Error("failed to connect the limits admin service", err)
	}
	clientAdmin := grpcApiAdmin.NewServiceClient(connAdmin)
	svcAdmin := grpcApiAdmin.NewService(clientAdmin)
	svcAdmin = grpcApiAdmin.NewServiceLogging(svcAdmin, log)

	// init handlers
	callbackHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.CmdDelete:      subscriptions.DeleteHandlerFunc(clientAwk, cfg.Api.GroupId),
		subscriptions.CmdDetails:     subscriptions.DetailsHandlerFunc(clientAwk, cfg.Api.GroupId),
		subscriptions.CmdDescription: subscriptions.DescriptionHandlerFunc(clientAwk, cfg.Api.GroupId),
		subscriptions.CmdDisable:     subscriptions.DisableHandlerFunc(clientAwk, cfg.Api.GroupId),
		subscriptions.CmdEnable:      subscriptions.EnableHandlerFunc(clientAwk, cfg.Api.GroupId),
	}
	webappHandlers := map[string]service.ArgHandlerFunc{
		service.LabelMsgSendCustom:   messages.PublishCustomHandlerFunc(clientAwk, cfg.Api.GroupId),
		service.LabelSubCreateCustom: subscriptions.CreateCustomHandlerFunc(clientAwk, cfg.Api.GroupId),
		service.LabelLimitIncrease:   usage.ExtendLimitsHandlerFunc(cfg.Api.PaymentProviderToken),
	}
	txtHandlers := map[string]telebot.HandlerFunc{
		service.LabelSubList:        subscriptions.ListHandlerFunc(clientAwk, cfg.Api.GroupId),
		service.LabelSubCreateBasic: subscriptions.CreateBasicRequest,
		service.LabelMsgDetails:     messages.DetailsHandlerFunc(clientAwk, cfg.Api.GroupId),
		service.LabelMsgSendBasic:   messages.PublishBasicRequest,
	}
	menuKbd := service.MakeReplyKeyboard() // main menu keyboard
	replyHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.ReqDescribe:       subscriptions.DescriptionReplyHandlerFunc(clientAwk, cfg.Api.GroupId, menuKbd),
		subscriptions.ReqSubCreateBasic: subscriptions.CreateBasicReplyHandlerFunc(clientAwk, cfg.Api.GroupId, menuKbd),
		messages.ReqMsgPubBasic:         messages.PublishBasicReplyHandlerFunc(clientAwk, cfg.Api.GroupId, menuKbd),
	}

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

	// assign handlers
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return service.LoggingHandlerFunc(next, log)
	})
	b.Handle("/start", service.ErrorHandlerFunc(service.StartHandlerFunc(menuKbd), menuKbd))
	b.Handle(telebot.OnCallback, service.ErrorHandlerFunc(service.Callback(callbackHandlers), menuKbd))
	b.Handle(telebot.OnText, service.ErrorHandlerFunc(service.TextHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnWebApp, service.ErrorHandlerFunc(service.WebAppData(webappHandlers), menuKbd))
	b.Handle(telebot.OnCheckout, service.ErrorHandlerFunc(usage.ExtendLimitsPreCheckout(clientAwk, cfg.Api.GroupId), menuKbd))
	b.Handle(telebot.OnPayment, service.ErrorHandlerFunc(usage.ExtendLimits(svcAdmin, cfg.Api.GroupId), menuKbd))

	b.Start()
}
