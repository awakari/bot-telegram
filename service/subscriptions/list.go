package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	awkUsage "github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const subListLimit = 256 // TODO: implement the proper pagination

func ListHandlerFunc(clientAwk api.Client, groupId string) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		//
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		//
		var respTxt string
		var u awkUsage.Usage
		u, err = clientAwk.ReadUsage(groupIdCtx, userId, awkUsage.SubjectSubscriptions)
		var l awkUsage.Limit
		if err == nil {
			l, err = clientAwk.ReadUsageLimit(groupIdCtx, userId, awkUsage.SubjectSubscriptions)
		}
		if err == nil {
			respTxt += usage.FormatUsageLimit(u, l)
		}
		//
		var subIds []string
		subIds, err = clientAwk.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
		m := &telebot.ReplyMarkup{}
		if err == nil {
			var sub subscription.Data
			var rows []telebot.Row
			for _, subId := range subIds {
				sub, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
				if err != nil {
					break
				}
				descr := sub.Description
				now := time.Now().UTC()
				switch {
				case sub.Expires.IsZero(): // never expires
					descr += " ∞"
				case sub.Expires.Before(now):
					descr += " ⚠"
				case sub.Expires.Sub(now) < 168*time.Hour: // expires earlier than in 1 week
					descr += " ⏳"
				}
				row := m.Row(telebot.Btn{
					Text: descr,
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
