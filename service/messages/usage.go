package messages

import (
	"context"
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
		u, err = clientAwk.ReadUsage(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
		var l awkUsage.Limit
		if err == nil {
			l, err = clientAwk.ReadUsageLimit(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
		}
		if err == nil {
			respTxt += usage.FormatUsageLimit("Publishing", u, l)
		}
		m := &telebot.ReplyMarkup{}
		m.Inline(m.Row(telebot.Btn{
			Text: usage.LabelLimitIncrease,
			WebApp: &telebot.WebApp{
				URL: "https://awakari.app/price-calc-msgs.html",
			},
		}))
		if err == nil {
			err = tgCtx.Send(respTxt, m, telebot.ModeHTML)
		}
		return
	}
}
