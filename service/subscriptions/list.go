package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service/chats"
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

func ListHandlerFunc(clientAwk api.Client, chatStor chats.Storage, groupId string) telebot.HandlerFunc {
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
		m, err = listButtons(groupIdCtx, userId, clientAwk, chatStor, tgCtx.Chat().ID, CmdDetails)
		if err == nil {
			err = tgCtx.Send(respTxt, m, telebot.ModeHTML)
		}
		return
	}
}

func ListOnGroupStartHandlerFunc(clientAwk api.Client, chatStor chats.Storage, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, chatStor, tgCtx.Chat().ID, CmdStart)
		if err == nil {
			err = tgCtx.Send("Select a subscription to read in this chat:", m)
		}
		return
	}
}

func listButtons(
	groupIdCtx context.Context,
	userId string,
	clientAwk api.Client,
	chatStor chats.Storage,
	chatId int64,
	btnCmd string,
) (m *telebot.ReplyMarkup, err error) {
	var subIds []string
	subIds, err = clientAwk.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
	if err == nil {
		m = &telebot.ReplyMarkup{}
		var sub subscription.Data
		var rows []telebot.Row
		for _, subId := range subIds {
			sub, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
			var subLinked bool
			var subLinkedHere bool
			if err == nil {
				var c chats.Chat
				c, err = chatStor.GetSubscriptionLink(groupIdCtx, subId)
				if err == nil {
					subLinked = true
					if c.Key.Id == chatId {
						subLinkedHere = true
					}
				}
				err = nil
			}
			if err == nil {
				descr := sub.Description
				if subLinkedHere {
					descr += " ✓"
				} else if subLinked {
					descr += " 🔗"
				}
				now := time.Now().UTC()
				switch {
				case sub.Expires.IsZero(): // never expires
					descr += " ∞"
				case sub.Expires.Before(now):
					descr += " ⚠"
				case sub.Expires.Sub(now) < 168*time.Hour: // expires earlier than in 1 week
					descr += " ⏳"
				}
				btn := telebot.Btn{
					Text: descr,
				}
				if btnCmd == CmdStart && subLinkedHere {
					btn.Data = fmt.Sprintf("%s %s", CmdStop, subId)
				} else {
					btn.Data = fmt.Sprintf("%s %s", btnCmd, subId)
				}
				row := m.Row(btn)
				rows = append(rows, row)
			}
			if err != nil {
				break
			}
		}
		m.Inline(rows...)
	}
	return
}
