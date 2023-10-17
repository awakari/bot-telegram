package messages

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

func DetailsHandlerFunc(clientAwk api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		//
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		//
		var u awkUsage.Usage
		u, err = clientAwk.ReadUsage(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
		var l awkUsage.Limit
		if err == nil {
			l, err = clientAwk.ReadUsageLimit(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
		}
		if err == nil {
			respTxt := usage.FormatUsageLimit("Publishing", u, l)
			m := &telebot.ReplyMarkup{}
			m.Inline(m.Row(telebot.Btn{
				Text: usage.LabelLimitIncrease,
				Data: fmt.Sprintf("%s %d", usage.CmdLimit, awkUsage.SubjectPublishEvents),
			}))
			err = tgCtx.Send(respTxt, m, telebot.ModeHTML)
		}
		if err == nil {
			m := &telebot.ReplyMarkup{}
			m.Inline(m.Row(
				telebot.Btn{
					Text: "Own",
				},
				telebot.Btn{
					Text: "Common",
				},
			))
			err = tgCtx.Send("List Sources:", m)
		}
		//
		return
	}
}
