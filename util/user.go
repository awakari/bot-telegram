package util

import (
	"gopkg.in/telebot.v3"
	"strconv"
)

const PrefixUserId = "tg://user?id="

func SenderToUserId(ctxTg telebot.Context) (id string) {
	sender := ctxTg.Sender()
	if sender != nil {
		id = PrefixUserId + strconv.FormatInt(sender.ID, 10)
	}
	return
}
