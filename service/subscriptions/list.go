package subscriptions

import (
	"context"
	"fmt"
	protoInterests "github.com/awakari/bot-telegram/api/grpc/interests"
	"github.com/awakari/bot-telegram/api/http/interests"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/model"
	"github.com/awakari/bot-telegram/model/interest"
	"github.com/awakari/bot-telegram/service"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"math"
	"strconv"
)

const CmdPageNext = "subs_next"
const CmdPageNextFollowing = "following_next"

func ListOnGroupStartHandlerFunc(svcInterests interests.Service, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		cursor := interest.Cursor{}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupId, userId, svcInterests, svcReader, tgCtx.Chat().ID, CmdStart, cursor, false, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Own interests list. Select one or more to follow in this chat:", m)
		}
		return
	}
}

func ListPublicHandlerFunc(svcInterests interests.Service, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		cursor := interest.Cursor{
			Id:        "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
			Followers: math.MaxInt64,
		}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupId, userId, svcInterests, svcReader, tgCtx.Chat().ID, CmdStart, cursor, true, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Available interests list. Select one or more to follow in this chat:", m)
		}
		return
	}
}

func PageNext(svcInterests interests.Service, svcReader reader.Service, groupId, urlCallBackBase string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var cursor interest.Cursor
		var public bool
		if len(args) > 2 {
			cursor.Id = args[1]
			cursor.Followers, _ = strconv.ParseInt(args[2], 10, 64)
		}
		if len(args) > 3 {
			public = true
		}
		var m *telebot.ReplyMarkup
		m, err = listButtons(groupId, userId, svcInterests, svcReader, tgCtx.Chat().ID, args[0], cursor, public, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Interests list page:", m, telebot.ModeHTML)
		}
		return
	}
}

func listButtons(
	groupId string,
	userId string,
	svcInterests interests.Service,
	svcReader reader.Service,
	chatId int64,
	btnCmd string,
	cursor interest.Cursor,
	public bool,
	urlCallBackBase string,
) (m *telebot.ReplyMarkup, err error) {
	var page []*protoInterests.Interest
	q := interest.Query{
		Limit: service.PageLimit,
	}
	if public {
		q.Order = interest.OrderDesc
		q.Public = true
		q.Sort = interest.SortFollowers
	}
	page, err = svcInterests.Search(context.TODO(), groupId, userId, q, cursor)
	if err == nil {
		m = &telebot.ReplyMarkup{}
		var rows []telebot.Row
		var lastFollowers int64
		for _, interest := range page {
			var subLinkedHere bool
			lastFollowers = interest.Followers
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), model.KeyGroupId, groupId)
			_, err = svcReader.GetCallback(groupIdCtx, interest.Id, reader.MakeCallbackUrl(urlCallBackBase, chatId))
			if err == nil {
				subLinkedHere = true
			}
			err = nil
			if err == nil {
				descr := interest.Description
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
				if interest.Public {
					btn.Text = "👁 " + btn.Text
				}
				if btnCmd == CmdStart && subLinkedHere {
					btn.Data = fmt.Sprintf("%s %s", CmdStop, interest)
				} else {
					btn.Data = fmt.Sprintf("%s %s", btnCmd, interest)
				}
				row := m.Row(btn)
				rows = append(rows, row)
			}
			if err != nil {
				break
			}
		}
		if len(page) == service.PageLimit {
			cmdData := fmt.Sprintf("%s %s %s %d", CmdPageNext, btnCmd, page[len(page)-1], lastFollowers)
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

func ListFollowing(svcInterests interests.Service, svcReader reader.Service, groupId, urlCallBackBase string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), model.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var m *telebot.ReplyMarkup
		m, err = listButtonsFollowing(groupIdCtx, groupId, userId, svcInterests, svcReader, tgCtx.Chat().ID, "", urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("List of interests you following in this chat. Select any to stop:", m)
		}
		return
	}
}

func PageNextFollowing(svcInterests interests.Service, svcReader reader.Service, groupId, urlCallBackBase string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), model.KeyGroupId, groupId)
		userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
		var cursor string
		if len(args) > 0 {
			cursor = args[0]
		}
		var m *telebot.ReplyMarkup
		m, err = listButtonsFollowing(groupIdCtx, groupId, userId, svcInterests, svcReader, tgCtx.Chat().ID, cursor, urlCallBackBase)
		if err == nil {
			err = tgCtx.Send("Interests list page:", m, telebot.ModeHTML)
		}
		return
	}
}

func listButtonsFollowing(
	groupIdCtx context.Context,
	groupId, userId string,
	svcInterests interests.Service,
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
		var sub interest.Data
		var rows []telebot.Row
		for _, interestId := range interestIds {
			var descr string
			sub, err = svcInterests.Read(groupIdCtx, groupId, userId, interestId)
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
