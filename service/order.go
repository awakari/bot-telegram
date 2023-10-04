package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

type Order struct {
	Purpose string `json:"purpose"`
	Payload any    `json:"payload"`
}

const Currency = "EUR"
const SubCurrencyFactor = 100 // this is valid for roubles, dollars, euros

func PreCheckout(handlers map[string]ArgHandlerFunc) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		q := tgCtx.PreCheckoutQuery()
		var o Order
		err = json.Unmarshal([]byte(q.Payload), &o)
		if err == nil {
			h, hOk := handlers[o.Purpose]
			switch hOk {
			case true:
				err = h(tgCtx, q.Payload)
			default:
				err = errors.New(fmt.Sprintf("unknown pre-checkout purpose key: %s", o.Purpose))
			}
		}
		return
	}
}

func Payment(handlers map[string]ArgHandlerFunc) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		p := tgCtx.Message().Payment
		var o Order
		err = json.Unmarshal([]byte(p.Payload), &o)
		if err == nil {
			h, hOk := handlers[o.Purpose]
			switch hOk {
			case true:
				err = h(tgCtx, p.Payload)
			default:
				err = errors.New(fmt.Sprintf("unknown payment purpose key: %s", o.Purpose))
			}
		}
		return
	}
}
