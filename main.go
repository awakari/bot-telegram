package main

import (
	"crypto/tls"
	"fmt"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	grpcApiMsgs "github.com/awakari/bot-telegram/api/grpc/messages"
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

	// init the internal Awakari messages storage client
	connMsgs, err := grpc.Dial(cfg.Api.Messages.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the messages service")
		defer connMsgs.Close()
	} else {
		log.Error("failed to connect the messages service", err)
	}
	clientMsgs := grpcApiMsgs.NewServiceClient(connMsgs)
	svcMsgs := grpcApiMsgs.NewService(clientMsgs)
	svcMsgs = grpcApiMsgs.NewServiceLogging(svcMsgs, log)

	// init the internal Awakari messages writer client that bypasses the limits
	clientAwkInternal, err := api.
		NewClientBuilder().
		WriterUri(cfg.Api.Writer.Uri).
		Build()
	defer clientAwkInternal.Close()

	// init handlers
	groupId := cfg.Api.GroupId
	callbackHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.CmdDelete:      subscriptions.DeleteHandlerFunc(clientAwk, groupId),
		subscriptions.CmdDetails:     subscriptions.DetailsHandlerFunc(clientAwk, groupId),
		subscriptions.CmdDescription: subscriptions.DescriptionHandlerFunc(clientAwk, groupId),
		subscriptions.CmdExtend:      subscriptions.ExtendReqHandlerFunc(),
	}
	webappHandlers := map[string]service.ArgHandlerFunc{
		service.LabelMsgSendCustom:   messages.PublishCustomHandlerFunc(clientAwk, groupId, svcMsgs, cfg.Payment),
		service.LabelSubCreateCustom: subscriptions.CreateCustomHandlerFunc(clientAwk, groupId),
		service.LabelLimitIncrease:   usage.ExtendLimitsInvoice(cfg.Payment),
	}
	txtHandlers := map[string]telebot.HandlerFunc{
		service.LabelSubList:        subscriptions.ListHandlerFunc(clientAwk, groupId),
		service.LabelSubCreateBasic: subscriptions.CreateBasicRequest,
		service.LabelMsgDetails:     messages.DetailsHandlerFunc(clientAwk, groupId),
		service.LabelMsgSendBasic:   messages.PublishBasicRequest,
	}
	menuKbd := service.MakeReplyKeyboard() // main menu keyboard
	replyHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.ReqDescribe:       subscriptions.DescriptionReplyHandlerFunc(clientAwk, groupId, menuKbd),
		subscriptions.ReqSubCreateBasic: subscriptions.CreateBasicReplyHandlerFunc(clientAwk, groupId, menuKbd),
		messages.ReqMsgPubBasic:         messages.PublishBasicReplyHandlerFunc(clientAwk, groupId, svcMsgs, cfg.Payment, menuKbd),
		subscriptions.ReqSubExtend:      subscriptions.ExtendReplyHandlerFunc(cfg.Payment, menuKbd),
		"support": func(tgCtx telebot.Context, args ...string) (err error) {
			return tgCtx.ForwardTo(&telebot.Chat{
				ID: cfg.Api.Telegram.SupportChatId,
			})
		},
	}
	preCheckoutHandlers := map[string]service.ArgHandlerFunc{
		usage.PurposeLimits:         usage.ExtendLimitsPreCheckout(clientAwk, groupId, cfg.Payment),
		subscriptions.PurposeExtend: subscriptions.ExtendPreCheckout(clientAwk, groupId, cfg.Payment),
		messages.PurposePublish:     messages.PublishPreCheckout(svcMsgs, cfg.Payment),
	}
	paymentHandlers := map[string]service.ArgHandlerFunc{
		usage.PurposeLimits:         usage.ExtendLimitsPaid(svcAdmin, groupId, log, cfg.Payment.Backoff),
		subscriptions.PurposeExtend: subscriptions.ExtendPaid(clientAwk, groupId, log, cfg.Payment.Backoff),
		messages.PurposePublish:     messages.PublishPaid(svcMsgs, clientAwkInternal, groupId, log, cfg.Payment.Backoff),
	}

	// init Telegram bot
	s := telebot.Settings{
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
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

	// set commands
	err = b.SetCommands([]telebot.CommandParams{
		{
			Commands: []telebot.Command{
				{
					Description: "Start",
					Text:        "/start",
				},
				{
					Description: "Help",
					Text:        "/help",
				},
				{
					Description: "Terms",
					Text:        "/terms",
				},
				{
					Description: "Privacy",
					Text:        "/privacy",
				},
				{
					Description: "Support",
					Text:        "/support",
				},
			},
			Scope: &telebot.CommandScope{
				Type: telebot.CommandScopeAllPrivateChats,
			},
		},
	})

	// assign handlers
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return service.LoggingHandlerFunc(next, log)
	})
	b.Handle("/start", service.ErrorHandlerFunc(service.StartHandlerFunc(menuKbd), menuKbd))
	b.Handle("/help", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.app/help.html\">help link</a>", telebot.ModeHTML)
	})
	b.Handle("/terms", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.app/terms.html\">terms link</a>", telebot.ModeHTML)
	})
	b.Handle("/privacy", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.app/privacy.html\">privacy link</a>", telebot.ModeHTML)
	})
	b.Handle("/support", func(tgCtx telebot.Context) error {
		_ = tgCtx.Send("Describe the issue in the reply to the next message")
		return tgCtx.Send("support")
	})
	b.Handle(telebot.OnCallback, service.ErrorHandlerFunc(service.Callback(callbackHandlers), menuKbd))
	b.Handle(telebot.OnText, service.ErrorHandlerFunc(service.TextHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnWebApp, service.ErrorHandlerFunc(service.WebAppData(webappHandlers), menuKbd))
	b.Handle(telebot.OnCheckout, service.ErrorHandlerFunc(service.PreCheckout(preCheckoutHandlers), menuKbd))
	b.Handle(telebot.OnPayment, service.ErrorHandlerFunc(service.Payment(paymentHandlers), menuKbd))

	b.Start()
}
