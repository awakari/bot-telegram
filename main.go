package main

import (
	"crypto/tls"
	"fmt"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
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
	createSimpleSubHandlerFunc := subscriptions.CreateSimpleHandlerFunc(awakariClient, cfg.Api.GroupId)
	listSubsHandlerFunc := subscriptions.ListHandlerFunc(awakariClient, cfg.Api.GroupId)
	callbackHandlers := map[string]func(ctx telebot.Context, args ...string) (err error){
		subscriptions.CmdDelete:      subscriptions.DeleteHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions.CmdDetails:     subscriptions.DetailsHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions.CmdDescription: subscriptions.DescriptionHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions.CmdDisable:     subscriptions.DisableHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions.CmdEnable:      subscriptions.EnableHandlerFunc(awakariClient, cfg.Api.GroupId),
	}
	callbackHandlerFunc := service.Callback(callbackHandlers)
	webappHandlers := map[string]func(ctx telebot.Context, args ...string) (err error){
		service.LabelMsgSendCustom:   service.SubmitCustomHandlerFunc(awakariClient, cfg.Api.GroupId),
		service.LabelSubCreateCustom: subscriptions.CreateCustomHandlerFunc(awakariClient, cfg.Api.GroupId),
		service.LabelLimitIncrease:   usage.ExtendLimitsHandlerFunc(cfg.Api.PaymentProviderToken),
	}
	txtHandlers := map[string]telebot.HandlerFunc{
		service.LabelSubList:        listSubsHandlerFunc,
		service.LabelSubCreateBasic: subscriptions.CreateBasicRequest,
	}
	replyHandlers := map[string]func(tgCtx telebot.Context, args ...string) error{
		subscriptions.ReqDescribe:       subscriptions.DescriptionReplyHandlerFunc(awakariClient, cfg.Api.GroupId),
		subscriptions.ReqSubCreateBasic: subscriptions.CreateBasicReplyHandlerFunc(awakariClient, cfg.Api.GroupId),
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
	b.Handle("/start", service.ErrorHandlerFunc(service.StartHandlerFunc()))
	b.Handle(fmt.Sprintf("/%s", subscriptions.CmdList), service.ErrorHandlerFunc(listSubsHandlerFunc))
	b.Handle(fmt.Sprintf("/%s", usage.CmdUsage), service.ErrorHandlerFunc(usage.ViewHandlerFunc(awakariClient, cfg.Api.GroupId)))
	b.Handle(subscriptions.CmdPrefixSubCreateSimplePrefix, service.ErrorHandlerFunc(createSimpleSubHandlerFunc))
	b.Handle(telebot.OnCallback, service.ErrorHandlerFunc(callbackHandlerFunc))
	b.Handle(telebot.OnText, service.ErrorHandlerFunc(service.TextHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnWebApp, service.ErrorHandlerFunc(service.WebAppData(webappHandlers)))
	b.Handle(telebot.OnCheckout, service.ErrorHandlerFunc(usage.ExtendLimitsPreCheckout(cfg.Api.GroupId)))

	b.Start()
}
