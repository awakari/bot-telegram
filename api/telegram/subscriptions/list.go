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
const subListLimit = 256 // TODO: implement the proper pagination later

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
				row := m.Row(
					telebot.Btn{
						Text: fmt.Sprintf("📥 %s", sub.Description),
						Data: fmt.Sprintf("%s %s", "inbox", subId),
					},
					telebot.Btn{
						Text: "✎",
						Data: fmt.Sprintf("%s %s", "details", subId),
					},
					telebot.Btn{
						Text: "❌",
						Data: fmt.Sprintf("%s %s", "delete", subId),
					},
				)
				rows = append(rows, row)
			}
			m.Inline(rows...)
			err = ctx.Send("Subscriptions", m)
		}
		return
	}
}
