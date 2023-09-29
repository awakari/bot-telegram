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

const CmdList = "list"
const subListLimit = 256 // TODO: implement the proper pagination

func ListHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		var subIds []string
		subIds, err = awakariClient.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
		if err == nil {
			var sub subscription.Data
			m := &telebot.ReplyMarkup{}
			var rows []telebot.Row
			for _, subId := range subIds {
				sub, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
				if err != nil {
					break
				}
				row := m.Row(telebot.Btn{
					Text: sub.Description,
					Data: fmt.Sprintf("%s %s", CmdDetails, subId),
				})
				rows = append(rows, row)
			}
			m.Inline(rows...)
			err = ctx.Send("Subscriptions", m, telebot.ModeHTML)
		}
		return
	}
}
