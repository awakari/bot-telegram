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

const PurposeLimits = "limits"
const msgFmtUsageLimit = `<pre>Usage:
  Count:   %d
  Limit:   %d
  Expires: %s
</pre>`
const msgFmtRunOnceFailed = "failed to set limits, user id: %s, cause: %s, retrying in: %s"

func ExtendLimitsInvoice(cfgPayment config.PaymentConfig) service.ArgHandlerFunc {
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
			price := int(op.Price.Total * cfgPayment.Currency.SubFactor)
			invoice := telebot.Invoice{
				Start:       uuid.NewString(),
				Title:       fmt.Sprintf("%s limit", formatUsageSubject(op.Limit.Subject)),
				Description: label,
				Payload:     string(orderData),
				Currency:    op.Price.Unit,
				Prices: []telebot.Price{
					{
						Label:  label,
						Amount: price,
					},
				},
				Token: cfgPayment.Provider.Token,
				Total: price,
			}
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
		return
	}
}

func ExtendLimitsPreCheckout(
	clientAwk api.Client, groupId string, cfgPayment config.PaymentConfig,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(tgCtx.PreCheckoutQuery().Sender.ID, 10)
		var ol OrderLimit
		err = json.Unmarshal([]byte(args[0]), &ol)
		var currentLimit usage.Limit
		if err == nil {
			ctx, cancel := context.WithTimeout(context.TODO(), cfgPayment.PreCheckout.Timeout)
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

func ExtendLimitsPaid(
	clientAdmin admin.Service,
	groupId string,
	log *slog.Logger,
	cfgBackoff config.BackoffConfig,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var ol OrderLimit
		err = json.Unmarshal([]byte(args[0]), &ol)
		if err == nil {
			expires := time.Now().Add(time.Duration(ol.TimeDays) * time.Hour * 24)
			a := extendLimitsAction{
				clientAdmin: clientAdmin,
				groupId:     groupId,
				userId:      userId,
				ol:          ol,
				expires:     expires,
			}
			b := backoff.NewExponentialBackOff()
			b.InitialInterval = cfgBackoff.Init
			b.Multiplier = cfgBackoff.Factor
			b.MaxElapsedTime = cfgBackoff.LimitTotal
			err = backoff.RetryNotify(a.runOnce, b, func(err error, d time.Duration) {
				log.Warn(fmt.Sprintf(msgFmtRunOnceFailed, userId, err, d))
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
}

type extendLimitsAction struct {
	clientAdmin admin.Service
	groupId     string
	userId      string
	ol          OrderLimit
	expires     time.Time
}

func (a extendLimitsAction) runOnce() (err error) {
	err = a.clientAdmin.SetLimits(context.TODO(), a.groupId, a.userId, a.ol.Subject, int64(a.ol.Count), a.expires)
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

func FormatUsageLimit(u usage.Usage, l usage.Limit) (txt string) {
	var expires string
	switch l.Expires.IsZero() {
	case true:
		expires = "never"
	default:
		expires = l.Expires.Format(time.RFC3339)
	}
	txt = fmt.Sprintf(msgFmtUsageLimit, u.Count, l.Count, expires)
	return
}
