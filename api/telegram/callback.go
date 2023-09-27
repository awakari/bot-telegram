package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"strings"
)

var errInvalidCallbackData = errors.New("invalid callback data")
var errInvalidCallbackCmd = errors.New("invalid callback command")

func Callback(handlers map[string]func(ctx telebot.Context, args ...string) (err error)) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		data := ctx.Callback().Data
		parts := strings.Split(data, " ")
		if len(parts) != 2 {
			err = fmt.Errorf("%w: %s", errInvalidCallbackData, data)
		}
		var arg string
		var f func(ctx telebot.Context, args ...string) (err error)
		if err == nil {
			cmd := parts[0]
			arg = parts[1]
			var ok bool
			f, ok = handlers[cmd]
			if !ok {
				err = fmt.Errorf("%w: %s", errInvalidCallbackCmd, cmd)
			}
		}
		if err == nil {
			err = f(ctx, arg)
		}
		return
	}
}
