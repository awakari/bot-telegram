package telegram

import (
	"fmt"
	"gopkg.in/telebot.v3"
)

func UserLeft(ctx telebot.Context) (err error) {
	msg := ctx.Message()
	fmt.Printf("User %d left the chat %d\n", msg.UserLeft.ID, ctx.Chat().ID)
	return
}
