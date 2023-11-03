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
	"time"
)

type UsageHandler struct {
	ClientAwk api.Client
	GroupId   string
}

func (uh UsageHandler) Show(tgCtx telebot.Context) (err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", uh.GroupId)
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	var u awkUsage.Usage
	if err == nil {
		u, err = uh.ClientAwk.ReadUsage(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
	}
	var l awkUsage.Limit
	if err == nil {
		l, err = uh.ClientAwk.ReadUsageLimit(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
	}
	if err == nil {
		respTxt := usage.FormatUsageLimit("Daily Messages Publishing", u, l)
		m := &telebot.ReplyMarkup{}
		btns := []telebot.Btn{
			{
				Text: usage.LabelIncrease,
				Data: fmt.Sprintf("%s %d", usage.CmdIncrease, awkUsage.SubjectPublishEvents),
			},
		}
		switch {
		case l.Expires.After(time.Now()):
			btns = append(btns, telebot.Btn{
				Text: usage.LabelExtend,
				Data: fmt.Sprintf("%s %d", usage.CmdExtend, awkUsage.SubjectPublishEvents),
			})
		}
		m.Inline(m.Row(btns...))
		err = tgCtx.Send(respTxt, m, telebot.ModeHTML)
	}
	return
}
