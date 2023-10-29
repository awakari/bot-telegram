package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/admin"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

const ExpiresDefaultDays = 30

const LabelIncrease = "▲ Increase Limit"
const CmdIncrease = "lim_incr"
const ReqLimitIncrease = "lim_incr_req"

const LabelExtend = "▲ Extend Time"
const CmdExtend = "lim_extend"
const ReqLimitExtend = "lim_extend_req"

const PurposeLimitSet = "lim_set"

const msgFmtUsageLimit = `%s Usage:<pre>
  Count:   %d
  Limit:   %d
  Expires: %s
</pre>`
const msgFmtRunOnceFailed = "failed to set limit, user id: %s, cause: %s, retrying in: %s"

type LimitsHandler struct {
	CfgPayment  config.PaymentConfig
	ClientAdmin admin.Service
	ClientAwk   api.Client
	GroupId     string
	Log         *slog.Logger
}

func (lh LimitsHandler) RequestExtension(tgCtx telebot.Context, args ...string) (err error) {
	var subjCode int64
	subjCode, err = strconv.ParseInt(args[0], 10, strconv.IntSize)
	if err == nil {
		err = tgCtx.Send("Reply with the count of days to add:")
	}
	if err == nil {
		err = tgCtx.Send(
			fmt.Sprintf("%s %d", ReqLimitExtend, subjCode),
			&telebot.ReplyMarkup{
				ForceReply:  true,
				Placeholder: "30",
			},
		)
	}
	return
}

func (lh LimitsHandler) HandleExtension(tgCtx telebot.Context, args ...string) (err error) {
	var subjCode int64
	subjCode, err = strconv.ParseInt(args[1], 10, strconv.IntSize)
	var subj usage.Subject
	var daysAdd int64
	if err == nil {
		subj = usage.Subject(subjCode)
		daysAdd, err = strconv.ParseInt(args[2], 10, strconv.IntSize)
	}
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	var l usage.Limit
	ctxGroupId := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", lh.GroupId)
	l, err = lh.ClientAwk.ReadUsageLimit(ctxGroupId, userId, subj)
	var priceTotal float64
	if err == nil {
		var pricePerItem float64
		switch subj {
		case usage.SubjectSubscriptions:
			pricePerItem = lh.CfgPayment.Price.Subscription.CountLimit
			priceTotal = pricePerItem * float64(daysAdd*(l.Count-1))
		case usage.SubjectPublishEvents:
			pricePerItem = lh.CfgPayment.Price.MessagePublishing.DailyLimit
			priceTotal = pricePerItem * float64(daysAdd*(l.Count-10))
		}
		if priceTotal <= 1 {
			err = fmt.Errorf("%w: total price too low: %s %f", errInvalidOrder, lh.CfgPayment.Currency.Code, priceTotal)
		}
	}
	var ol OrderLimit
	if err == nil {
		ol.Expires = l.Expires.Add(time.Duration(daysAdd) * time.Hour * 24).UTC()
		ol.Count = uint32(l.Count)
		ol.Subject = usage.Subject(subjCode)
		err = ol.validate()
	}
	var orderPayloadData []byte
	if err == nil {
		orderPayloadData, err = json.Marshal(ol)
	}
	var orderData []byte
	if err == nil {
		o := service.Order{
			Purpose: PurposeLimitSet,
			Payload: string(orderPayloadData),
		}
		orderData, err = json.Marshal(o)
	}
	label := fmt.Sprintf("%d %s: add %d days", l.Count, formatUsageSubject(subj), daysAdd)
	if err == nil {
		price := int(priceTotal * lh.CfgPayment.Currency.SubFactor)
		invoice := telebot.Invoice{
			Start:       uuid.NewString(),
			Title:       fmt.Sprintf("%s new expiration time: %s", formatUsageSubject(subj), ol.Expires.Format(time.RFC3339)),
			Description: label,
			Payload:     string(orderData),
			Currency:    lh.CfgPayment.Currency.Code,
			Prices: []telebot.Price{
				{
					Label:  label,
					Amount: price,
				},
			},
			Token: lh.CfgPayment.Provider.Token,
			Total: price,
		}
		_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
	}
	return
}

func (lh LimitsHandler) PreCheckout(tgCtx telebot.Context, args ...string) (err error) {
	err = tgCtx.Accept()
	return
}

func (lh LimitsHandler) HandleLimitOrderPaid(tgCtx telebot.Context, args ...string) (err error) {
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	var ol OrderLimit
	err = json.Unmarshal([]byte(args[0]), &ol)
	if err == nil {
		a := setLimitAction{
			clientAdmin: lh.ClientAdmin,
			groupId:     lh.GroupId,
			userId:      userId,
			order:       ol,
		}
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = lh.CfgPayment.Backoff.Init
		b.Multiplier = lh.CfgPayment.Backoff.Factor
		b.MaxElapsedTime = lh.CfgPayment.Backoff.LimitTotal
		err = backoff.RetryNotify(a.runOnce, b, func(err error, d time.Duration) {
			lh.Log.Warn(fmt.Sprintf(msgFmtRunOnceFailed, userId, err, d))
			if d > 1*time.Second {
				_ = tgCtx.Send("Updating the usage limit, please wait...")
			}
		})
	}
	if err == nil {
		err = tgCtx.Send("Limit has been successfully increased")
	}
	return
}

