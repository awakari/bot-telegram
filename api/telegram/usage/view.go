package usage

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const CmdUsage = "usage"
const msgFmtDetails = `<b>%s</b>:<pre>
  Spent: %d
  Quota: %d
  Since: %s
</pre>`

var subjects = []usage.Subject{
	usage.SubjectSubscriptions,
	usage.SubjectPublishEvents,
}

func ViewHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		for _, subj := range subjects {
			var u usage.Usage
			u, err = awakariClient.ReadUsage(groupIdCtx, userId, subj)
			var l usage.Limit
			if err == nil {
				l, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, subj)
			}
			if err == nil {
				m := &telebot.ReplyMarkup{}
				m.Inline(m.Row(telebot.Btn{
					Text: "Change Quota",
					Data: fmt.Sprintf("%s %d %d", CmdQuota, subj, l.Count),
				}))
				err = tgCtx.Send(
					fmt.Sprintf(msgFmtDetails, formatSubject(subj), u.Count, l.Count, u.Since.Format(time.RFC3339)),
					telebot.ModeHTML,
				)
			}
		}
		return
	}
}

func formatSubject(subj usage.Subject) (txt string) {
	switch subj {
	case usage.SubjectSubscriptions:
		txt = "Subscription Create"
	case usage.SubjectPublishEvents:
		txt = "Message Publish"
	default:
		txt = "Undefined"
	}
	return
}
