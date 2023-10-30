package subscriptions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

type ExtendOrder struct {
	SubId   string `json:"subId"`
	DaysAdd uint64 `json:"daysAdd"`
}

const PurposeExtend = "sub_extend"
const CmdExtend = "extend"
const ReqSubExtend = "sub_extend"
const daysMin = 10
const daysMax = 365
const msgFmtRunOnceFailed = "failed to extend subscription, id: %s, user id: %s, cause: %s, retrying in: %s"

type ExtendHandler struct {
	CfgPayment config.PaymentConfig
	ClientAwk  api.Client
	GroupId    string
	Log        *slog.Logger
	RestoreKbd *telebot.ReplyMarkup
}

func (eh ExtendHandler) RequestExtensionDaysCount(tgCtx telebot.Context, args ...string) (err error) {
	subId := args[0]
	_ = tgCtx.Send(
		fmt.Sprintf(
			"Reply the number of days to extend (%d-%d). Price is %s %.2f per day.",
			daysMin,
			daysMax,
			eh.CfgPayment.Currency.Code,
			eh.CfgPayment.Price.Subscription.Extension,
		),
	)
	err = tgCtx.Send(
		fmt.Sprintf("%s %s", ReqSubExtend, subId),
		&telebot.ReplyMarkup{
			ForceReply:  true,
			Placeholder: strconv.Itoa(usage.ExpiresDefaultDays),
		},
	)
	return
}

func (eh ExtendHandler) HandleExtensionReply(tgCtx telebot.Context, args ...string) (err error) {
	if len(args) != 3 {
		err = errors.New("invalid argument count")
	}
	subId, daysReply := args[1], args[2]
	var countDays uint64
	countDays, err = strconv.ParseUint(daysReply, 10, 16)
	if err == nil {
		if countDays < daysMin || countDays > daysMax {
			err = errors.New(fmt.Sprintf("invalid days count, should be %d-%d", daysMin, daysMax))
		}
	}
	var orderPayloadData []byte
	if err == nil {
		orderPayloadData, err = json.Marshal(ExtendOrder{
			SubId:   subId,
			DaysAdd: countDays,
		})
	}
	var orderData []byte
	if err == nil {
		o := service.Order{
			Purpose: PurposeExtend,
			Payload: string(orderPayloadData),
		}
		orderData, err = json.Marshal(o)
	}
	if err == nil {
		label := fmt.Sprintf("Subscription %s: add %d days", subId, countDays)
		price := int(float64(countDays) * eh.CfgPayment.Price.Subscription.Extension * eh.CfgPayment.Currency.SubFactor)
		invoice := telebot.Invoice{
			Start:       uuid.NewString(),
			Title:       fmt.Sprintf("Extend subscription by %d days", countDays),
			Description: label,
			Payload:     string(orderData),
			Currency:    eh.CfgPayment.Currency.Code,
			Prices: []telebot.Price{
				{
					Label:  label,
					Amount: price,
				},
			},
			Token: eh.CfgPayment.Provider.Token,
			Total: price,
		}
		err = tgCtx.Send("To proceed, please pay the below invoice", eh.RestoreKbd)
		_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
	}
	return
}

func (eh ExtendHandler) ExtensionPreCheckout(tgCtx telebot.Context, args ...string) (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), eh.CfgPayment.PreCheckout.Timeout)
	defer cancel()
	groupIdCtx := metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", eh.GroupId)
	userId := strconv.FormatInt(tgCtx.PreCheckoutQuery().Sender.ID, 10)
	var op ExtendOrder
	err = json.Unmarshal([]byte(args[0]), &op)
	if err == nil {
		_, err = eh.ClientAwk.ReadSubscription(groupIdCtx, userId, op.SubId)
	}
	switch err {
	case nil:
		err = tgCtx.Accept()
	default:
		err = tgCtx.Accept(err.Error())
	}
	return
}

func (eh ExtendHandler) ExtendPaid(tgCtx telebot.Context, args ...string) (err error) {
	var op ExtendOrder
	err = json.Unmarshal([]byte(args[0]), &op)
	if err == nil {
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		e := extendAction{
			clientAwk: eh.ClientAwk,
			groupId:   eh.GroupId,
			userId:    userId,
			op:        op,
		}
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = eh.CfgPayment.Backoff.Init
		b.Multiplier = eh.CfgPayment.Backoff.Factor
		b.MaxElapsedTime = eh.CfgPayment.Backoff.LimitTotal
		err = backoff.RetryNotify(e.runOnce, b, func(err error, d time.Duration) {
			eh.Log.Warn(fmt.Sprintf(msgFmtRunOnceFailed, op.SubId, userId, err, d))
			if d > 1*time.Second {
				_ = tgCtx.Send("Extending the subscription, please wait...")
			}
		})
	}
	if err == nil {
		err = tgCtx.Send(fmt.Sprintf("Subscription has been successfully extended by %d days", op.DaysAdd), eh.RestoreKbd)
	}
	return
}

type extendAction struct {
	clientAwk api.Client
	groupId   string
	userId    string
	op        ExtendOrder
}

func (e extendAction) runOnce() (err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", e.groupId)
	var sd subscription.Data
	if err == nil {
		sd, err = e.clientAwk.ReadSubscription(groupIdCtx, e.userId, e.op.SubId)
	}
	if err == nil {
		now := time.Now().UTC()
		if sd.Expires.Before(now) {
			sd.Expires = now
		}
		sd.Expires = sd.Expires.Add(time.Duration(e.op.DaysAdd) * 24 * time.Hour)
		err = e.clientAwk.UpdateSubscription(groupIdCtx, e.userId, e.op.SubId, sd)
	}
	return
}
