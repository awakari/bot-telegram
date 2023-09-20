package subscriptions

import (
	"context"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
)

const subListLimit = 10 // TODO: implement the proper pagination later
const msgStart = "Select a subscription from the list below:"

func ListHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		var subIds []string
		subIds, err = awakariClient.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
		m := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		if err == nil {
			var sub subscription.Data
			for _, subId := range subIds {
				sub, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
				if err != nil {
					break
				}
				row := m.Row(telebot.Btn{
					Text: sub.Description,
					Data: "viewsub " + subId,
				})
				rows = append(rows, row)
			}
		}
		m.Inline(rows...)
		err = ctx.Send(msgStart, m)
		return
	}
}
