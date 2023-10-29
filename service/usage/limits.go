package usage

import (
	"context"
	"encoding/json"
	"errors"
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

const LabelLimitIncrease = "🛒 Set New Limit"
const CmdLimit = "limit"
const ReqLimitSet = "limit_set"

const PurposeLimits = "limits"
const msgFmtUsageLimit = `%s Usage:<pre>
  Count:   %d
  Limit:   %d
  Expires: %s
</pre>`
const msgFmtRunOnceFailed = "failed to set limits, user id: %s, cause: %s, retrying in: %s"

func RequestNewLimit() service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		var subjCode int64
		subjCode, err = strconv.ParseInt(args[0], 10, strconv.IntSize)
		subj := usage.Subject(subjCode)
		if err == nil {
			switch subj {
			case usage.SubjectSubscriptions:
				err = tgCtx.Send("Reply with a new count limit (at least 2):")
			case usage.SubjectPublishEvents:
				err = tgCtx.Send("Reply with a new count limit (at least 11):")
			default:
				err = errors.New(fmt.Sprintf("unrecognzied subject code: %d", subjCode))
			}
		}
		if err == nil {
			err = tgCtx.Send(
				fmt.Sprintf("%s %d", ReqLimitSet, subjCode),
				&telebot.ReplyMarkup{
					ForceReply:  true,
					Placeholder: "100",
				},
			)
		}
		return
	}
}

func HandleNewLimit(cfgPayment config.PaymentConfig) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		var subjCode int64
		subjCode, err = strconv.ParseInt(args[1], 10, strconv.IntSize)
		var subj usage.Subject
		var count int64
		if err == nil {
			subj = usage.Subject(subjCode)
			count, err = strconv.ParseInt(args[2], 10, strconv.IntSize)
		}
		var priceTotal float64
		if err == nil {
			var pricePerItem float64
			switch subj {
			case usage.SubjectSubscriptions:
				pricePerItem = cfgPayment.Price.Subscription.CountLimit
				priceTotal = pricePerItem * float64(ExpiresDefaultDays*(count-1))
			case usage.SubjectPublishEvents:
				pricePerItem = cfgPayment.Price.MessagePublishing.DailyLimit
				priceTotal = pricePerItem * float64(ExpiresDefaultDays*(count-10))
			}
			if priceTotal <= 0 {
				err = fmt.Errorf("%w: non-positive total price %f", errInvalidOrder, priceTotal)
			}
		}
		var ol OrderLimit
		if err == nil {
			ol.TimeDays = ExpiresDefaultDays
			ol.Count = uint32(count)
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
				Purpose: PurposeLimits,
				Payload: string(orderPayloadData),
			}
			orderData, err = json.Marshal(o)
		}
		label := fmt.Sprintf(
			"%s: %d x %d days", formatUsageSubject(subj), count, ExpiresDefaultDays,
		)
		if err == nil {
			price := int(priceTotal * cfgPayment.Currency.SubFactor)
			invoice := telebot.Invoice{
				Start:       uuid.NewString(),
				Title:       fmt.Sprintf("%s limit", formatUsageSubject(subj)),
				Description: label,
				Payload:     string(orderData),
				Currency:    cfgPayment.Currency.Code,
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

func NewLimitPreCheckout(
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

func HandleNewLimitPaid(
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
