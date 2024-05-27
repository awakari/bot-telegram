package main

import (
	"crypto/tls"
	"fmt"
	grpcApi "github.com/awakari/bot-telegram/api/grpc"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	grpcApiMsgs "github.com/awakari/bot-telegram/api/grpc/messages"
	grpcApiTgBot "github.com/awakari/bot-telegram/api/grpc/tgbot"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/bot-telegram/service/subscriptions"
	"github.com/awakari/bot-telegram/service/support"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
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

	// init websub reader
	clientHttp := http.Client{}
	svcReader := reader.NewService(&clientHttp, cfg.Api.Reader.Uri)
	svcReader = reader.NewServiceLogging(svcReader, log)
	urlCallbackBase := fmt.Sprintf(
		"%s://%s:%d%s",
		cfg.Api.Reader.CallBack.Protocol,
		cfg.Api.Reader.CallBack.Host,
		cfg.Api.Reader.CallBack.Port,
		cfg.Api.Reader.CallBack.Path,
	)

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
	fmtMsg := messages.Format{
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
	chanPostHandler := messages.ChanPostHandler{
		ClientAwk: clientAwk,
		GroupId:   groupId,
		Log:       log,
		Writers:   map[string]model.Writer[*pb.CloudEvent]{},
		Channels:  map[string]time.Time{},
		ChansLock: &sync.Mutex{},
	}
	defer chanPostHandler.Close()

	callbackHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.CmdDescription: subscriptions.DescriptionHandlerFunc(clientAwk, groupId),
		subscriptions.CmdExtend:      subExtHandler.RequestExtensionDaysCount,
		subscriptions.CmdStart:       subscriptions.Start(clientAwk, svcReader, urlCallbackBase, groupId),
		subscriptions.CmdStop:        subscriptions.Stop(svcReader),
		subscriptions.CmdPageNext:    subscriptions.PageNext(clientAwk, svcReader, groupId),
		usage.CmdExtend:              limitsHandler.RequestExtension,
		usage.CmdIncrease:            limitsHandler.RequestIncrease,
	}
	webappHandlers := map[string]service.ArgHandlerFunc{
		usage.LabelExtend: limitsHandler.HandleExtension,
	}
	txtHandlers := map[string]telebot.HandlerFunc{}
	replyHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.ReqDescribe:  subscriptions.DescriptionReplyHandlerFunc(clientAwk, groupId),
		subscriptions.ReqSubCreate: subscriptions.CreateBasicReplyHandlerFunc(clientAwk, groupId, svcReader, urlCallbackBase),
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
			},
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
			Text:        "app",
			Description: "Manage Subscriptions and Publishing in Application",
		},
		{
			Text:        "pub",
			Description: "Publish a basic Message",
		},
		{
			Text:        "sub",
			Description: "Subscribe for Keywords",
		},
		{
			Text:        "donate",
			Description: "Help Awakari to be Free",
		},
		{
			Text:        "help",
			Description: "User Guide",
		},
		{
			Text:        "support",
			Description: "Request Support",
		},
		{
			Text:        "terms",
			Description: "Terms of Service",
		},
		{
			Text:        "privacy",
			Description: "Privacy Policy",
		},
	})
	if err != nil {
		panic(err)
	}

	// init the Telegram Bot grpc service
	controllerGrpc := grpcApiTgBot.NewController(
		[]byte(cfg.Api.Telegram.Token),
		chanPostHandler,
		svcReader,
		urlCallbackBase,
		log,
		clientAwk,
		b,
		fmtMsg,
	)
	go func() {
		log.Info(fmt.Sprintf("starting to listen the grpc API @ port #%d...", cfg.Api.Telegram.Bot.Port))
		err = grpcApi.Serve(cfg.Api.Telegram.Bot.Port, controllerGrpc)
		if err != nil {
			panic(err)
		}
	}()

	// assign handlers
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return service.LoggingHandlerFunc(next, log)
	})
	subListHandlerFunc := subscriptions.ListOnGroupStartHandlerFunc(clientAwk, svcReader, groupId)
	b.Handle(
		"/start",
		service.ErrorHandlerFunc(func(tgCtx telebot.Context) (err error) {
			chat := tgCtx.Chat()
			switch chat.Type {
			case telebot.ChatChannel:
			case telebot.ChatChannelPrivate:
			case telebot.ChatGroup:
				err = subListHandlerFunc(tgCtx)
			case telebot.ChatSuperGroup:
				err = subListHandlerFunc(tgCtx)
			case telebot.ChatPrivate:
				err = service.DonationMessagePin(tgCtx)
				err = subListHandlerFunc(tgCtx)
			default:
				err = fmt.Errorf("unsupported chat type (supported options: \"private\", \"group\", \"supergroup\"): %s", chat.Type)
			}
			return
		}),
	)
	b.Handle("/app", func(tgCtx telebot.Context) error {
		return tgCtx.Send("<a href=\"https://awakari.com/login.html\">Link to App</a>", telebot.ModeHTML)
	})
	b.Handle("/pub", messages.PublishBasicRequest)
	b.Handle("/sub", subscriptions.CreateBasicRequest)
	b.Handle("/donate", service.DonationHandler)
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
	b.Handle(telebot.OnChannelPost, func(tgCtx telebot.Context) error {
		return chanPostHandler.Publish(tgCtx)
	})
	b.Handle(telebot.OnAddedToGroup, func(tgCtx telebot.Context) error {
		err = service.DonationMessagePin(tgCtx)
		return service.ErrorHandlerFunc(subListHandlerFunc)(tgCtx)
	})
	//
	go b.Start()

	// chats websub handler (subscriber)
	hChats := chats.NewHandler(cfg.Api.Reader.Uri, fmtMsg, urlCallbackBase, svcReader, b)
	r := gin.Default()
	r.
		Group(cfg.Api.Reader.CallBack.Path).
		GET("/:chatId", hChats.Confirm).
		POST("/:chatId", hChats.DeliverMessages)
	err = r.Run(fmt.Sprintf(":%d", cfg.Api.Reader.CallBack.Port))
	if err != nil {
		panic(err)
	}
}
