package main

import (
	"context"
	"crypto/tls"
	"fmt"
	apiGrpc "github.com/awakari/bot-telegram/api/grpc"
	"github.com/awakari/bot-telegram/api/grpc/queue"
	apiGrpcTgBot "github.com/awakari/bot-telegram/api/grpc/tgbot"
	apiGrpcUsageLimits "github.com/awakari/bot-telegram/api/grpc/usage/limits"
	"github.com/awakari/bot-telegram/api/http/interests"
	"github.com/awakari/bot-telegram/api/http/pub"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/limits"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/bot-telegram/service/subscriptions"
	"github.com/awakari/bot-telegram/service/support"
	"github.com/awakari/bot-telegram/util"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/gin-gonic/gin"
	"github.com/microcosm-cc/bluemonday"
	grpcpool "github.com/processout/grpc-go-pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {

	// init config and logger
	slog.Info("starting...")
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to load the config: %s", err))
	}
	opts := slog.HandlerOptions{
		Level: slog.Level(cfg.Log.Level),
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &opts))

	svcPub := pub.NewService(http.DefaultClient, cfg.Api.Writer.Uri, cfg.Api.Token.Internal)
	svcPub = pub.NewLogging(svcPub, log)
	log.Info("initialized the Awakari publish API client")

	svcInterests := interests.NewService(http.DefaultClient, cfg.Api.Interests.Uri, cfg.Api.Token.Internal)
	svcInterests = interests.NewLogging(svcInterests, log)
	log.Info("initialized the Awakari interests API client")

	// init websub reader
	clientHttp := http.Client{}
	svcReader := reader.NewService(&clientHttp, cfg.Api.Reader.Uri, cfg.Api.Token.Internal)
	svcReader = reader.NewServiceLogging(svcReader, log)
	urlCallbackBase := fmt.Sprintf(
		"%s://%s:%d%s",
		cfg.Api.Reader.CallBack.Protocol,
		cfg.Api.Reader.CallBack.Host,
		cfg.Api.Reader.CallBack.Port,
		cfg.Api.Reader.CallBack.Path,
	)

	// init queues
	connQueue, err := grpc.NewClient(cfg.Api.Queue.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	log.Info("connected to the queue service")
	clientQueue := queue.NewServiceClient(connQueue)
	svcQueue := queue.NewService(clientQueue)
	svcQueue = queue.NewLoggingMiddleware(svcQueue, log)
	err = svcQueue.SetConsumer(context.TODO(), cfg.Api.Queue.InterestsCreated.Name, cfg.Api.Queue.InterestsCreated.Subj)
	if err != nil {
		panic(err)
	}
	log.Info(fmt.Sprintf("initialized the %s queue", cfg.Api.Queue.InterestsCreated.Name))
	go func() {
		err = consumeQueueInterestsCreated(
			context.Background(),
			svcReader,
			urlCallbackBase,
			cfg.Api.GroupId,
			svcQueue,
			cfg.Api.Queue.InterestsCreated.Name,
			cfg.Api.Queue.InterestsCreated.Subj,
			cfg.Api.Queue.InterestsCreated.BatchSize,
		)
		if err != nil {
			panic(err)
		}
	}()

	connPoolLimits, err := grpcpool.New(
		func() (*grpc.ClientConn, error) {
			return grpc.NewClient(cfg.Api.Usage.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		int(cfg.Api.Usage.Connection.Count.Init),
		int(cfg.Api.Usage.Connection.Count.Max),
		cfg.Api.Usage.Connection.IdleTimeout,
	)
	if err != nil {
		panic(err)
	}
	defer connPoolLimits.Close()
	clientLimits := apiGrpcUsageLimits.NewClientPool(connPoolLimits)
	svcLimits := limits.NewService(clientLimits)
	svcLimits = limits.NewLogging(svcLimits, log)

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
		HtmlPolicy:       htmlPolicy,
		UriReaderEvtBase: cfg.Api.Reader.UriEventBase,
	}

	// init handlers
	groupId := cfg.Api.GroupId
	supportHandler := support.Handler{
		SupportChatId: cfg.Api.Telegram.SupportChatId,
	}
	chanPostHandler := messages.ChanPostHandler{
		SvcPub:    svcPub,
		GroupId:   groupId,
		Log:       log,
		Channels:  map[string]time.Time{},
		ChansLock: &sync.Mutex{},
		CfgMsgs:   cfg.Api.Messages,
	}

	callbackHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.CmdStart:             subscriptions.StartHandler(svcInterests, svcReader, svcLimits, urlCallbackBase, groupId),
		subscriptions.CmdStop:              subscriptions.Stop(svcReader, urlCallbackBase, cfg.Api.GroupId),
		subscriptions.CmdPageNext:          subscriptions.PageNext(svcInterests, svcReader, groupId, urlCallbackBase),
		subscriptions.CmdPageNextFollowing: subscriptions.PageNextFollowing(svcInterests, svcReader, groupId, urlCallbackBase),
	}
	replyHandlers := map[string]service.ArgHandlerFunc{
		subscriptions.ReqSubCreate: subscriptions.CreateBasicReplyHandlerFunc(svcInterests, groupId),
		subscriptions.ReqStart:     subscriptions.StartHandler(svcInterests, svcReader, svcLimits, urlCallbackBase, groupId),
		messages.ReqMsgPub:         messages.PublishBasicReplyHandlerFunc(svcPub, groupId, cfg),
		"support":                  supportHandler.Request,
	}
	txtHandlers := map[string]telebot.HandlerFunc{}
	hRoot := service.RootHandler{
		ReplyHandlers: replyHandlers,
		TxtHandlers:   txtHandlers,
	}

	hPaid := service.PaidChatMemberHandler{
		GroupId:                      groupId,
		LimitByChatIdSubscriptions:   cfg.Api.Usage.Limits.Subscriptions,
		LimitByChatIdInterests:       cfg.Api.Usage.Limits.Interests,
		LimitByChatIdInterestsPublic: cfg.Api.Usage.Limits.InterestsPublic,
		SvcLimits:                    svcLimits,
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
			AllowedUpdates: []string{
				"callback_query",
				"channel_post",
				"chat_member",
				"chosen_inline_result",
				"inline_query",
				"message",
				"poll",
			},
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
			Description: "Start: list own interests",
		},
		{
			Text:        "app",
			Description: "Go to application",
		},
		{
			Text:        "pub",
			Description: "Publish a simple message",
		},
		{
			Text:        "sub",
			Description: "Create a simple interest and subscribe",
		},
		{
			Text:        "following",
			Description: "List subscriptions in this chat",
		},
		{
			Text:        "interests",
			Description: "List all available interests",
		},
		{
			Text:        "donate",
			Description: "Donate",
		},
		{
			Text:        "help",
			Description: "Help",
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

	// init the Telegram Bot grpc service
	controllerGrpc := apiGrpcTgBot.NewController(
		[]byte(cfg.Api.Telegram.Token),
		chanPostHandler,
		svcReader,
		urlCallbackBase,
		log,
		b,
		fmtMsg,
	)
	go func() {
		log.Info(fmt.Sprintf("starting to listen the grpc API @ port #%d...", cfg.Api.Telegram.Bot.Port))
		err = apiGrpc.Serve(cfg.Api.Telegram.Bot.Port, controllerGrpc)
		if err != nil {
			panic(err)
		}
	}()

	// assign handlers
	b.Use(func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return service.LoggingHandlerFunc(next, log)
	})
	subListHandlerFunc := subscriptions.ListOnGroupStartHandlerFunc(svcInterests, svcReader, groupId, urlCallbackBase)
	b.Handle(
		"/start",
		service.ErrorHandlerFunc(func(tgCtx telebot.Context) (err error) {
			cmdTxt := tgCtx.Text()
			if strings.HasPrefix(cmdTxt, "/start ") && len(cmdTxt) > len("/start ") {
				arg := cmdTxt[len("/start "):]
				err = subscriptions.StartIntervalRequest(tgCtx, arg)
			} else {
				chat := tgCtx.Chat()
				switch chat.Type {
				case telebot.ChatChannel:
				case telebot.ChatChannelPrivate:
				case telebot.ChatGroup:
					err = subListHandlerFunc(tgCtx)
				case telebot.ChatSuperGroup:
					err = subListHandlerFunc(tgCtx)
				case telebot.ChatPrivate:
					// err = service.DonationMessagePin(tgCtx)
					err = subListHandlerFunc(tgCtx)
				default:
					err = fmt.Errorf("unsupported chat type (supported options: \"private\", \"group\", \"supergroup\"): %s", chat.Type)
				}
			}
			return
		}),
	)
	b.Handle("/app", func(tgCtx telebot.Context) error {
		return tgCtx.Send("<a href=\"https://awakari.com/login.html\">Link to App</a>", telebot.ModeHTML)
	})
	b.Handle("/pub", messages.PublishBasicRequest)
	b.Handle("/sub", subscriptions.CreateBasicRequest)
	b.Handle("/following", subscriptions.ListFollowing(svcInterests, svcReader, groupId, urlCallbackBase))
	b.Handle("/interests", subscriptions.ListPublicHandlerFunc(svcInterests, svcReader, groupId, urlCallbackBase))
	b.Handle("/donate", service.DonationHandler)
	b.Handle("/help", func(tgCtx telebot.Context) error {
		return tgCtx.Send("Open the <a href=\"https://awakari.com/#resources\">link</a>", telebot.ModeHTML)
	})
	b.Handle("/support", func(tgCtx telebot.Context) error {
		_ = tgCtx.Send("Describe your issue in the reply to the next message")
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
	b.Handle(telebot.OnText, service.ErrorHandlerFunc(hRoot.Handle))
	b.Handle(telebot.OnPhoto, service.ErrorHandlerFunc(hRoot.Handle))
	b.Handle(telebot.OnAudio, service.ErrorHandlerFunc(hRoot.Handle))
	b.Handle(telebot.OnVideo, service.ErrorHandlerFunc(hRoot.Handle))
	b.Handle(telebot.OnDocument, service.ErrorHandlerFunc(hRoot.Handle))
	b.Handle(telebot.OnLocation, service.ErrorHandlerFunc(hRoot.Handle))
	//
	b.Handle(telebot.OnChannelPost, func(tgCtx telebot.Context) (err error) {
		txt := tgCtx.Text()
		ch := tgCtx.Chat()
		chanUserName := ch.Username
		if strings.HasPrefix(chanUserName, cfg.Api.Telegram.PublicInterestChannelPrefix) && strings.HasPrefix(txt, "/start ") {
			// public interest channel created by Awakari
			arg := txt[len("/start "):]
			err = subscriptions.StartIntervalRequest(tgCtx, arg)
		} else {
			err = chanPostHandler.Publish(tgCtx, chanUserName)
		}
		return
	})
	b.Handle(telebot.OnAddedToGroup, func(tgCtx telebot.Context) error {
		// err = service.DonationMessagePin(tgCtx)
		return service.ErrorHandlerFunc(subListHandlerFunc)(tgCtx)
	})
	b.Handle(telebot.OnChatMember, func(tgCtx telebot.Context) error {
		err = hPaid.Handle(tgCtx)
		ll := util.LogLevel(err)
		log.Log(context.TODO(), ll, fmt.Sprintf("PaidChatMemberHandler.Handle(): %s", err))
		return err
	})
	//
	go b.Start()

	// chats websub handler (subscriber)
	hChats := chats.NewHandler(cfg.Api.Reader.Uri+"/v1", fmtMsg, urlCallbackBase, svcReader, b, svcInterests, groupId)
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

func consumeQueueInterestsCreated(
	ctx context.Context,
	svcReader reader.Service,
	urlCallbackBase string,
	groupId string,
	svcQueue queue.Service,
	name, subj string,
	batchSize uint32,
) (err error) {
	consume := func(evts []*pb.CloudEvent) (err error) {
		// commented because the bot should not consume the user's subscriptions permits anymore
		//for _, evt := range evts {
		//    interestId := evt.GetTextData()
		//    var userId string
		//    if userIdAttr, userIdPresent := evt.Attributes["awakariuserid"]; userIdPresent {
		//        userId = userIdAttr.GetCeString()
		//    }
		//    if !strings.HasPrefix(userId, util.PrefixUserId) {
		//        continue
		//    }
		//    var chatId int64
		//    if err == nil {
		//        chatId, err = strconv.ParseInt(userId[len(util.PrefixUserId):], 10, 64)
		//        if err != nil {
		//            err = status.Error(codes.InvalidArgument, fmt.Sprintf("User id should end with numeric id: %s, %s", userId, err))
		//        }
		//    }
		//    if err == nil {
		//        err = svcReader.Subscribe(ctx, interestId, groupId, userId, reader.MakeCallbackUrl(urlCallbackBase, chatId, userId), 0)
		//    }
		//    if err != nil {
		//        break
		//    }
		//}
		return
	}
	for {
		err = svcQueue.ReceiveMessages(ctx, name, subj, batchSize, consume)
		if err != nil {
			break
		}
	}
	return
}
