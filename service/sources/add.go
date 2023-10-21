package sources

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	grpcApiAdmin "github.com/awakari/bot-telegram/api/grpc/admin"
	grpcApiSrcFeeds "github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/model/usage"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/telebot.v3"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const srcTypeTgCh = "tgch"
const srcTypeFeed = "feed"
const limitCountMax = 1_440
const daysMax = 3_652
const priceMax = 10_000
const feedFetchTimeout = 1 * time.Minute
const PurposeSrcAdd = "src_add"
const srcAddrLenMax = 80
const orderPayloadSep = ","
const day = 24 * time.Hour
const updatesPerDayMax = 1_440
const msgFmtRunOnceFailed = "failed to add source, cause: %s, retrying in: %s"

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

func (ap addPayload) validate(cfgPayment config.PaymentConfig, bot *telebot.Bot) (err error) {
	if err == nil && (ap.Limit.Count < 1 || ap.Limit.Count > limitCountMax) {
		err = fmt.Errorf("%w: count limit is %d, should in the range of 1..%d", errInvalidAddPayload, ap.Limit.Count, limitCountMax)
	}
	if err == nil && (ap.Limit.Time < 1 || ap.Limit.Count > daysMax) {
		err = fmt.Errorf("%w: time in days is %d, should in the range of 1..%d", errInvalidAddPayload, ap.Limit.Time, daysMax)
	}
	if err == nil && ((ap.Price.Total < 1 && ap.Price.Total != 0) || ap.Price.Total > 10_000) {
		err = fmt.Errorf("%w: total price is %f, should in the range of 1..%d", errInvalidAddPayload, ap.Price.Total, priceMax)
	}
	if err == nil && ap.Price.Unit != cfgPayment.Currency.Code {
		err = fmt.Errorf("%w: currency is %s, should be %s", errInvalidAddPayload, ap.Price.Unit, cfgPayment.Currency.Code)
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
	CfgPayment config.PaymentConfig
	CfgFeeds   config.FeedsConfig
	KbdRestore *telebot.ReplyMarkup
	SvcFeeds   grpcApiSrcFeeds.Service
	SvcAdmin   grpcApiAdmin.Service
	Log        *slog.Logger
}

func (ah AddHandler) HandleFormData(tgCtx telebot.Context, args ...string) (err error) {
	var ap addPayload
	err = json.Unmarshal([]byte(args[0]), &ap)
	if err == nil {
		err = ap.validate(ah.CfgPayment, tgCtx.Bot())
	}
	switch ap.Price.Total {
	case 0:
		// add for free, don't change a limit
		err = ah.registerSource(context.TODO(), ap, strconv.FormatInt(tgCtx.Sender().ID, 10))
		if err == nil {
			err = tgCtx.Send(fmt.Sprintf("Source added successfully: %s", ap.Src.Addr))
		}
	default:
		err = ah.sendInvoice(tgCtx, ap)
	}
	return
}

func (ah AddHandler) sendInvoice(tgCtx telebot.Context, ap addPayload) (err error) {
	var orderData []byte
	if err == nil {
		o := service.Order{
			Purpose: PurposeSrcAdd,
			Payload: fmt.Sprintf(
				"%d%s%d%s%d%s%s%s%s",
				ap.Limit.Count,
				orderPayloadSep,
				ap.Limit.Freq,
				orderPayloadSep,
				ap.Limit.Time,
				orderPayloadSep,
				ap.Src.Addr,
				orderPayloadSep,
				ap.Src.Type,
			),
		}
		orderData, err = json.Marshal(o)
	}
	if err == nil {
		label := fmt.Sprintf("Source: %s", ap.Src.Addr)
		price := int(ap.Price.Total * ah.CfgPayment.Currency.SubFactor)
		invoice := telebot.Invoice{
			Start:       uuid.NewString(),
			Title:       fmt.Sprintf("Add custom source for %d days", ap.Limit.Time),
			Description: label,
			Payload:     string(orderData),
			Currency:    ah.CfgPayment.Currency.Code,
			Prices: []telebot.Price{
				{
					Label:  label,
					Amount: price,
				},
			},
			Token: ah.CfgPayment.Provider.Token,
			Total: price,
		}
		err = tgCtx.Send("To proceed, please pay the below invoice", ah.KbdRestore)
		_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
	}
	return err
}

func (ah AddHandler) registerSource(ctx context.Context, ap addPayload, userId string) (err error) {
	addr := ap.Src.Addr
	expires := time.Now().UTC().Add(day * time.Duration(ap.Limit.Time))
	switch ap.Src.Type {
	case srcTypeFeed:
		feed := grpcApiSrcFeeds.Feed{
			Url:          addr,
			UserId:       userId,
			UpdatePeriod: durationpb.New(day / time.Duration(ap.Limit.Freq)),
			NextUpdate:   timestamppb.New(time.Now().UTC()),
			Expires:      timestamppb.New(expires),
		}
		err = ah.SvcFeeds.Write(ctx, &feed)
	default:
		err = fmt.Errorf("%w: unsupported source type: %s", errInvalidAddPayload, ap.Src.Type)
	}
	return
}

func (ah AddHandler) AddPrecheckout(tgCtx telebot.Context, args ...string) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), ah.CfgPayment.PreCheckout.Timeout)
	defer cancel()
	orderPayloadParts := strings.Split(args[0], orderPayloadSep)
	if len(orderPayloadParts) != 5 {
		err = fmt.Errorf("%w: %s", errInvalidAddPayload, args[0])
	}
	var srcAddr string
	var srcType string
	if err == nil {
		srcAddr = orderPayloadParts[3]
		srcType = orderPayloadParts[4]
	}
	switch srcType {
	case srcTypeFeed:
		var feed *grpcApiSrcFeeds.Feed
		feed, err = ah.SvcFeeds.Read(ctx, srcAddr)
		switch {
		case err == nil:
			if feed.Expires != nil && feed.Expires.AsTime().After(time.Now().UTC()) {
				err = errors.New(fmt.Sprintf("cannot add the source with the same address, it already exists and not expired yet: %s", srcAddr))
			}
		case status.Code(err) == codes.NotFound:
			err = nil
		}
		switch err {
		case nil:
			err = tgCtx.Accept()
		default:
			err = tgCtx.Accept(err.Error())
		}
	default:
		err = tgCtx.Accept(fmt.Sprintf("unsupported source type: %s", srcType))
	}
	return
}

