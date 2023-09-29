package subscriptions

import (
	"context"
	"github.com/awakari/client-sdk-go/api"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
)

const CmdDelete = "delete"

func DeleteHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		err = awakariClient.DeleteSubscription(groupIdCtx, userId, subId)
		if err == nil {
			err = tgCtx.Send("Subscription deleted")
		}
		return
	}
}
