package main

import (
	"context"
	"crypto/tls"
	"fmt"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	grpcApiMsgs "github.com/awakari/bot-telegram/api/grpc/messages"
	grpcApiSrcFeeds "github.com/awakari/bot-telegram/api/grpc/source/feeds"
	grpcApiSrcTg "github.com/awakari/bot-telegram/api/grpc/source/telegram"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/bot-telegram/service/sources"
	"github.com/awakari/bot-telegram/service/subscriptions"
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

	// init the source-feeds client
	connSrcFeeds, err := grpc.Dial(cfg.Api.Source.Feeds.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the source-feeds service")
		defer connMsgs.Close()
	} else {
		log.Error("failed to connect the source-feeds service", err)
	}
	clientSrcFeeds := grpcApiSrcFeeds.NewServiceClient(connSrcFeeds)
	svcSrcFeeds := grpcApiSrcFeeds.NewService(clientSrcFeeds)

	// init the source-telegram client
	connSrcTg, err := grpc.Dial(cfg.Api.Source.Telegram.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err == nil {
		log.Info("connected the source-telegram service")
		defer connMsgs.Close()
	} else {
		log.Error("failed to connect the source-telegram service", err)
	}
	clientSrcTg := grpcApiSrcTg.NewServiceClient(connSrcTg)
	svcSrcTg := grpcApiSrcTg.NewService(clientSrcTg)
	svcSrcTg = grpcApiSrcTg.NewServiceLogging(svcSrcTg, log)

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
	menuKbd := service.MakeReplyKeyboard() // main menu keyboard
	supportHandler := service.SupportHandler{
		SupportChatId: cfg.Api.Telegram.SupportChatId,
		RestoreKbd:    menuKbd,
	}
	pricesHandler := service.PricesHandler{
		CfgPayment: cfg.Payment,
		RestoreKbd: menuKbd,
	}
	srcAddHandler := sources.AddHandler{
		SvcFeeds:       svcSrcFeeds,
		SvcTg:          svcSrcTg,
		Log:            log,
		SupportHandler: supportHandler,
		GroupId:        groupId,
	}
	srcListHandler := sources.ListHandler{
		SvcSrcFeeds: svcSrcFeeds,
		SvcSrcTg:    svcSrcTg,
		Log:         log,
		GroupId:     groupId,
	}
	srcDetailsHandler := sources.DetailsHandler{
		CfgFeeds:    cfg.Api.Source.Feeds,
		CfgTelegram: cfg.Api.Source.Telegram,
		ClientAwk:   clientAwk,
		SvcSrcFeeds: svcSrcFeeds,
		SvcSrcTg:    svcSrcTg,
		Log:         log,
		GroupId:     groupId,
	}
	srcDeleteHandler := sources.DeleteHandler{
		SvcSrcFeeds:    svcSrcFeeds,
		SvcSrcTg:       svcSrcTg,
		RestoreKbd:     menuKbd,
		GroupId:        groupId,
		SupportHandler: supportHandler,
	}
	limitsHandler := usage.LimitsHandler{
		CfgPayment:  cfg.Payment,
		ClientAdmin: svcAdmin,
		ClientAwk:   clientAwk,
		GroupId:     groupId,
		Log:         log,
		RestoreKbd:  menuKbd,
	}
	subExtHandler := subscriptions.ExtendHandler{
		CfgPayment: cfg.Payment,
		ClientAwk:  clientAwk,
		GroupId:    groupId,
		Log:        log,
		RestoreKbd: menuKbd,
	}
	pubUsageHandler := messages.UsageHandler{
		ClientAwk: clientAwk,
		GroupId:   groupId,
	}
	subCondHandler := subscriptions.ConditionHandler{
		ClientAwk:  clientAwk,
		GroupId:    groupId,
		RestoreKbd: menuKbd,
	}
	callbackHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.CmdDelete:      subscriptions.DeleteHandlerFunc(),
		subscriptions.CmdDetails:     subscriptions.DetailsHandlerFunc(clientAwk, groupId),
		subscriptions.CmdDescription: subscriptions.DescriptionHandlerFunc(clientAwk, groupId),
		subscriptions.CmdExtend:      subExtHandler.RequestExtensionDaysCount,
		subscriptions.CmdStart:       subscriptions.Start(log, clientAwk, chatStor, groupId, msgFmt),
		subscriptions.CmdStop:        subscriptions.Stop(chatStor),
		subscriptions.CmdPageNext:    subscriptions.PageNext(clientAwk, chatStor, groupId),
		usage.CmdExtend:              limitsHandler.RequestExtension,
		usage.CmdIncrease:            limitsHandler.RequestIncrease,
		sources.CmdTgChListAll:       srcListHandler.TelegramChannelsAll,
		sources.CmdTgChListOwn:       srcListHandler.TelegramChannelsOwn,
		sources.CmdFeedListAll:       srcListHandler.FeedListAll,
		sources.CmdFeedListOwn:       srcListHandler.FeedListOwn,
		sources.CmdFeedDetailsAny:    srcDetailsHandler.GetFeedAny,
		sources.CmdFeedDetailsOwn:    srcDetailsHandler.GetFeedOwn,
		sources.CmdTgChDetails:       srcDetailsHandler.GetTelegramChannel,
		sources.CmdDelete:            srcDeleteHandler.RequestConfirmation,
	}
	webappHandlers := map[string]service.ArgHandlerFunc{
		service.LabelPubMsgCustom:    messages.PublishCustomHandlerFunc(clientAwk, groupId, svcMsgs, cfg.Payment),
		service.LabelSubCreateCustom: subscriptions.CreateCustomHandlerFunc(clientAwk, groupId),
		usage.LabelExtend:            limitsHandler.HandleExtension,
		messages.LabelPubAddSource:   srcAddHandler.HandleFormData,
		subscriptions.LabelCond:      subCondHandler.Update,
	}
	txtHandlers := map[string]telebot.HandlerFunc{
		service.LabelSubList:        subscriptions.ListHandlerFunc(clientAwk, chatStor, groupId),
		service.LabelSubCreateBasic: subscriptions.CreateBasicRequest,
		service.LabelUsageSub:       subscriptions.Usage(clientAwk, groupId),
		service.LabelPublishing:     messages.Details,
		service.LabelPubMsgBasic:    messages.PublishBasicRequest,
		service.LabelUsagePub:       pubUsageHandler.Show,
		service.LabelMainMenu: func(tgCtx telebot.Context) error {
			return tgCtx.Send("Main menu reply keyboard", menuKbd)
		},
	}
	replyHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.ReqDescribe:       subscriptions.DescriptionReplyHandlerFunc(clientAwk, groupId, menuKbd),
		subscriptions.ReqDelete:         subscriptions.DeleteReplyHandlerFunc(clientAwk, groupId, menuKbd),
		subscriptions.ReqSubCreateBasic: subscriptions.CreateBasicReplyHandlerFunc(clientAwk, groupId, menuKbd),
		messages.ReqMsgPubBasic:         messages.PublishBasicReplyHandlerFunc(clientAwk, groupId, svcMsgs, cfg.Payment, menuKbd),
		subscriptions.ReqSubExtend:      subExtHandler.HandleExtensionReply,
		usage.ReqLimitExtend:            limitsHandler.HandleExtension,
		usage.ReqLimitIncrease:          limitsHandler.HandleIncrease,
		sources.CmdDeleteConfirm:        srcDeleteHandler.HandleConfirmation,
		"support":                       supportHandler.Support,
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
			Description: "Start and show main menu",
		},
		{
			Text:        "support",
			Description: "Request support",
		},
		{
			Text:        "prices",
			Description: "Prices information",
		},
		{
			Text:        "help",
			Description: "User guide",
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
			case telebot.ChatGroup:
				err = subListHandlerFunc(tgCtx)
			case telebot.ChatSuperGroup:
				err = subListHandlerFunc(tgCtx)
			case telebot.ChatPrivate:
				err = tgCtx.Send("Main menu reply keyboard", menuKbd)
			default:
				err = fmt.Errorf("unsupported chat type (supported options: \"private\", \"group\", \"supergroup\"): %s", chat.Type)
			}
			return
		}, menuKbd),
	)
	b.Handle("/help", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.app/help.html\">help link</a>", telebot.ModeHTML)
	})
	b.Handle("/terms", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.app/tos.html\">terms link</a>", telebot.ModeHTML)
	})
	b.Handle("/privacy", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.app/privacy.html\">privacy link</a>", telebot.ModeHTML)
	})
	b.Handle("/support", func(tgCtx telebot.Context) error {
		_ = tgCtx.Send("Describe the issue in the reply to the next message")
		return tgCtx.Send("support", &telebot.ReplyMarkup{
			ForceReply: true,
		})
	})
	b.Handle("/prices", pricesHandler.Prices)
	b.Handle(telebot.OnCallback, service.ErrorHandlerFunc(service.Callback(callbackHandlers), menuKbd))
	b.Handle(telebot.OnText, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnPhoto, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnAudio, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnVideo, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnDocument, service.ErrorHandlerFunc(service.RootHandlerFunc(txtHandlers, replyHandlers), menuKbd))
	b.Handle(telebot.OnWebApp, service.ErrorHandlerFunc(service.WebAppData(webappHandlers), menuKbd))
	b.Handle(telebot.OnCheckout, service.ErrorHandlerFunc(service.PreCheckout(preCheckoutHandlers), menuKbd))
	b.Handle(telebot.OnPayment, service.ErrorHandlerFunc(service.Payment(paymentHandlers), menuKbd))
	//
	b.Handle(telebot.OnAddedToGroup, service.ErrorHandlerFunc(subListHandlerFunc, nil))
	b.Handle(telebot.OnUserLeft, service.ErrorHandlerFunc(chats.UserLeftHandlerFunc(chatStor), nil))

	go func() {
		var count uint32
		count, err = chats.ResumeAllReaders(ctx, log, chatStor, b, clientAwk, msgFmt, replicaIndex, cfg.Replica.Range)
		log.Info(fmt.Sprintf("Resumed %d readers, errors: %s", count, err))
	}()

	b.Start()
}
