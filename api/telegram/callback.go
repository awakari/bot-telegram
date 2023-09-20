package telegram

import (
	"fmt"
	"gopkg.in/telebot.v3"
)

func Callback(ctx telebot.Context) (err error) {
	fmt.Printf("Callback received\n")
	return
}
