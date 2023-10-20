package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
	"strconv"
)

const CmdFeedListAll = "src_feed_list_all"
const CmdFeedListOwn = "src_feed_list_own"
const CmdFeedDetails = "src_feed_details"

const pageLimit = 10

type ListHandler struct {
	ClientAwk   api.Client
	SvcSrcFeeds feeds.Service
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

			rows = append(rows, m.Row(telebot.Btn{
				Text: feedUrl,
				Data: fmt.Sprintf("%s %s", CmdFeedDetails /*feedUrl*/, "TODO"),
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
			rows = append(rows, m.Row(telebot.Btn{
				Text: "Next Page >",
				Data: fmt.Sprintf("%s %s", cmdList /*page[len(page)-1]*/, "TODO"),
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
