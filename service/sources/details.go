package sources

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/client-sdk-go/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type DetailsHandler struct {
	CfgFeeds    config.FeedsConfig
	CfgTelegram config.TelegramConfig
	ClientAwk   api.Client
	SvcSrcFeeds feeds.Service
	Log         *slog.Logger
}

const CmdFeedDetailsAny = "feed_any"
const CmdFeedDetailsOwn = "feed_own"
const CmdTgChDetails = "tgch"

const fmtFeedDetails = `Feed Details:
%s
Update Period: <pre>%s</pre>
Next Update: <pre>%s</pre>
Last Message: <pre>%s</pre>
`
const fmtTgChDetails = `Source Telegram Channel Details:
%s
Title: %s
Description: %s
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
		urls, err = dh.SvcSrcFeeds.ListUrls(context.TODO(), filter, 1, url)
		dh.Log.Debug(fmt.Sprintf("List feeds with cursor \"%s\" results: %+v, %s", url, urls, err))
		if err == nil && len(urls) > 0 {
			feed, err = dh.SvcSrcFeeds.Read(context.TODO(), urls[0])
		}
	}
	url = feed.Url
	//
	if err == nil {
		txtSummary := url
		if feed.UserId != "" {
			txtSummary += fmt.Sprintf("\nAdded by <a href=\"tg://user?id=%s\">the user</a> from <pre>%s</pre>", feed.UserId, feed.GroupId)
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
			feed.UpdatePeriod.AsDuration(),
			feed.NextUpdate.AsTime().Format(time.RFC3339),
			txtItemLast,
		)
		err = tgCtx.Send(txt, telebot.ModeHTML)
	}
	//
	return
}

func (dh DetailsHandler) GetTelegramChannel(tgCtx telebot.Context, args ...string) (err error) {
	url := args[0]
	var chatLink string
	var title string
	var descr string
	if strings.HasPrefix(url, tgChLinkPrefix) && !strings.HasPrefix(url, tgChLinkPrefixPrivate) {
		chatLink = fmt.Sprintf("@%s", url[len(tgChLinkPrefix):])
		var chat *telebot.Chat
		chat, err = tgCtx.Bot().ChatByUsername(chatLink)
		switch err {
		case nil:
			title = chat.Title
			descr = chat.Description
		default:
			dh.Log.Warn(fmt.Sprintf("Failed to resolve the chat by username: %s, cause: %s", chatLink, err))
			title = "N/A (error)"
			descr = "N/A (error)"
		}
	} else {
		chatLink = url
		title = "N/A (private)"
		descr = "N/A (private)"
	}
	//
	detailsTxt := fmt.Sprintf(
		fmtTgChDetails,
		chatLink,
		title,
		descr,
	)
	err = tgCtx.Send(detailsTxt, telebot.ModeHTML)
	if err != nil {
		detailsTxt = fmt.Sprintf(
			fmtTgChDetails,
			chatLink,
			title,
			descr,
		)
		err = tgCtx.Send(detailsTxt)
	}
	return
}
