package service

import (
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"gopkg.in/telebot.v3"
)

type PricesHandler struct {
	CfgPayment config.PaymentConfig
	RestoreKbd *telebot.ReplyMarkup
}

const fmtMsgPrices = `
<b>Always Free</b>:
🎁 1 subscription that never expires. 
🎁 Publish up to 10 messages daily.
🎁 Adding own publishing sources.

<b>Prices</b> (in %s):
<pre>
- Custom Usage Limits:
  - Subscriptions Count, starting from 2nd:
	%.2f per item per day
  - Messages Publication, starting from 11th: 
	%.2f per item per day
- On Demand:
  - A subscription extension: 
	%.2f per day 
  - A message publication when limit is reached: 
	%.2f
</pre>
`

func (ph PricesHandler) Prices(tgCtx telebot.Context) (err error) {
	err = tgCtx.Send(
		fmt.Sprintf(
			fmtMsgPrices,
			ph.CfgPayment.Currency.Code,
			ph.CfgPayment.Price.Subscription.CountLimit,
			ph.CfgPayment.Price.MessagePublishing.DailyLimit,
			ph.CfgPayment.Price.Subscription.Extension,
			ph.CfgPayment.Price.MessagePublishing.Extra,
		),
		ph.RestoreKbd,
		telebot.ModeHTML,
	)
	return
}
