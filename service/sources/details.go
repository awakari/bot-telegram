package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"strings"
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
const CmdTgChDetails = "tgch"

const fmtFeedDetails = `Feed Details:
%s
Daily Messages Limit: <pre>%d</pre>
Limit Expires: <pre>%s</pre>
Update Period: <pre>%s</pre>
Next Update: <pre>%s</pre>
Last Message: <pre>%s</pre>
Total Messages: <pre>%d</pre>
`
const fmtTgChDetails = `Telegram Channel Details:
Title: %s
<a href="%s">Link</a>
Description: %s
Daily Messages Limit: <pre>%d</pre>
Limit Expires: <pre>%s</pre>
`
const tgChLinkPrefix = "https://t.me/"
const tgChLinkPrefixPrivate = "https://t.me/c/"

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
		ctxGroupId := metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", dh.CfgFeeds.GroupId)
		url = feed.Url
		l, err = dh.ClientAwk.ReadUsageLimit(ctxGroupId, url, usage.SubjectPublishEvents)
		fmt.Printf("limit=%+v, err=%s\n", l, err)
	}
	//
	if err == nil {
		txtSummary := url
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
			l.Count,
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

func (dh DetailsHandler) GetTelegramChannel(tgCtx telebot.Context, args ...string) (err error) {
	//
	url := args[0]
	//
	var title string
	var descr string
	if strings.HasPrefix(url, tgChLinkPrefix) && !strings.HasPrefix(url, tgChLinkPrefixPrivate) {
		chatName := url[len(tgChLinkPrefix):]
		var chat *telebot.Chat
		chat, err = tgCtx.Bot().ChatByUsername(fmt.Sprintf("@%s", chatName))
		switch err {
		case nil:
			title = chat.Title
			descr = chat.Description
		default:
			dh.Log.Warn(fmt.Sprintf("Failed to resolve the chat by username: %s, cause: %s", chatName, err))
			title = "N/A (error)"
			descr = "N/A (error)"
		}
	} else {
		title = "N/A (private)"
		descr = "N/A (private)"
	}
	//
	var l usage.Limit
	var expiresTxt string
	ctxGroupId := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", dh.CfgFeeds.GroupId)
	l, err = dh.ClientAwk.ReadUsageLimit(ctxGroupId, url, usage.SubjectPublishEvents)
	switch {
	case err != nil:
		expiresTxt = "N/A (error)"
	case l.Expires.Unix() <= 0:
		expiresTxt = "never"
	default:
		expiresTxt = l.Expires.Format(time.RFC3339)
	}
	//
	detailsTxt := fmt.Sprintf(fmtTgChDetails, title, url, descr, l.Count, expiresTxt)
	err = tgCtx.Send(detailsTxt, telebot.ModeHTML)
	//
	return
}
