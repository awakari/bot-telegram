package util

import (
	"gopkg.in/telebot.v3"
	"strconv"
)

const prefixUserId = "tg://user?id="

func SenderToUserId(ctxTg telebot.Context) (id string) {
	sender := ctxTg.Sender()
	if sender != nil {
		id = TelegramToAwakariUserId(sender.ID)
	}
	return
}

func TelegramToAwakariUserId(tgUserId int64) (id string) {
	id = prefixUserId + strconv.FormatInt(tgUserId, 10)
	return
}
