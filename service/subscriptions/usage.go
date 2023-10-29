package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	awkUsage "github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
)

func Usage(clientAwk api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var respTxt string
		var u awkUsage.Usage
		u, err = clientAwk.ReadUsage(groupIdCtx, userId, awkUsage.SubjectSubscriptions)
		var l awkUsage.Limit
		if err == nil {
			l, err = clientAwk.ReadUsageLimit(groupIdCtx, userId, awkUsage.SubjectSubscriptions)
		}
		if err == nil {
			respTxt += usage.FormatUsageLimit("Subscriptions", u, l)
			m := &telebot.ReplyMarkup{}
			switch {
			case l.Expires.IsZero():
				m.Inline(m.Row(telebot.Btn{
					Text: usage.LabelLimitSet,
					Data: fmt.Sprintf("%s %d", usage.CmdLimit, awkUsage.SubjectPublishEvents),
				}))
			}
			err = tgCtx.Send(respTxt, m, telebot.ModeHTML)
		}
		return
	}
}
