package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/api/grpc/source/telegram"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
)

type ListHandler struct {
	SvcSrcFeeds feeds.Service
	SvcSrcTg    telegram.Service
	Log         *slog.Logger
}

const CmdFeedListAll = "feeds_all"
const CmdFeedListOwn = "feeds_own"
const CmdTgChanList = "tgchans"

const pageLimit = 10
const cmdLimit = 64

func (lh ListHandler) TelegramChannels(tgCtx telebot.Context, args ...string) (err error) {
	var cursor string
	if len(args) > 0 {
		cursor = args[0]
	}
	if err != nil {
		err = tgCtx.Send("Failed to parse telegram chat id: %s, cause: %s", args[0], err)
	}
	var page []*telegram.Channel
	if err == nil {
		page, err = lh.SvcSrcTg.List(context.TODO(), pageLimit, cursor)
	}
	if err == nil {
		//
		m := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		for _, ch := range page {
			cmd := fmt.Sprintf("%s %s", CmdTgChDetails, ch.Link)
			if len(cmd) > cmdLimit {
				rows = append(rows, m.Row(telebot.Btn{
					Text: ch.Name,
					URL:  ch.Link,
				}))
			} else {
				rows = append(rows, m.Row(telebot.Btn{
					Text: ch.Name,
					Data: cmd,
				}))
			}

		}
		if len(page) == pageLimit {
			cmdNextPage := fmt.Sprintf("%s %s", CmdTgChanList, page[len(page)-1].Link)
			if len(cmdNextPage) > cmdLimit {
				cmdNextPage = cmdNextPage[:cmdLimit]
			}
			rows = append(rows, m.Row(telebot.Btn{
				Text: "Next Page >",
				Data: cmdNextPage,
			}))
		}
		m.Inline(rows...)
		switch len(page) {
		case 0:
			err = tgCtx.Send("End of the list")
		default:
			err = tgCtx.Send("Source Telegram Channels:", m, telebot.ModeHTML)
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
	page, err = lh.SvcSrcFeeds.ListUrls(context.TODO(), filter, pageLimit, cursor)
	if err == nil {
		m := &telebot.ReplyMarkup{}
		var rows []telebot.Row
		for _, url := range page {
			var cmdData string
			switch filter {
			case nil:
				cmdData = fmt.Sprintf("%s %s", CmdFeedDetailsAny, url)
			default:
				cmdData = fmt.Sprintf("%s %s", CmdFeedDetailsOwn, url)
			}
			if len(cmdData) > cmdLimit {
				cmdData = cmdData[:cmdLimit]
			}
			rows = append(rows, m.Row(telebot.Btn{
				Text: url,
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
			err = tgCtx.Send("Source Feeds:", m)
		}
	}
	return
}
