package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/api/grpc/source/telegram"
	"gopkg.in/telebot.v3"
	"math"
	"strconv"
)

type ListHandler struct {
	SvcSrcFeeds feeds.Service
	SvcSrcTg    telegram.Service
}

const CmdFeedListAll = "feeds_all"
const CmdFeedListOwn = "feeds_own"
const CmdTgChanList = "tgchans"

const pageLimit = 10
const cmdLimit = 64

func (lh ListHandler) TelegramChannels(tgCtx telebot.Context, args ...string) (err error) {
	cursor := int64(math.MinInt64)
	if len(args) > 0 {
		cursor, err = strconv.ParseInt(args[0], 10, 64)
	}
	if err != nil {
		err = tgCtx.Send("Failed to parse telegram chat id: %s, cause: %s", args[0], err)
	}
	var page []int64
	if err == nil {
		page, err = lh.SvcSrcTg.List(context.TODO(), pageLimit, cursor)
	}
	if err == nil {
		//
		pageTxt := ""
		var chat *telebot.Chat
		bot := tgCtx.Bot()
		for _, chatId := range page {
			chat, err = bot.ChatByID(chatId)
			if err == nil {
				pageTxt += fmt.Sprintf("<a href=\"https://t.me/%s\">%s</a>\n", chat.Username, chat.Title)
			}
		}
		//
		m := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		if len(page) == pageLimit {
			rows = append(rows, m.Row(telebot.Btn{
				Text: "Next Page >",
				Data: fmt.Sprintf("%s %d", CmdTgChanList, page[len(page)-1]),
			}))
		}
		m.Inline(rows...)
		switch len(page) {
		case 0:
			err = tgCtx.Send("End of the list")
		default:
			err = tgCtx.Send(pageTxt, m)
		}
	}
	return
}

func (lh ListHandler) FeedListAll(tgCtx telebot.Context, args ...string) (err error) {
	err = lh.feedList(tgCtx, nil, args...)
	return
}

func (lh ListHandler) FeedListOwn(tgCtx telebot.Context, args ...string) (err error) {
	filterOwn := &feeds.Filter{
		UserId: strconv.FormatInt(tgCtx.Sender().ID, 10),
	}
	err = lh.feedList(tgCtx, filterOwn, args...)
	return
}

func (lh ListHandler) feedList(tgCtx telebot.Context, filter *feeds.Filter, args ...string) (err error) {
	var cursor string
	if len(args) > 0 {
		cursor = args[0]
	}
	var page []string
	page, err = lh.SvcSrcFeeds.List(context.TODO(), filter, pageLimit, cursor)
	if err == nil {
		m := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		for _, feedUrl := range page {
			var cmdData string
			switch filter {
			case nil:
				cmdData = fmt.Sprintf("%s %s", CmdFeedDetailsAny, feedUrl)
			default:
				cmdData = fmt.Sprintf("%s %s", CmdFeedDetailsOwn, feedUrl)
			}
			if len(cmdData) > cmdLimit {
				cmdData = cmdData[:cmdLimit]
			}
			rows = append(rows, m.Row(telebot.Btn{
				Text: feedUrl,
				Data: cmdData,
			}))
		}
		if len(page) == pageLimit {
			var cmdList string
			switch filter {
			case nil:
				cmdList = CmdFeedListAll
			default:
				cmdList = CmdFeedListOwn
			}
			cmdData := fmt.Sprintf("%s %s", cmdList, page[len(page)-1])
			if len(cmdData) > cmdLimit {
				cmdData = cmdData[:cmdLimit]
			}
			rows = append(rows, m.Row(telebot.Btn{
				Text: "Next Page >",
				Data: cmdData,
			}))
		}
		m.Inline(rows...)
		switch len(page) {
		case 0:
			err = tgCtx.Send("End of the list")
		default:
			err = tgCtx.Send("Feeds page:", m)
		}
	}
	return
}