func (ah AddHandler) AddPaid(tgCtx telebot.Context, args ...string) (err error) {
	orderPayloadParts := strings.Split(args[0], orderPayloadSep)
	if len(orderPayloadParts) != 5 {
		err = fmt.Errorf("%w: %s", errInvalidAddPayload, args[0])
	}
	var lc uint64
	if err == nil {
		lc, err = strconv.ParseUint(orderPayloadParts[0], 10, 16)
	}
	var lf uint64
	if err == nil {
		lf, err = strconv.ParseUint(orderPayloadParts[1], 10, 16)
	}
	var lt uint64
	if err == nil {
		lt, err = strconv.ParseUint(orderPayloadParts[2], 10, 16)
	}
	var ap addPayload
	if err == nil {
		ap.Limit.Count = uint16(lc)
		ap.Limit.Freq = uint16(lf)
		ap.Limit.Time = uint16(lt)
		ap.Src.Addr = orderPayloadParts[3]
		ap.Src.Type = orderPayloadParts[4]
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = ah.CfgPayment.Backoff.Init
		b.Multiplier = ah.CfgPayment.Backoff.Factor
		b.MaxElapsedTime = ah.CfgPayment.Backoff.LimitTotal
		action := func() error {
			return ah.registerPaidSource(ap, userId)
		}
		err = backoff.RetryNotify(action, b, func(err error, d time.Duration) {
			ah.Log.Warn(fmt.Sprintf(msgFmtRunOnceFailed, err, d))
			if d > 1*time.Second {
				_ = tgCtx.Send("adding the source, please wait...")
			}
		})
		switch err {
		case nil:
			err = tgCtx.Send(fmt.Sprintf("Source added successfully: %s", ap.Src.Addr))
		default:
			err = tgCtx.Send(fmt.Sprintf("Failed to add the source: %s, cause: %s", ap.Src.Addr, err))
		}
	}
	return
}

func (ah AddHandler) registerPaidSource(ap addPayload, userId string) (err error) {
	ctx := context.TODO()
	err = ah.registerSource(ctx, ap, userId)
	if err == nil {
		expires := time.Now().UTC().Add(day * time.Duration(ap.Limit.Time))
		switch ap.Src.Type {
		case srcTypeFeed:
			err = ah.SvcAdmin.SetLimits(
				ctx,
				ah.CfgFeeds.GroupId,
				ap.Src.Addr,
				usage.SubjectPublishEvents,
				int64(ap.Limit.Count),
				expires,
			)
		default:
			err = errors.New(fmt.Sprintf("unsupported source type: %s", ap.Src.Type))
		}
	}
	return
}
