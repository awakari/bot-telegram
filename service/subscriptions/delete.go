package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"strings"
)

const CmdDelete = "delete"
const ReqDelete = "sub_delete"

func DeleteHandlerFunc(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		_ = tgCtx.Send("Are you sure? Reply \"yes\" or \"no\" to the next message:")
		err = tgCtx.Send(
			fmt.Sprintf("%s %s", ReqDelete, subId),
			&telebot.ReplyMarkup{
				ForceReply:  true,
				Placeholder: "no",
			},
		)
		return
	}
}

func DeleteReplyHandlerFunc(clientAwk api.Client, groupId string, kbd *telebot.ReplyMarkup) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		if len(args) != 3 {
			err = errors.New("invalid argument count")
		}
		subId, reply := args[1], strings.ToLower(args[2])
		switch reply {
		case "yes":
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
			userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
			err = clientAwk.DeleteSubscription(groupIdCtx, userId, subId)
			if err == nil {
				err = tgCtx.Send("Subscription deleted", kbd)
			}
		default:
			err = tgCtx.Send("Subscription deletion cancelled", kbd)
		}
		return
	}
}
