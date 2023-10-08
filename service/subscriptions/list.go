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
			respTxt += usage.FormatUsageLimit(u, l)
		}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, CmdDetails)
		if err == nil {
			err = tgCtx.Send(respTxt, m, telebot.ModeHTML)
		}
		return
	}
}

func ListOnGroupStartHandlerFunc(clientAwk api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, CmdStart)
		if err == nil {
			err = tgCtx.Send("Select a subscription to read in this chat:", m)
		}
		return
	}
}

func listButtons(groupIdCtx context.Context, userId string, clientAwk api.Client, btnCmd string) (m *telebot.ReplyMarkup, err error) {
	var subIds []string
	subIds, err = clientAwk.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
	if err == nil {
		m = &telebot.ReplyMarkup{}
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
				Data: fmt.Sprintf("%s %s", btnCmd, subId),
			})
			rows = append(rows, row)
		}
		m.Inline(rows...)
	}
	return
}
