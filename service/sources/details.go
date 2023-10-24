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

Daily Messages Limit: <pre>%d</pre>
Limit Expires: <pre>%s</pre>
Count Today: <pre>%d</pre>
Count Total: <pre>%d</pre>
Since: <pre>%s</pre>

Update Period: <pre>%s</pre>
Next Update: <pre>%s</pre>
Last Message: <pre>%s</pre>
`
const fmtTgChDetails = `Source Telegram Channel Details:
%s

Daily Messages Limit: %s
Limit Expires: %s
Count Today: %s
Count Total: %s
Since: %s

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
		urls, err = dh.SvcSrcFeeds.List(context.TODO(), filter, 1, url)
		dh.Log.Debug(fmt.Sprintf("List feeds with cursor \"%s\" results: %+v, %s", url, urls, err))
		if err == nil && len(urls) > 0 {
			feed, err = dh.SvcSrcFeeds.Read(context.TODO(), urls[0])
		}
	}
	//
	url = feed.Url
	var l usage.Limit
	if err == nil {
		ctxGroupId := metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", dh.CfgFeeds.GroupId)
		url = feed.Url
		l, err = dh.ClientAwk.ReadUsageLimit(ctxGroupId, url, usage.SubjectPublishEvents)
	}
	var u usage.Usage
	if err == nil {
		ctxGroupId := metadata.AppendToOutgoingContext(ctx, "x-awakari-group-id", dh.CfgFeeds.GroupId)
		url = feed.Url
		u, err = dh.ClientAwk.ReadUsage(ctxGroupId, url, usage.SubjectPublishEvents)
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
			u.Count,
			u.CountTotal,
			u.Since.Format(time.RFC3339),
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
	var countLimitTxt string
	var expiresTxt string
	var countTodayTxt string
	var countTotalTxt string
	var since string
	if strings.HasPrefix(url, tgChLinkPrefix) && !strings.HasPrefix(url, tgChLinkPrefixPrivate) {
		chatLink = fmt.Sprintf("@%s", url[len(tgChLinkPrefix):])
		var chat *telebot.Chat
		chat, err = tgCtx.Bot().ChatByUsername(chatLink)
		switch err {
		case nil:
			title = chat.Title
			descr = chat.Description
			ctxGroupId := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", dh.CfgTelegram.GroupId)
			srcUserId := strconv.FormatInt(chat.ID, 10)
			var l usage.Limit
			l, err = dh.ClientAwk.ReadUsageLimit(ctxGroupId, srcUserId, usage.SubjectPublishEvents)
			switch {
			case err != nil:
				countLimitTxt = "N/A (error)"
				expiresTxt = "N/A (error)"
			case l.Expires.Unix() <= 0:
				countLimitTxt = strconv.FormatInt(l.Count, 10)
				expiresTxt = "never"
			default:
				countLimitTxt = strconv.FormatInt(l.Count, 10)
				expiresTxt = l.Expires.Format(time.RFC3339)
			}
			var u usage.Usage
			u, err = dh.ClientAwk.ReadUsage(ctxGroupId, srcUserId, usage.SubjectPublishEvents)
			switch {
			case err != nil:
				countTodayTxt = "N/A (error)"
				countTotalTxt = "N/A (error)"
				since = "N/A (error)"
			default:
				countTodayTxt = strconv.FormatInt(u.Count, 10)
				countTotalTxt = strconv.FormatInt(u.CountTotal, 10)
				since = u.Since.Format(time.RFC3339)
			}
		default:
			dh.Log.Warn(fmt.Sprintf("Failed to resolve the chat by username: %s, cause: %s", chatLink, err))
			title = "N/A (error)"
			descr = "N/A (error)"
			countLimitTxt = "N/A (error)"
			expiresTxt = "N/A (error)"
			since = "N/A (error)"
		}
	} else {
		chatLink = url
		title = "N/A (private)"
		descr = "N/A (private)"
		countLimitTxt = "N/A (private)"
		expiresTxt = "N/A (private)"
		since = "N/A (private)"
	}
	//
	detailsTxt := fmt.Sprintf(
		fmtTgChDetails,
		chatLink,
		fmt.Sprintf("<pre>%s</pre>", countLimitTxt),
		fmt.Sprintf("<pre>%s</pre>", expiresTxt),
		fmt.Sprintf("<pre>%s</pre>", countTodayTxt),
		fmt.Sprintf("<pre>%s</pre>", countTotalTxt),
		fmt.Sprintf("<pre>%s</pre>", since),
		title,
		descr,
	)
	err = tgCtx.Send(detailsTxt, telebot.ModeHTML)
	if err != nil {
		detailsTxt = fmt.Sprintf(
			fmtTgChDetails,
			chatLink,
			countLimitTxt,
			expiresTxt,
			countTodayTxt,
			countTotalTxt,
			since,
			title,
			descr,
		)
		err = tgCtx.Send(detailsTxt)
	}
	return
}
