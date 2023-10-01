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
const msgFmtDetails = `
<pre>
  %s:
    Used:  %d
    Limit: %d
</pre>`

var subjects = []usage.Subject{
	usage.SubjectSubscriptions,
	usage.SubjectPublishEvents,
}

func ViewHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		msgTxt := "Current usage:"
		for _, subj := range subjects {
			var u usage.Usage
			u, err = awakariClient.ReadUsage(groupIdCtx, userId, subj)
			var l usage.Limit
			if err == nil {
				l, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, subj)
			}
			if err == nil {
				msgTxt += fmt.Sprintf(msgFmtDetails, formatSubject(subj), u.Count, l.Count)
			}
		}
		msgTxt += "\nTap the \"Extend Usage Limits\" keyboard button to change."
		err = tgCtx.Send(msgTxt, telebot.ModeHTML)
		return
	}
}

func formatSubject(subj usage.Subject) (txt string) {
	switch subj {
	case usage.SubjectSubscriptions:
		txt = "Enabled Subscriptions"
	case usage.SubjectPublishEvents:
		txt = "Message Daily Publications"
	default:
		txt = "Undefined"
	}
	return
}
