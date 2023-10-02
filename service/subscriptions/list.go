package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const CmdList = "list"
const subListLimit = 256 // TODO: implement the proper pagination
const respFmt = `<pre>Usage:
  Used:  %d
  Limit: %d
  Until: %s
</pre>`

func ListHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		//
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		//
		var respTxt string
		var u usage.Usage
		u, err = awakariClient.ReadUsage(groupIdCtx, userId, usage.SubjectSubscriptions)
		var l usage.Limit
		if err == nil {
			l, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, usage.SubjectSubscriptions)
		}
		if err == nil {
			var until string
			switch l.Expires.IsZero() {
			case true:
				until = "&lt;not set&gt;"
			default:
				until = l.Expires.Format(time.RFC3339)
			}
			respTxt += fmt.Sprintf(respFmt, u.Count, l.Count, until)
		}
		//
		var subIds []string
		subIds, err = awakariClient.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
		m := &telebot.ReplyMarkup{}
		if err == nil {
			var sub subscription.Data
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
		}
		//
		if err == nil {
			err = ctx.Send(respTxt, m, telebot.ModeHTML)
		}
		//
		return
	}
}
