package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"strings"
)

var ErrCreateSubNoArgs = errors.New("not enough arguments to create a text subscription")

func CreateTextSubscription(ctx telebot.Context) (err error) {
	txt := ctx.Text()
	args, ok := strings.CutPrefix(txt, "/sub ")
	if !ok {
		err = ErrCreateSubNoArgs
	}
	if err == nil {
		fmt.Printf("Create a text subscription with args: %s\n", args)
	}
	return
}
