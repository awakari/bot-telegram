package main

import (
	"context"
	"crypto/tls"
	"fmt"
	grpcApi "github.com/awakari/bot-telegram/api/grpc"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	grpcApiAuth "github.com/awakari/bot-telegram/api/grpc/auth"
	grpcApiMsgs "github.com/awakari/bot-telegram/api/grpc/messages"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/bot-telegram/service/subscriptions"
	"github.com/awakari/bot-telegram/service/support"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	"github.com/microcosm-cc/bluemonday"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	// determine the replica index
	replicaNameParts := strings.Split(cfg.Replica.Name, "-")
	if len(replicaNameParts) < 2 {
		panic("unable to parse the replica name: " + cfg.Replica.Name)
	}
	var replicaIndexTmp uint64
	replicaIndexTmp, err = strconv.ParseUint(replicaNameParts[len(replicaNameParts)-1], 10, 16)
	if err != nil {
		panic(err)
	}
	replicaIndex := uint32(replicaIndexTmp)
	log.Info(fmt.Sprintf("Replica: %d/%d", replicaIndex, cfg.Replica.Range))

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

	// init the Telegram Login validation grpc service
	controllerAuth := grpcApiAuth.NewController([]byte(cfg.Api.Telegram.Token))
	go func() {
		log.Info(fmt.Sprintf("starting to listen the grpc API @ port #%d...", cfg.Api.Telegram.Auth.Port))
		err = grpcApi.Serve(cfg.Api.Telegram.Auth.Port, controllerAuth)
		if err != nil {
			panic(err)
		}
	}()

	// init chat storage
	var chatStor chats.Storage
	chatStor, err = chats.NewStorage(ctx, cfg.Chats.Db)
	if err != nil {
		panic(err)
	}
	defer chatStor.Close()

	// init events format, see https://core.telegram.org/bots/api#html-style for details
	htmlPolicy := bluemonday.NewPolicy()
	htmlPolicy.AllowStandardURLs()
	htmlPolicy.
		AllowAttrs("href").
		OnElements("a")
	htmlPolicy.AllowElements("b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "code", "pre")
	htmlPolicy.
		AllowAttrs("class").
		OnElements("span")
	htmlPolicy.AllowURLSchemes("tg")
	htmlPolicy.
		AllowAttrs("emoji-ids").
		OnElements("tg-emoji")
	htmlPolicy.
		AllowAttrs("class").
		OnElements("code")
	htmlPolicy.AllowDataURIImages()
	msgFmt := messages.Format{
		HtmlPolicy: htmlPolicy,
	}

	// init handlers
	groupId := cfg.Api.GroupId
	supportHandler := support.Handler{
		SupportChatId: cfg.Api.Telegram.SupportChatId,
	}
	limitsHandler := usage.LimitsHandler{
		CfgPayment:     cfg.Payment,
		ClientAdmin:    svcAdmin,
		ClientAwk:      clientAwk,
		GroupId:        groupId,
		Log:            log,
		SupportHandler: supportHandler,
	}
	subExtHandler := subscriptions.ExtendHandler{
		CfgPayment: cfg.Payment,
		ClientAwk:  clientAwk,
		GroupId:    groupId,
		Log:        log,
	}
	callbackHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.CmdDescription: subscriptions.DescriptionHandlerFunc(clientAwk, groupId),
		subscriptions.CmdExtend:      subExtHandler.RequestExtensionDaysCount,
		subscriptions.CmdStart:       subscriptions.Start(log, clientAwk, chatStor, groupId, msgFmt),
		subscriptions.CmdStop:        subscriptions.Stop(chatStor),
		subscriptions.CmdPageNext:    subscriptions.PageNext(clientAwk, chatStor, groupId),
		usage.CmdExtend:              limitsHandler.RequestExtension,
		usage.CmdIncrease:            limitsHandler.RequestIncrease,
	}
	webappHandlers := map[string]service.ArgHandlerFunc{
		usage.LabelExtend: limitsHandler.HandleExtension,
	}
	txtHandlers := map[string]telebot.HandlerFunc{}
	replyHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.ReqDescribe:  subscriptions.DescriptionReplyHandlerFunc(clientAwk, groupId),
		subscriptions.ReqSubCreate: subscriptions.CreateBasicReplyHandlerFunc(clientAwk, groupId),
		messages.ReqMsgPub:         messages.PublishBasicReplyHandlerFunc(clientAwk, groupId, svcMsgs, cfg.Payment),
		subscriptions.ReqSubExtend: subExtHandler.HandleExtensionReply,
		usage.ReqLimitExtend:       limitsHandler.HandleExtension,
		usage.ReqLimitIncrease:     limitsHandler.HandleIncrease,
		"support":                  supportHandler.Request,
	}
	preCheckoutHandlers := map[string]service.ArgHandlerFunc{
		usage.PurposeLimitSet:       limitsHandler.PreCheckout,
		subscriptions.PurposeExtend: subExtHandler.ExtensionPreCheckout,
		messages.PurposePublish:     messages.PublishPreCheckout(svcMsgs, cfg.Payment),
	}
	paymentHandlers := map[string]service.ArgHandlerFunc{
		usage.PurposeLimitSet:       limitsHandler.HandleLimitOrderPaid,
		subscriptions.PurposeExtend: subExtHandler.ExtendPaid,
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
			HasCustomCert:  true,
			Listen:         fmt.Sprintf(":%d", cfg.Api.Telegram.Webhook.Port),
			MaxConnections: int(cfg.Api.Telegram.Webhook.ConnMax),
			SecretToken:    cfg.Api.Telegram.Webhook.Token,
		},
		Token: cfg.Api.Telegram.Token,
	}
	log.Debug(fmt.Sprintf("Telegram bot settigs: %+v", s))
	var b *telebot.Bot
	b, err = telebot.NewBot(s)
	if err != nil {
		panic(err)
	}
	err = b.SetCommands([]telebot.Command{
		{
			Text:        "start",
			Description: "Start",
		},
		{
			Text:        "pub",
			Description: "Publish a basic message",
		},
		{
			Text:        "sub",
			Description: "Subscribe for keywords",
		},
		{
			Text:        "donate",
			Description: "Help Awakari to be Free",
		},
		{
			Text:        "help",
			Description: "User guide",
		},
		{
			Text:        "support",
			Description: "Request support",
		},
		{
			Text:        "terms",
			Description: "Terms of service",
		},
		{
			Text:        "privacy",
			Description: "Privacy policy",
		},
	})
	if err != nil {
		panic(err)
	}

	// resolve the donation invoice - should be pinned in the dedicated private channel
	var dCh *telebot.Chat
	dCh, err = b.ChatByID(cfg.Payment.DonationChatId)
	if err != nil {
		panic(err)
	}
	donateMsg := dCh.PinnedMessage
	if donateMsg == nil {
		panic(fmt.Sprintf("Failed to resolve the pinned donation invoice message in the chat: %+v", dCh))
	}

	// assign handlers
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return service.LoggingHandlerFunc(next, log)
	})
	subListHandlerFunc := subscriptions.ListOnGroupStartHandlerFunc(clientAwk, chatStor, groupId)
	b.Handle(
		"/start",
		service.ErrorHandlerFunc(func(tgCtx telebot.Context) (err error) {
			chat := tgCtx.Chat()
			switch chat.Type {
			case telebot.ChatChannel:
				log.Info(fmt.Sprintf("Started in the public channel %+v", chat))
				err = tgCtx.Send("Wow, I'm in a public channel!")
			case telebot.ChatGroup:
				err = subListHandlerFunc(tgCtx)
			case telebot.ChatSuperGroup:
				err = subListHandlerFunc(tgCtx)
			case telebot.ChatPrivate:
				var msg *telebot.Message
				msg, err = b.Forward(chat, donateMsg)
				if err == nil {
					err = b.Pin(msg)
				}
				log.Warn(fmt.Sprintf("Failed to forward or pin the donation invoice in the chat %+v, cause: %s", chat, err))
				err = tgCtx.Send("Use the commands menu. To receive messages, invite this bot to a group chat and select a subscription.")
			default:
				err = fmt.Errorf("unsupported chat type (supported options: \"private\", \"group\", \"supergroup\"): %s", chat.Type)
			}
			return
		}),
	)
	b.Handle("/pub", messages.PublishBasicRequest)
	b.Handle("/sub", subscriptions.CreateBasicRequest)
	b.Handle("/donate", func(tgCtx telebot.Context) (err error) {
		return tgCtx.Forward(donateMsg)
	})
	b.Handle("/help", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.com\">link</a>", telebot.ModeHTML)
	})
	b.Handle("/support", func(tgCtx telebot.Context) error {
		_ = tgCtx.Send("Describe the issue in the reply to the next message")
		return tgCtx.Send("support", &telebot.ReplyMarkup{
			ForceReply: true,
		})
	})
	b.Handle("/terms", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.com/tos.html\">terms link</a>", telebot.ModeHTML)
	})
	b.Handle("/privacy", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.com/privacy.html\">privacy link</a>", telebot.ModeHTML)
	})
	b.Handle(telebot.OnCallback, service.ErrorHandlerFunc(service.Callback(callbackHandlers)))
	b.Handle(telebot.OnText, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnPhoto, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnAudio, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnVideo, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnDocument, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnLocation, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers)))
	b.Handle(telebot.OnWebApp, service.ErrorHandlerFunc(service.WebAppData(webappHandlers)))
	b.Handle(telebot.OnCheckout, service.ErrorHandlerFunc(service.PreCheckout(preCheckoutHandlers)))
	b.Handle(telebot.OnPayment, service.ErrorHandlerFunc(service.Payment(paymentHandlers)))
	//
	b.Handle(telebot.OnAddedToGroup, func(tgCtx telebot.Context) error {
		chat := tgCtx.Chat()
		var msg *telebot.Message
		msg, err = b.Forward(chat, donateMsg)
		if err == nil {
			err = b.Pin(msg)
		}
		log.Warn(fmt.Sprintf("Failed to forward or pin the donation invoice in the chat %+v, cause: %s", chat, err))
		return service.ErrorHandlerFunc(subListHandlerFunc)(tgCtx)
	})
	b.Handle(telebot.OnUserLeft, service.ErrorHandlerFunc(chats.UserLeftHandlerFunc(chatStor)))

	go func() {
		var count uint32
		count, err = chats.ResumeAllReaders(ctx, log, chatStor, b, clientAwk, msgFmt, replicaIndex, cfg.Replica.Range)
		log.Info(fmt.Sprintf("Resumed %d readers, errors: %s", count, err))
	}()

	b.Start()
}
