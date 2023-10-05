package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/admin"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const PurposeLimits = "limits"

const fmtUsageLimit = `<pre>Usage:
  Count:   %d
  Limit:   %d
  Expires: %s
</pre>`

func ExtendLimitsInvoice(paymentProviderToken string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		var op OrderPayload
		err = json.Unmarshal([]byte(args[0]), &op)
		if err == nil {
			err = op.validate()
		}
		var orderPayloadData []byte
		if err == nil {
			orderPayloadData, err = json.Marshal(op.Limit)
		}
		var orderData []byte
		if err == nil {
			o := service.Order{
				Purpose: PurposeLimits,
				Payload: string(orderPayloadData),
			}
			orderData, err = json.Marshal(o)
		}
		label := fmt.Sprintf(
			"%s: %d x %d days", formatUsageSubject(op.Limit.Subject), op.Limit.Count, op.Limit.TimeDays,
		)
		if err == nil {
			invoice := telebot.Invoice{
				Start:       uuid.NewString(),
				Title:       fmt.Sprintf("%s limit", formatUsageSubject(op.Limit.Subject)),
				Description: label,
				Payload:     string(orderData),
				Currency:    op.Price.Unit,
				Prices: []telebot.Price{
					{
						Label:  label,
						Amount: int(op.Price.Total * service.SubCurrencyFactor),
					},
				},
				Token: paymentProviderToken,
				Total: int(op.Price.Total * service.SubCurrencyFactor),
			}
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
		return
	}
}

func ExtendLimitsPreCheckout(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(tgCtx.PreCheckoutQuery().Sender.ID, 10)
		var ol OrderLimit
		err = json.Unmarshal([]byte(args[0]), &ol)
		var currentLimit usage.Limit
		if err == nil {
			ctx, cancel := context.WithTimeout(context.TODO(), service.PreCheckoutTimeout)
			defer cancel()
			ctx = metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", groupId)
			currentLimit, err = clientAwk.ReadUsageLimit(ctx, userId, ol.Subject)
		}
		if err == nil {
			cle := currentLimit.Expires.UTC()
			if !cle.IsZero() && cle.After(time.Now().UTC()) {
				err = tgCtx.Accept(
					fmt.Sprintf(
						"can not apply new limit, current is not expired yet (expires: %s)",
						cle.Format(time.RFC3339),
					),
				)
			}
		}
		if err == nil {
			err = tgCtx.Accept()
		}
		return
	}
}

func ExtendLimits(clientAdmin admin.Service, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var ol OrderLimit
		err = json.Unmarshal([]byte(args[0]), &ol)
		if err == nil {
			expires := time.Now().Add(time.Duration(ol.TimeDays) * time.Hour * 24)
			err = clientAdmin.SetLimits(context.TODO(), groupId, userId, ol.Subject, int64(ol.Count), expires)
		}
		if err == nil {
			err = tgCtx.Send("Limit has been successfully increased")
		}
		return
	}
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

func FormatUsageLimit(u usage.Usage, l usage.Limit) (txt string) {
	var expires string
	switch l.Expires.IsZero() {
	case true:
		expires = "never"
	default:
		expires = l.Expires.Format(time.RFC3339)
	}
	txt = fmt.Sprintf(fmtUsageLimit, u.Count, l.Count, expires)
	return
}
