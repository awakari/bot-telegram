package service

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"strings"
)

func HandleReply(
	tgCtx telebot.Context,
	replyHandlers map[string]func(tgCtx telebot.Context, args ...string) error,
) (err error) {
	msgResp := tgCtx.Message()
	txtResp := msgResp.Text
	msgReq := msgResp.ReplyTo
	txtReq := msgReq.Text
	args := strings.Split(txtReq, " ")
	handlerKey := args[0]
	rh, rhOk := replyHandlers[handlerKey]
	switch rhOk {
	case true:
		args = append(args, txtResp)
		err = rh(tgCtx, args...)
	default:
		err = errors.New(fmt.Sprintf("unknown reply handler key: %s", handlerKey))
	}
	return
}
