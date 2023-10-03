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

const fmtUsageLimit = `<pre>Usage:
  Count:   %d
  Limit:   %d
  Expires: %s
</pre>`

const subCurrencyFactor = 100 // this is valid for roubles, dollars, euros
const readUsageLimitTimeout = 10 * time.Second

func ExtendLimitsInvoice(paymentProviderToken string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		var o Order
		err = json.Unmarshal([]byte(args[0]), &o)
		if err == nil {
			err = o.validate()
		}
		if err == nil {
			invoice := telebot.Invoice{
				Title:       "Usage Limit Increase",
				Description: formatUsageSubject(o.Limit.Subject),
				Payload:     args[0],
				Currency:    o.Price.Unit,
				Prices: []telebot.Price{
					{
						Label:  formatUsageSubject(o.Limit.Subject),
						Amount: int(o.Price.Total * subCurrencyFactor),
					},
				},
				Token: paymentProviderToken,
				Total: int(o.Price.Total * subCurrencyFactor),
			}
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
		return
	}
}

func ExtendLimitsPreCheckout(clientAwk api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		q := tgCtx.PreCheckoutQuery()
		userId := strconv.FormatInt(q.Sender.ID, 10)
		var o Order
		err = json.Unmarshal([]byte(q.Payload), &o)
		var currentLimit usage.Limit
		if err == nil {
			ctx, cancel := context.WithTimeout(context.TODO(), readUsageLimitTimeout)
			ctx = metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
			defer cancel()
			currentLimit, err = clientAwk.ReadUsageLimit(ctx, userId, o.Limit.Subject)
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

func ExtendLimits(clientAdmin admin.Service, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		p := tgCtx.Message().Payment
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var o Order
		err = json.Unmarshal([]byte(p.Payload), &o)
		if err == nil {
			expires := time.Now().Add(time.Duration(o.Limit.TimeDays) * time.Hour * 24)
			err = clientAdmin.SetLimits(context.TODO(), groupId, userId, o.Limit.Subject, int64(o.Limit.Count), expires)
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
		s = "Enabled Subscriptions Count"
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
