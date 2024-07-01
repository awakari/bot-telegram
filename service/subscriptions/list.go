package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
)

const CmdPageNext = "subs_next"

func ListOnGroupStartHandlerFunc(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, CmdStart, "", urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Own interests list. Select one or more to follow/unfollow in this chat:", m)
		}
		return
	}
}

func PageNext(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var cursor string
		if len(args) > 1 {
			cursor = args[1]
		}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, args[0], cursor, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Own interests list page:", m, telebot.ModeHTML)
		}
		return
	}
}

func listButtons(
	groupIdCtx context.Context,
	userId string,
	clientAwk api.Client,
	svcReader reader.Service,
	chatId int64,
	btnCmd string,
	cursor string,
	urlCallBackBase string,
) (m *telebot.ReplyMarkup, err error) {
	var subIds []string
	subIds, err = clientAwk.SearchSubscriptions(
		groupIdCtx,
		userId,
		subscription.Query{
			Limit: service.PageLimit,
		},
		subscription.Cursor{
			Id: cursor,
		},
	)
	if err == nil {
		m = &telebot.ReplyMarkup{}
		var sub subscription.Data
		var rows []telebot.Row
		for _, subId := range subIds {
			sub, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
			var subLinkedHere bool
			if err == nil {
				_, err = svcReader.GetCallback(groupIdCtx, subId, reader.MakeCallbackUrl(urlCallBackBase, chatId))
				if err == nil {
					subLinkedHere = true
				}
				err = nil
			}
			if err == nil {
				descr := sub.Description
				if subLinkedHere {
					descr += " ✓"
				}
				// TODO: uncomment the code below when payments are in use
				//now := time.Now().UTC()
				//switch {
				//case sub.Expires.IsZero(): // never expires
				//	descr += " ∞"
				//case sub.Expires.Before(now):
				//	descr += " ⚠"
				//case sub.Expires.Sub(now) < 168*time.Hour: // expires earlier than in 1 week
				//	descr += " ⏳"
				//}
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
		if len(subIds) == service.PageLimit {
			cmdData := fmt.Sprintf("%s %s %s", CmdPageNext, btnCmd, subIds[len(subIds)-1])
			if len(cmdData) > service.CmdLimit {
				cmdData = cmdData[:service.CmdLimit]
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