type setLimitAction struct {
	clientAdmin admin.Service
	groupId     string
	userId      string
	order       OrderLimit
}

func (a setLimitAction) runOnce() (err error) {
	err = a.clientAdmin.SetLimits(context.TODO(), a.groupId, a.userId, a.order.Subject, int64(a.order.Count), a.order.Expires)
	return
}

func formatUsageSubject(subj usage.Subject) (s string) {
	switch subj {
	case usage.SubjectPublishEvents:
		s = "Message Daily Publications"
	case usage.SubjectSubscriptions:
		s = "Subscriptions Count"
	default:
		s = "undefined"
	}
	return
}

func FormatUsageLimit(subj string, u usage.Usage, l usage.Limit) (txt string) {
	var expires string
	switch l.Expires.IsZero() {
	case true:
		expires = "never"
	default:
		expires = l.Expires.Format(time.RFC3339)
	}
	txt = fmt.Sprintf(msgFmtUsageLimit, subj, u.Count, l.Count, expires)
	return
}

func (lh LimitsHandler) RequestIncrease(tgCtx telebot.Context, args ...string) (err error) {
	var subjCode int64
	subjCode, err = strconv.ParseInt(args[0], 10, strconv.IntSize)
	if err == nil {
		err = tgCtx.Send("Reply the count to add to the current limit:")
	}
	if err == nil {
		err = tgCtx.Send(
			fmt.Sprintf("%s %d", ReqLimitIncrease, subjCode),
			&telebot.ReplyMarkup{
				ForceReply:  true,
				Placeholder: "10",
			},
		)
	}
	return
}

func (lh LimitsHandler) HandleIncrease(tgCtx telebot.Context, args ...string) (err error) {
	//
	var subjCode int64
	subjCode, err = strconv.ParseInt(args[1], 10, strconv.IntSize)
	var subj usage.Subject
	var countAdd int64
	if err == nil {
		subj = usage.Subject(subjCode)
		countAdd, err = strconv.ParseInt(args[2], 10, strconv.IntSize)
	}
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	var l usage.Limit
	ctxGroupId := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", lh.GroupId)
	l, err = lh.ClientAwk.ReadUsageLimit(ctxGroupId, userId, subj)
	//
	var days int64
	var expiresNew time.Time
	switch l.Expires.After(time.Now()) {
	case true: // not expired yet, increase for remaining period only
		days = int64(l.Expires.Sub(time.Now()) / (24 * time.Hour))
		expiresNew = l.Expires
	default: // expired, set the limit for the default period
		days = ExpiresDefaultDays
		expiresNew = time.Now().UTC().Add(time.Hour * time.Duration(24*days))
	}
	//
	var priceTotal float64
	if err == nil {
		var pricePerItem float64
		switch subj {
		case usage.SubjectSubscriptions:
			pricePerItem = lh.CfgPayment.Price.Subscription.CountLimit
			priceTotal = pricePerItem * float64(days*countAdd)
		case usage.SubjectPublishEvents:
			pricePerItem = lh.CfgPayment.Price.MessagePublishing.DailyLimit
			priceTotal = pricePerItem * float64(days*countAdd)
		}
		if priceTotal <= 1 {
			err = fmt.Errorf("%w: total price too low: %s %f", errInvalidOrder, lh.CfgPayment.Currency.Code, priceTotal)
		}
	}
	//
	var ol OrderLimit
	if err == nil {
		ol.Expires = expiresNew
		ol.Count = uint32(l.Count + countAdd)
		ol.Subject = usage.Subject(subjCode)
		err = ol.validate()
	}
	var orderPayloadData []byte
	if err == nil {
		orderPayloadData, err = json.Marshal(ol)
	}
	var orderData []byte
	if err == nil {
		o := service.Order{
			Purpose: PurposeLimitSet,
			Payload: string(orderPayloadData),
		}
		orderData, err = json.Marshal(o)
	}
	//
	label := fmt.Sprintf("%s: %d x %d days", formatUsageSubject(subj), ol.Count, days)
	if err == nil {
		price := int(priceTotal * lh.CfgPayment.Currency.SubFactor)
		invoice := telebot.Invoice{
			Start:       uuid.NewString(),
			Title:       fmt.Sprintf("%s set %d until %s", formatUsageSubject(subj), ol.Count, expiresNew.Format(time.RFC3339)),
			Description: label,
			Payload:     string(orderData),
			Currency:    lh.CfgPayment.Currency.Code,
			Prices: []telebot.Price{
				{
					Label:  label,
					Amount: price,
				},
			},
			Token: lh.CfgPayment.Provider.Token,
			Total: price,
		}
		_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
	}
	//
	return
}

func (lh LimitsHandler) IncreasePreCheckout(tgCtx telebot.Context, args ...string) (err error) {
	err = tgCtx.Accept()
	return
}
