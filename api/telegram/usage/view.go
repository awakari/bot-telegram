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
const msgFmtDetails = `<pre>
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
		msgTxt := "Current usage:\n"
		var expires time.Time
		for _, subj := range subjects {
			var u usage.Usage
			u, err = awakariClient.ReadUsage(groupIdCtx, userId, subj)
			var l usage.Limit
			if err == nil {
				l, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, subj)
			}
			if err == nil {
				if l.Expires.After(expires) {
					expires = l.Expires
				}
				msgTxt += fmt.Sprintf(msgFmtDetails, formatSubject(subj), u.Count, l.Count)
			}
		}
		if !expires.IsZero() {
			msgTxt += fmt.Sprintf("\n<pre>  Expires: %s</pre>", expires.Format(time.RFC3339))
		}
		sendOpts := []any{
			telebot.ModeHTML,
		}
		if expires.Before(time.Now()) {
			m := &telebot.ReplyMarkup{}
			m.Inline(m.Row(telebot.Btn{
				Text: "Extend",
				WebApp: &telebot.WebApp{
					URL: "https://awakari.app/price-calc.html",
				},
			}))
			sendOpts = append(sendOpts, m)
		}
		err = tgCtx.Send(msgTxt, sendOpts...)
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
