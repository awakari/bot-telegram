package sources

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	grpcApiSrcFeeds "github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/service"
	"github.com/mmcdole/gofeed"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

const srcTypeTgCh = "tgch"
const srcTypeFeed = "feed"
const limitCountMax = 1_440
const daysMax = 3_652
const priceMax = 10_000
const feedFetchTimeout = 1 * time.Minute
const srcAddrLenMax = 80
const day = 24 * time.Hour
const updatesPerDayMax = 1_440

type addPayload struct {
	Limit srcLimit `json:"limit"`
	Price srcPrice `json:"price"`
	Src   src      `json:"src"`
}

type srcLimit struct {
	Count uint16 `json:"count"`
	Freq  uint16 `json:"freq"`
	Time  uint16 `json:"time"`
}

type srcPrice struct {
	Total float64 `json:"total"`
	Unit  string  `json:"unit"`
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
		var chat *telebot.Chat
		chat, err = bot.ChatByUsername(ap.Src.Addr)
		if err == nil && chat.Type != telebot.ChatChannel {
			err = fmt.Errorf("%w: telegram chat type is %s, should be %s", errInvalidAddPayload, chat.Type, telebot.ChatChannel)
		}
	case srcTypeFeed:
		if ap.Limit.Freq < 1 || ap.Limit.Freq > updatesPerDayMax {
			err = fmt.Errorf("%w: source fetch daily frequency is %d, should be [1..%d]", errInvalidAddPayload, ap.Limit.Freq, updatesPerDayMax)
		}
		var resp *http.Response
		if err == nil {
			clientHttp := http.Client{
				Timeout: feedFetchTimeout,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
			resp, err = clientHttp.Get(ap.Src.Addr)
		}
		if err == nil {
			defer resp.Body.Close()
			_, err = gofeed.NewParser().Parse(resp.Body)
		}
	default:
		err = fmt.Errorf("%w: unrecognized source type %s", errInvalidAddPayload, ap.Src.Type)
	}
	return
}

type AddHandler struct {
	SvcFeeds       grpcApiSrcFeeds.Service
	Log            *slog.Logger
	SupportHandler service.SupportHandler
}

func (ah AddHandler) HandleFormData(tgCtx telebot.Context, args ...string) (err error) {
	var ap addPayload
	err = json.Unmarshal([]byte(args[0]), &ap)
	if err == nil {
		err = ap.validate(tgCtx.Bot())
	}
	if err == nil {
		// add for free, don't change a limit
		switch ap.Src.Type {
		case srcTypeTgCh:
			err = ah.SupportHandler.Support(tgCtx, fmt.Sprintf("Request to add source telegram channel:\n%+v", ap.Src.Addr))
		default:
			_, err = ah.registerSource(context.TODO(), ap, strconv.FormatInt(tgCtx.Sender().ID, 10))
			if err == nil {
				err = tgCtx.Send(fmt.Sprintf("Source added successfully: %s", ap.Src.Addr))
			}
		}
	}
	return
}

func (ah AddHandler) registerSource(ctx context.Context, ap addPayload, userId string) (expires time.Time, err error) {
	addr := ap.Src.Addr
	expires = time.Now().UTC().Add(day * time.Duration(ap.Limit.Time))
	switch ap.Src.Type {
	case srcTypeFeed:
		feed := grpcApiSrcFeeds.Feed{
			Url:          addr,
			UserId:       userId,
			UpdatePeriod: durationpb.New(day / time.Duration(ap.Limit.Freq)),
			NextUpdate:   timestamppb.New(time.Now().UTC()),
		}
		err = ah.SvcFeeds.Create(ctx, &feed)
	default:
		err = fmt.Errorf("%w: unsupported source type: %s", errInvalidAddPayload, ap.Src.Type)
	}
	return
}
