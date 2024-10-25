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
	"math"
	"strconv"
)

const CmdPageNext = "subs_next"
const CmdPageNextFollowing = "following_next"

func ListOnGroupStartHandlerFunc(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		cursor := subscription.Cursor{}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, CmdStart, cursor, false, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Own interests list. Select one or more to follow in this chat:", m)
		}
		return
	}
}

func ListPublicHandlerFunc(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		cursor := subscription.Cursor{
			Id:        "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
			Followers: math.MaxInt64,
		}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, CmdStart, cursor, true, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Available interests list. Select one or more to follow in this chat:", m)
		}
		return
	}
}

func PageNext(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var cursor subscription.Cursor
		var public bool
		if len(args) > 2 {
			cursor.Id = args[1]
			cursor.Followers, _ = strconv.ParseInt(args[2], 10, 64)
		}
		if len(args) > 3 {
			public = true
		}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, args[0], cursor, public, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Interests list page:", m, telebot.ModeHTML)
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
	cursor subscription.Cursor,
	public bool,
	urlCallBackBase string,
) (m *telebot.ReplyMarkup, err error) {
	var subIds []string
	q := subscription.Query{
		Limit: service.PageLimit,
	}
	if public {
		q.Order = subscription.OrderDesc
		q.Public = true
		q.Sort = subscription.SortFollowers
	}
	subIds, err = clientAwk.SearchSubscriptions(groupIdCtx, userId, q, cursor)
	if err == nil {
		m = &telebot.ReplyMarkup{}
		var sub subscription.Data
		var rows []telebot.Row
		var lastFollowers int64
		for _, subId := range subIds {
			sub, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
			var subLinkedHere bool
			if err == nil {
				lastFollowers = sub.Followers
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
				if sub.Public {
					btn.Text = "👁 " + btn.Text
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
			cmdData := fmt.Sprintf("%s %s %s %d", CmdPageNext, btnCmd, subIds[len(subIds)-1], lastFollowers)
			if public {
				cmdData += " public"
			}
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

func ListFollowing(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var m *telebot.ReplyMarkup
		m, err = listButtonsFollowing(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, "", urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("List of interests you following in this chat. Select any to stop:", m)
		}
		return
	}
}

func PageNextFollowing(clientAwk api.Client, svcReader reader.Service, groupId, urlCallBackBase string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var cursor string
		if len(args) > 0 {
			cursor = args[0]
		}
		var m *telebot.ReplyMarkup
		m, err = listButtonsFollowing(groupIdCtx, userId, clientAwk, svcReader, tgCtx.Chat().ID, cursor, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Interests list page:", m, telebot.ModeHTML)
		}
		return
	}
}

func listButtonsFollowing(
	groupIdCtx context.Context,
	userId string,
	clientAwk api.Client,
	svcReader reader.Service,
	chatId int64,
	cursor string,
	urlCallBackBase string,
) (m *telebot.ReplyMarkup, err error) {
	cbUrl := reader.MakeCallbackUrl(urlCallBackBase, chatId)
	var interestIds []string
	interestIds, err = svcReader.ListByUrl(groupIdCtx, service.PageLimit, cbUrl, cursor)
	if err == nil {
		m = &telebot.ReplyMarkup{}
		var sub subscription.Data
		var rows []telebot.Row
		for _, interestId := range interestIds {
			var descr string
			sub, err = clientAwk.ReadSubscription(groupIdCtx, userId, interestId)
			switch err {
			case nil:
				descr = sub.Description
				if sub.Public {
					descr = "👁 " + descr
				}
			default:
				descr = "ID: " + interestId
				err = nil
			}
			btn := telebot.Btn{
				Text: descr,
			}
			btn.Data = fmt.Sprintf("%s %s", CmdStop, interestId)
			row := m.Row(btn)
			rows = append(rows, row)
		}
		if len(interestIds) == service.PageLimit {
			cmdData := fmt.Sprintf("%s %s", CmdPageNextFollowing, interestIds[len(interestIds)-1])
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
