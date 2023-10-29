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

const LabelExtend = "▲ Extend Time"
const CmdExtend = "usage_extend"
const ReqUsageExtend = "usage_extend_req"

const PurposeUsageExtend = "usage_extend"
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
			fmt.Sprintf("%s %d", ReqUsageExtend, subjCode),
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
		if priceTotal <= 0 {
			err = fmt.Errorf("%w: non-positive total price %f", errInvalidOrder, priceTotal)
		}
	}
	var oe OrderExtend
	if err == nil {
		oe.Expires = l.Expires.Add(time.Duration(daysAdd) * time.Hour * 24).UTC()
		oe.Count = uint32(l.Count)
		oe.Subject = usage.Subject(subjCode)
		err = oe.validate()
	}
	var orderPayloadData []byte
	if err == nil {
		orderPayloadData, err = json.Marshal(oe)
	}
	var orderData []byte
	if err == nil {
		o := service.Order{
			Purpose: PurposeUsageExtend,
			Payload: string(orderPayloadData),
		}
		orderData, err = json.Marshal(o)
	}
	label := fmt.Sprintf("%d %s: add %d days", l.Count, formatUsageSubject(subj), daysAdd)
	if err == nil {
		price := int(priceTotal * lh.CfgPayment.Currency.SubFactor)
		invoice := telebot.Invoice{
			Start:       uuid.NewString(),
			Title:       fmt.Sprintf("%s new expiration time: %s", formatUsageSubject(subj), oe.Expires.Format(time.RFC3339)),
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

func (lh LimitsHandler) ExtensionPreCheckout(tgCtx telebot.Context, args ...string) (err error) {
	err = tgCtx.Accept()
	return
}

func (lh LimitsHandler) HandleExtensionPaid(tgCtx telebot.Context, args ...string) (err error) {
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	var oe OrderExtend
	err = json.Unmarshal([]byte(args[0]), &oe)
	if err == nil {
		a := extendAction{
			clientAdmin: lh.ClientAdmin,
			groupId:     lh.GroupId,
			userId:      userId,
			oe:          oe,
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

type extendAction struct {
	clientAdmin admin.Service
	groupId     string
	userId      string
	oe          OrderExtend
}

func (ea extendAction) runOnce() (err error) {
	err = ea.clientAdmin.SetLimits(context.TODO(), ea.groupId, ea.userId, ea.oe.Subject, int64(ea.oe.Count), ea.oe.Expires)
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
