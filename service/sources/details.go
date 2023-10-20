package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/client-sdk-go/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

type DetailsHandler struct {
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
	var feed *feeds.Feed
	feed, err = dh.SvcSrcFeeds.Read(context.TODO(), url)
	switch {
	case status.Code(err) == codes.NotFound:
		dh.Log.Warn(fmt.Sprintf("Feed not found, URL may be truncated: %s", url))
		var urls []string
		urls, err = dh.SvcSrcFeeds.List(context.TODO(), filter, 1, url)
		dh.Log.Debug(fmt.Sprintf("List feeds with cursor \"%s\" results: %+v, %s", url, urls, err))
		if err == nil && len(urls) > 0 {
			feed, err = dh.SvcSrcFeeds.Read(context.TODO(), url)
		}
	}
	//
	if err == nil {
		txtSummary := feed.Url
		if feed.UserId != "" {
			txtSummary += fmt.Sprintf("\n<a href=\"tg://user?id=%s\">Owner</a>", feed.UserId)
		}
		var txtExpires string
		switch {
		case feed.Expires.Seconds <= 0:
			txtExpires = "never"
		default:
			txtExpires = feed.Expires.AsTime().Format(time.RFC3339)
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
		)
		err = tgCtx.Send(txt, telebot.ModeHTML)
	}
	//
	return
}
