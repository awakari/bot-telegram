package service

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"strings"
)

type ArgHandlerFunc func(tgCtx telebot.Context, args ...string) (err error)

var errInvalidCallbackData = errors.New("invalid callback data")
var errInvalidCallbackCmd = errors.New("invalid callback command")

func Callback(handlers map[string]ArgHandlerFunc) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		data := ctx.Callback().Data
		parts := strings.Split(data, " ")
		if len(parts) < 1 {
			err = fmt.Errorf("%w: %s", errInvalidCallbackData, data)
		}
		var args []string
		var f ArgHandlerFunc
		if err == nil {
			cmd := parts[0]
			if len(parts) > 1 {
				args = parts[1:]
			}
			var ok bool
			f, ok = handlers[cmd]
			if !ok {
				err = fmt.Errorf("%w: %s", errInvalidCallbackCmd, cmd)
			}
		}
		if err == nil {
			err = f(ctx, args...)
		}
		return
	}
}
