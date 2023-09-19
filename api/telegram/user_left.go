package telegram

import (
	"fmt"
	"gopkg.in/telebot.v3"
)

func UserLeft(ctx telebot.Context) (err error) {
	fmt.Printf("User id=%d left, TODO: delete the subscription if owner\n", ctx.Sender().ID)
	return
}
