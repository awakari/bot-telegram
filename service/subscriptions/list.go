package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const CmdPageNext = "subs_next"

const pageLimit = 10
const cmdLimit = 64

func ListHandlerFunc(clientAwk api.Client, chatStor chats.Storage, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, chatStor, tgCtx.Chat().ID, CmdDetails, "")
		if err == nil {
			err = tgCtx.Send("Select a subscription to see the details and available actions:", m, telebot.ModeHTML)
		}
		return
	}
}

func ListOnGroupStartHandlerFunc(clientAwk api.Client, chatStor chats.Storage, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, chatStor, tgCtx.Chat().ID, CmdStart, "")
		if err == nil {
			err = tgCtx.Send("Select a subscription to read in this chat:", m)
		}
		return
	}
}

func PageNext(clientAwk api.Client, chatStor chats.Storage, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, chatStor, tgCtx.Chat().ID, args[0], args[1])
		if err == nil {
			err = tgCtx.Send("Select a subscription to see the details and available actions:", m, telebot.ModeHTML)
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
	cursor string,
) (m *telebot.ReplyMarkup, err error) {
	var subIds []string
	subIds, err = clientAwk.SearchSubscriptions(groupIdCtx, userId, pageLimit, "")
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
		if len(subIds) == pageLimit {
			cmdData := fmt.Sprintf("%s %s %s", CmdPageNext, btnCmd, subIds[len(subIds)-1])
			if len(cmdData) > cmdLimit {
				cmdData = cmdData[:cmdLimit]
			}
			rows = append(rows, m.Row(telebot.Btn{
				Text: "Next Page >",
				Data: cmdData,
			}))
		}
		m.Inline(rows...)
	}
	return
}
