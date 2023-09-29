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
	argsReq = argsReq[1:]
	fmt.Printf("args req: %+v\n", argsReq)
	rh, rhOk := replyHandlers[handlerKey]
	switch rhOk {
	case false:
		err = errors.New(fmt.Sprintf("unknown reply handler key: %s", handlerKey))
	default:
		var args []string
		if len(argsReq) > 1 {
			args = append(args, argsReq...)
		}
		args = append(argsReq, txtResp)
		fmt.Printf("args: %+v\n", args)
		err = rh(tgCtx, awakariClient, groupId, args...)
	}
	return
}
