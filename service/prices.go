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
Always Free:
* 1 subscription that never expires. 
* Publish up to 10 messages daily.
* Adding own publishing sources.

Prices:
* Every subscription starting from the 2nd, daily: %s %.2f
* Every message publication starting from 11th, up to the limit: %s %.2f
* Every message publication above current limit: %s %.2f
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
			ph.CfgPayment.Price.MessagePublishing.Extra,
		),
		telebot.ModeMarkdownV2,
	)
	return
}
