package sources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/api/grpc/source/sites"
	"github.com/awakari/bot-telegram/api/grpc/source/telegram"
	"github.com/awakari/bot-telegram/service/support"
	"github.com/mmcdole/gofeed"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

const srcTypeTgCh = "tgch"
const srcTypeFeed = "feed"
const srcTypeSite = "site"
const feedFetchTimeout = 1 * time.Minute
const srcAddrLenMax = 80
const day = 24 * time.Hour
const updatesPerDayMax = 288

type addPayload struct {
	Limit srcLimit `json:"limit"`
	Src   src      `json:"src"`
}

type srcLimit struct {
	Freq uint16 `json:"freq"`
}

type src struct {
	Addr string `json:"addr"`
	Type string `json:"type"`
}

var errInvalidAddPayload = errors.New("invalid add source payload")

func (ap addPayload) validate(bot *telebot.Bot) (err error) {
	if err == nil && ap.Src.Addr == "" {
		err = fmt.Errorf("%w: empty source address", errInvalidAddPayload)
	}
	if err == nil && len(ap.Src.Addr) > srcAddrLenMax {
		err = fmt.Errorf("%w: source address too long: %s, should not be more than %d", errInvalidAddPayload, ap.Src.Addr, srcAddrLenMax)
	}
	switch ap.Src.Type {
	case srcTypeTgCh:
	case srcTypeSite:
	case srcTypeFeed:
		_, err = gofeed.NewParser().ParseURL(ap.Src.Addr)
	default:
		err = fmt.Errorf("%w: unrecognized source type %s", errInvalidAddPayload, ap.Src.Type)
	}
	return
}

type AddHandler struct {
	SvcFeeds       feeds.Service
	SvcSites       sites.Service
	SvcTg          telegram.Service
	Log            *slog.Logger
	SupportHandler support.Handler
	GroupId        string
}

func (ah AddHandler) HandleFormData(tgCtx telebot.Context, args ...string) (err error) {
	var ap addPayload
	err = json.Unmarshal([]byte(args[0]), &ap)
	if err == nil {
		err = ap.validate(tgCtx.Bot())
	}
	sender := tgCtx.Sender()
	if err == nil {
		err = ah.registerSource(context.TODO(), tgCtx, ap, strconv.FormatInt(sender.ID, 10))
	}
	if err == nil {
		switch ap.Src.Type {
		case srcTypeTgCh:
			err = ah.SupportHandler.Request(tgCtx, fmt.Sprintf("User @%s added a telegram channel: %s", sender.Username, ap.Src.Addr))
		}
	}
	if err == nil {
		err = tgCtx.Send(fmt.Sprintf("Source added successfully: %s", ap.Src.Addr))
	}
	return
}

func (ah AddHandler) registerSource(ctx context.Context, tgCtx telebot.Context, ap addPayload, userId string) (err error) {
	switch ap.Src.Type {
	case srcTypeFeed:
		err = ah.registerFeed(ctx, ap, userId)
	case srcTypeTgCh:
		var chat *telebot.Chat
		chat, err = tgCtx.Bot().ChatByUsername(ap.Src.Addr)
		if err == nil && chat.Type != telebot.ChatChannel {
			err = fmt.Errorf("%w: telegram chat type is %s, should be %s", errInvalidAddPayload, chat.Type, telebot.ChatChannel)
		}
		if err == nil {
			err = ah.registerTelegramChannel(ctx, chat, ap.Src.Addr, userId)
		}
	case srcTypeSite:
		err = ah.registerSite(ctx, ap, userId)
	default:
		err = fmt.Errorf("unrecognized source type: %s", ap.Src.Type)
	}
	return
}

func (ah AddHandler) registerFeed(ctx context.Context, ap addPayload, userId string) (err error) {
	feed := feeds.Feed{
		Url:          ap.Src.Addr,
		GroupId:      ah.GroupId,
		UserId:       userId,
		UpdatePeriod: durationpb.New(day / time.Duration(ap.Limit.Freq)),
		NextUpdate:   timestamppb.New(time.Now().UTC()),
	}
	err = ah.SvcFeeds.Create(ctx, &feed)
	return
}

func (ah AddHandler) registerTelegramChannel(ctx context.Context, chat *telebot.Chat, url, userId string) (err error) {
	ch := telegram.Channel{
		Id:      chat.ID,
		GroupId: ah.GroupId,
		UserId:  userId,
		Name:    chat.Title,
		Link:    url,
	}
	err = ah.SvcTg.Create(ctx, &ch)
	return
}

func (ah AddHandler) registerSite(ctx context.Context, ap addPayload, userId string) (err error) {
	site := sites.Site{
		Addr:    ap.Src.Addr,
		GroupId: ah.GroupId,
		UserId:  userId,
	}
	err = ah.SvcSites.Create(ctx, &site)
	return
}
