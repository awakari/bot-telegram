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
🎁 Follow up to 1 interest forever.
🎁 Up to 20 message publications daily ¹.
🎁 Adding own publishing sources.

<b>Prices</b> (in %s):
<i>Payments are currently in the test mode. There are no real money transfer.</i>

Committed Usage:
  - Interests quota ²: 
<pre>     %.2f per item-day</pre>
  - Message publications quota ³: 
<pre>     %.2f per item-day</pre>

On Demand:
  - Extend an interest following time:
<pre>     %.2f per day</pre>
  - Publish a message after the current limit is reached: 
<pre>     %.2f per message</pre>

(1) Includes the messages been published from added sources.
(2) Starting from 2nd interest.
(3) Starting from 11th message per day.`

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
