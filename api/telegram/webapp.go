package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

var errInvalidLabel = errors.New("no handler for webapp label")

func WebAppData(handlers map[string]func(ctx telebot.Context, args ...string) (err error)) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		data := tgCtx.Message().WebAppData
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
