package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
)

const CmdDescription = "description"

const ReplyKeyDescription = "describe"

func DescriptionHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var sd subscription.Data
		sd, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			_ = tgCtx.Send("Please enter the new subscription description:")
			err = tgCtx.Send(
				fmt.Sprintf("describe %s", subId),
				&telebot.ReplyMarkup{
					ForceReply:  true,
					Placeholder: sd.Description,
				},
			)
		}
		return
	}
}

func HandleDescriptionReply(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) (err error) {
	subId, descr := args[0], args[1]
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	var sd subscription.Data
	sd, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
	if err == nil {
		sd.Description = descr
		err = awakariClient.UpdateSubscription(groupIdCtx, userId, subId, sd)
	}
	if err == nil {
		err = tgCtx.Send("Subscription description updated")
	}
	return
}
