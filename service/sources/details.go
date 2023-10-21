package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

type DetailsHandler struct {
	CfgFeeds    config.FeedsConfig
	ClientAwk   api.Client
	SvcSrcFeeds feeds.Service
	Log         *slog.Logger
}

const CmdFeedDetailsAny = "feed_any"
const CmdFeedDetailsOwn = "feed_own"

const fmtFeedDetails = `Feed Details:
%s
Expires: <pre>%s</pre>
Update Period: <pre>%s</pre>
Next Update: <pre>%s</pre>
Last Message: <pre>%s</pre>
Total Messages: <pre>%d</pre>
Messages Daily Limit: <pre>%d</pre>
`

func (dh DetailsHandler) GetFeedAny(tgCtx telebot.Context, args ...string) (err error) {
	err = dh.getFeed(tgCtx, args[0], nil)
	return
}

func (dh DetailsHandler) GetFeedOwn(tgCtx telebot.Context, args ...string) (err error) {
	filterOwn := &feeds.Filter{
		UserId: strconv.FormatInt(tgCtx.Sender().ID, 10),
	}
	err = dh.getFeed(tgCtx, args[0], filterOwn)
	return
}
func (dh DetailsHandler) getFeed(tgCtx telebot.Context, url string, filter *feeds.Filter) (err error) {
	//
	ctx := context.TODO()
	//
	var feed *feeds.Feed
	feed, err = dh.SvcSrcFeeds.Read(ctx, url)
	switch {
	case status.Code(err) == codes.NotFound:
		dh.Log.Warn(fmt.Sprintf("Feed not found, URL may be truncated: %s", url))
		var urls []string
		urls, err = dh.SvcSrcFeeds.List(context.TODO(), filter, 1, url)
		dh.Log.Debug(fmt.Sprintf("List feeds with cursor \"%s\" results: %+v, %s", url, urls, err))
		if err == nil && len(urls) > 0 {
			feed, err = dh.SvcSrcFeeds.Read(context.TODO(), urls[0])
		}
	}
	//
	var l usage.Limit
	if err == nil {
		ctxGroupId := context.WithValue(ctx, "x-awakari-group-id", dh.CfgFeeds.GroupId)
		url = feed.Url
		l, _ = dh.ClientAwk.ReadUsageLimit(ctxGroupId, url, usage.SubjectPublishEvents)
	}
	//
	if err == nil {
		txtSummary := url
		if feed.UserId != "" {
			txtSummary += fmt.Sprintf("\n<a href=\"tg://user?id=%s\">Owner</a>", feed.UserId)
		}
		m := &telebot.ReplyMarkup{}
		var txtExpires string
		switch {
		case feed.Expires.Seconds <= 0:
			txtExpires = "never"
		default:
			txtExpires = feed.Expires.AsTime().Format(time.RFC3339)
			cmdExtend := fmt.Sprintf("%s %s", CmdExtend, url)
			if len(cmdExtend) > cmdLimit {
				cmdExtend = cmdExtend[:cmdLimit]
			}
			m.Inline(m.Row(telebot.Btn{
				Text: "▲ Extend",
				Data: fmt.Sprintf(cmdExtend),
			}))
		}
		var txtItemLast string
		switch {
		case feed.ItemLast.Seconds <= 0:
			txtItemLast = "never"
		default:
			txtItemLast = feed.ItemLast.AsTime().Format(time.RFC3339)
		}
		txt := fmt.Sprintf(
			fmtFeedDetails,
			txtSummary,
			txtExpires,
			feed.UpdatePeriod.AsDuration(),
			feed.NextUpdate.AsTime().Format(time.RFC3339),
			txtItemLast,
			feed.ItemCount,
			l.Count,
		)
		err = tgCtx.Send(txt, m, telebot.ModeHTML)
	}
	//
	return
}
