package usage

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
)

const CmdUsage = "usage"

func ViewHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var usageSubs usage.Usage
		usageSubs, err = awakariClient.ReadUsage(groupIdCtx, userId, usage.SubjectSubscriptions)
		var limitSubs usage.Limit
		if err == nil {
			limitSubs, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, usage.SubjectSubscriptions)
		}
		if err == nil {
			err = tgCtx.Send(
				fmt.Sprintf(
					"<b>Subscriptions</b>:\nCount: %d\nTotal: %d\nSince: %s\nLimit: %d",
					usageSubs.Count, usageSubs.CountTotal, usageSubs.Since,
					limitSubs.Count,
				),
				telebot.ModeHTML,
			)
		}
		var usageMsgs usage.Usage
		if err == nil {
			usageMsgs, err = awakariClient.ReadUsage(groupIdCtx, userId, usage.SubjectPublishEvents)
		}
		var limitMsgs usage.Limit
		if err == nil {
			limitMsgs, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, usage.SubjectPublishEvents)
		}
		if err == nil {
			err = tgCtx.Send(
				fmt.Sprintf(
					"<b>Publish Messages</b>:\nCount: %d\nTotal: %d\nSince: %s\nLimit: %d",
					usageMsgs.Count, usageMsgs.CountTotal, usageMsgs.Since,
					limitMsgs.Count,
				),
				telebot.ModeHTML,
			)
		}
		return
	}
}
