package usage

import (
	"encoding/json"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/model/usage"
	"gopkg.in/telebot.v3"
	"time"
)

const fmtUsageLimit = `<pre>Usage:
  Count:     %d
  Limit:     %d
    Type:    %s
    Expires: %s
</pre>`

const subCurrencyFactor = 100 // this is valid for roubles, dollars, euros

func ExtendLimitsHandlerFunc(paymentProviderToken string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		// TODO check the calculation and limits correctness 1st
		var o Order
		err = json.Unmarshal([]byte(args[0]), &o)
		invoice := telebot.Invoice{
			Title:       "Invoice",
			Description: "Usage Limit Increase",
			Payload:     args[0],
			Currency:    o.Price.Unit,
			Prices: []telebot.Price{
				{
					Label:  formatUsageSubject(o.Limit.Subject),
					Amount: int(o.Price.Total * subCurrencyFactor),
				},
			},
			Token:     paymentProviderToken,
			Total:     int(o.Price.Total * subCurrencyFactor),
			NeedEmail: true,
			SendEmail: true,
		}
		_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		return
	}
}

func ExtendLimitsPreCheckout(groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		pcq := tgCtx.PreCheckoutQuery()
		fmt.Printf("PreCheckoutQuery for %d:\n%s", pcq.Sender.ID, pcq.Payload)
		err = tgCtx.Accept()
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
	var t string
	switch l.UserId {
	case "":
		t = "group"
	default:
		t = "user"
	}
	var expires string
	switch l.Expires.IsZero() {
	case true:
		expires = "never"
	default:
		expires = l.Expires.Format(time.RFC3339)
	}
	txt = fmt.Sprintf(fmtUsageLimit, u.Count, l.Count, t, expires)
	return
}
