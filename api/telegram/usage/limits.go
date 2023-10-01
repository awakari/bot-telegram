package usage

import (
	"encoding/json"
	"fmt"
	"gopkg.in/telebot.v3"
)

const subCurrencyFactor = 100 // this is valid for roubles, dollars, euros

func ExtendLimitsHandlerFunc(paymentProviderToken string) func(tgCtx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		// TODO check the calculation and limits correctness 1st
		var o Order
		err = json.Unmarshal([]byte(args[0]), &o)
		invoice := telebot.Invoice{
			Title:       "Invoice",
			Description: "Usage Limits Extension",
			Payload:     args[0],
			Currency:    o.Price.Unit,
			Prices: []telebot.Price{
				{
					Label:  "Message Publication Rate",
					Amount: int(o.Price.MsgRate * subCurrencyFactor),
				},
				{
					Label:  "Enabled Subscription Count",
					Amount: int(o.Price.SubCount * subCurrencyFactor),
				},
			},
			Token:     paymentProviderToken,
			Total:     int(o.Price.Total * subCurrencyFactor),
			NeedName:  true,
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
