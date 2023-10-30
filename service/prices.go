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
🎁 Adding own publishing sources.
🎁 Publish up to 10 messages daily *.

<b>Prices</b> (in %s):

Committed Usage:
  - Subscriptions quota **: 
<pre>     %.2f per item-day</pre>
  - Message publications quota ***: 
<pre>     %.2f per item-day</pre>

On Demand:
  - Extend a subscription time:
<pre>     %.2f per day</pre>
  - Publish a message after the current limit is reached: 
<pre>     %.2f per message</pre>

* Includes the messages been published from added sources.
** Above free level (starting from 2nd subscription).
***  Above free level (starting from 11th message per day)`

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
