package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/admin"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
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
const readUsageLimitTimeout = 10 * time.Second

func ExtendLimitsInvoice(paymentProviderToken string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		var op OrderPayload
		err = json.Unmarshal([]byte(args[0]), &op)
		if err == nil {
			err = op.validate()
		}
		var orderData []byte
		if err == nil {
			o := service.Order{
				Purpose: PurposeLimits,
				Payload: args[0],
			}
			orderData, err = json.Marshal(o)
		}
		if err == nil {
			invoice := telebot.Invoice{
				Title:       "Usage Limit Increase",
				Description: formatUsageSubject(op.Limit.Subject),
				Payload:     string(orderData),
				Currency:    op.Price.Unit,
				Prices: []telebot.Price{
					{
						Label:  formatUsageSubject(op.Limit.Subject),
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
		var op OrderPayload
		err = json.Unmarshal([]byte(args[0]), &op)
		var currentLimit usage.Limit
		if err == nil {
			ctx, cancel := context.WithTimeout(context.TODO(), readUsageLimitTimeout)
			ctx = metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
			defer cancel()
			currentLimit, err = clientAwk.ReadUsageLimit(ctx, userId, op.Limit.Subject)
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
		var op OrderPayload
		err = json.Unmarshal([]byte(args[0]), &op)
		if err == nil {
			expires := time.Now().Add(time.Duration(op.Limit.TimeDays) * time.Hour * 24)
			err = clientAdmin.SetLimits(context.TODO(), groupId, userId, op.Limit.Subject, int64(op.Limit.Count), expires)
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
