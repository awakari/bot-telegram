package main

import (
	"crypto/tls"
	"fmt"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	"github.com/awakari/bot-telegram/config"
	service2 "github.com/awakari/bot-telegram/service"
	subscriptions2 "github.com/awakari/bot-telegram/service/subscriptions"
	usage2 "github.com/awakari/bot-telegram/service/usage"
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
	awakariClient, err := api.
		NewClientBuilder().
		ApiUri(cfg.Api.Uri).
		Build()
	defer awakariClient.Close()

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
	createSimpleSubHandlerFunc := subscriptions2.CreateSimpleHandlerFunc(awakariClient, cfg.Api.GroupId)
	listSubsHandlerFunc := subscriptions2.ListHandlerFunc(awakariClient, cfg.Api.GroupId)
	callbackHandlers := map[string]func(ctx telebot.Context, args ...string) (err error){
		subscriptions2.CmdDelete:      subscriptions2.DeleteHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions2.CmdDetails:     subscriptions2.DetailsHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions2.CmdDescription: subscriptions2.DescriptionHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions2.CmdDisable:     subscriptions2.DisableHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions2.CmdEnable:      subscriptions2.EnableHandlerFunc(awakariClient, cfg.Api.GroupId),
	}
	callbackHandlerFunc := service2.Callback(callbackHandlers)
	webappHandlers := map[string]func(ctx telebot.Context, args ...string) (err error){
		service2.LabelMsgSendCustom:   service2.SubmitCustomHandlerFunc(awakariClient, cfg.Api.GroupId),
		service2.LabelSubCreateCustom: subscriptions2.CreateCustomHandlerFunc(awakariClient, cfg.Api.GroupId),
		service2.LabelLimitIncrease:   usage2.ExtendLimitsHandlerFunc(cfg.Api.PaymentProviderToken),
	}
	txtHandlers := map[string]telebot.HandlerFunc{
		service2.LabelSubList: listSubsHandlerFunc,
	}
	replyHandlers := map[string]func(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) error{
		subscriptions2.ReplyKeyDescription: subscriptions2.HandleDescriptionReply,
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
		return service2.LoggingHandlerFunc(next, log)
	})
	b.Handle("/start", service2.ErrorHandlerFunc(service2.StartHandlerFunc()))
	b.Handle(fmt.Sprintf("/%s", subscriptions2.CmdList), service2.ErrorHandlerFunc(listSubsHandlerFunc))
	b.Handle(fmt.Sprintf("/%s", usage2.CmdUsage), service2.ErrorHandlerFunc(usage2.ViewHandlerFunc(awakariClient, cfg.Api.GroupId)))
	b.Handle(subscriptions2.CmdPrefixSubCreateSimplePrefix, service2.ErrorHandlerFunc(createSimpleSubHandlerFunc))
	b.Handle(telebot.OnCallback, service2.ErrorHandlerFunc(callbackHandlerFunc))
	b.Handle(telebot.OnText, service2.ErrorHandlerFunc(service2.TextHandlerFunc(awakariClient, cfg.Api.GroupId, txtHandlers, replyHandlers)))
	b.Handle(telebot.OnWebApp, service2.ErrorHandlerFunc(service2.WebAppData(webappHandlers)))
	b.Handle(telebot.OnCheckout, service2.ErrorHandlerFunc(usage2.ExtendLimitsPreCheckout(cfg.Api.GroupId)))

	b.Start()
}
