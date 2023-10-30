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

const fmtMsgPrices = `<b>Always Free</b>:
🎁 1 subscription that never expires.
🎁 First 30 days for every subscription starting from 2nd *.
🎁 Adding own publishing sources.
🎁 Publish up to 10 messages daily **.

<b>Prices</b> (in %s):

Committed Usage:
  - Subscriptions Count Limit above free level (starting from 2nd): 
<pre>     %.2f per item-day</pre>
  - Messages Publication Limit above free level (starting from 11th): 
<pre>     %.2f per item-day</pre>

On Demand:
  - A subscription time extension:
<pre>     %.2f per day</pre>
  - A message publication after the current limit is reached: 
<pre>     %.2f per message</pre>

* Requires a subscriptions count limit to be increased 1st.
** Includes the messages been published from added sources.`

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
