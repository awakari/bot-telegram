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
	argsReq := strings.Split(txtReq, " ")
	handlerKey := argsReq[0]
	argsReq = argsReq[1:]
	rh, rhOk := replyHandlers[handlerKey]
	switch rhOk {
	case true:
		var args []string
		if len(argsReq) > 1 {
			args = append(args, argsReq...)
		}
		args = append(argsReq, txtResp)
		err = rh(tgCtx, args...)
	default:
		err = errors.New(fmt.Sprintf("unknown reply handler key: %s", handlerKey))
	}
	return
}
