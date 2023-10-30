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
**Always Free**:
✅ 1 subscription that never expires. 
✅ Publish up to 10 messages daily.
✅ Adding own publishing sources.

**Prices**:
- Custom Usage Limit:
	- Every subscription starting from the 2nd: %s %.2f per day
	- Every message publication starting from 11th: %s %.2f per day
- On Demand:
	- A subscription extension: %s %.2f per day 
	- A message publication when limit is reached: %s %.2f
`

func (ph PricesHandler) Prices(tgCtx telebot.Context) (err error) {
	err = tgCtx.Send(
		fmt.Sprintf(
			fmtMsgPrices,
			ph.CfgPayment.Currency.Code,
			ph.CfgPayment.Price.Subscription.CountLimit,
			ph.CfgPayment.Currency.Code,
			ph.CfgPayment.Price.MessagePublishing.DailyLimit,
			ph.CfgPayment.Currency.Code,
			ph.CfgPayment.Price.Subscription.Extension,
			ph.CfgPayment.Currency.Code,
			ph.CfgPayment.Price.MessagePublishing.Extra,
		),
		ph.RestoreKbd,
		telebot.ModeMarkdown,
	)
	return
}
