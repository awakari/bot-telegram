package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
)

const CmdDescription = "description"

const ReqDescribe = "sub_describe"

func DescriptionHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var sd subscription.Data
		sd, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			_ = tgCtx.Send("Reply with a new description:")
			err = tgCtx.Send(
				fmt.Sprintf("%s %s", ReqDescribe, subId),
				&telebot.ReplyMarkup{
					ForceReply:  true,
					Placeholder: sd.Description,
				},
			)
		}
		return
	}
}

func DescriptionReplyHandlerFunc(awakariClient api.Client, groupId string) func(tgCtx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		if len(args) != 3 {
			err = errors.New("invalid argument count")
		}
		subId, descr := args[1], args[2]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var sd subscription.Data
		sd, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			sd.Description = descr
			err = awakariClient.UpdateSubscription(groupIdCtx, userId, subId, sd)
		}
		if err == nil {
			// force reply removes the keyboard, hence don't forget to restore it
			err = tgCtx.Send("Subscription description updated", service.GetReplyKeyboard())
		}
		return
	}
}
