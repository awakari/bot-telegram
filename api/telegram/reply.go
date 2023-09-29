package telegram

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
	"strings"
)

func HandleReply(
	tgCtx telebot.Context,
	awakariClient api.Client,
	groupId string,
	replyHandlers map[string]func(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) error,
) (err error) {
	msgResp := tgCtx.Message()
	txtResp := msgResp.Text
	msgReq := msgResp.ReplyTo
	txtReq := msgReq.Text
	argsReq := strings.Split(txtReq, " ")
	handlerKey := argsReq[0]
	fmt.Printf("handler key: %s\n", handlerKey)
	rh, rhOk := replyHandlers[handlerKey]
	fmt.Printf("rh: %+v, rhOk: %t\n", rh, rhOk)
	switch rhOk {
	case false:
		err = errors.New(fmt.Sprintf("unknown reply handler key: %s", handlerKey))
	default:
		var args []string
		if len(argsReq) > 1 {
			args = append(args, argsReq[1:]...)
		}
		args = append(argsReq, txtResp)
		fmt.Printf("args: %+v\n", args)
		err = rh(tgCtx, awakariClient, groupId, args...)
	}
	return
}
