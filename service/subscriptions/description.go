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
)

const CmdDescription = "description"
const ReqDescribe = "sub_describe"

func DescriptionHandlerFunc(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var sd subscription.Data
		sd, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
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

func DescriptionReplyHandlerFunc(clientAwk api.Client, groupId string, kbd *telebot.ReplyMarkup) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		if len(args) != 3 {
			err = errors.New("invalid argument count")
		}
		subId, descr := args[1], args[2]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var sd subscription.Data
		sd, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			sd.Description = descr
			err = clientAwk.UpdateSubscription(groupIdCtx, userId, subId, sd)
		}
		if err == nil {
			// force reply removes the keyboard, hence don't forget to restore it
			err = tgCtx.Send(fmt.Sprintf("Subscription description changed to \"%s\"", descr), kbd)
		}
		return
	}
}
