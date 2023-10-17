package service

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

var errInvalidLabel = errors.New("no handler for webapp label")

func WebAppData(handlers map[string]ArgHandlerFunc) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		fmt.Printf("Webapp data handler invoked, message is: %+v\n", tgCtx.Message())
		data := tgCtx.Message().WebAppData
		fmt.Printf("Webapp data handler invoked, data is: %+v\n", data)
		label := data.Text
		f, fOk := handlers[label]
		if !fOk {
			err = fmt.Errorf("%w: %s", errInvalidLabel, label)
		}
		if err == nil {
			err = f(tgCtx, data.Data)
		}
		return
	}
}
