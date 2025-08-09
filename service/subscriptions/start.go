package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/interests"
	"github.com/awakari/bot-telegram/api/http/subscriptions"
	"github.com/awakari/bot-telegram/model/interest"
	"github.com/awakari/bot-telegram/model/usage"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/limits"
	"github.com/awakari/bot-telegram/util"
	"gopkg.in/telebot.v3"
	"html"
	"time"
)

const CmdStart = "sub_start"
const ReqStart = "sub_start"
const MsgFmtChatLinked = "Subscribed to the interest %s in this chat. " +
	"New results will appear here with a minimum interval of %s. " +
	"To manage own interests use the <a href=\"https://awakari.com/login.html\" target=\"blank\">app</a>."

func StartHandler(
	svcInterests interests.Service,
	svcSubs subscriptions.Service,
	svcLimits limits.Service,
	urlCallbackBase string,
	groupId string,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		switch len(args) {
		case 1:
			subId := args[0]
			err = StartIntervalRequest(tgCtx, subId)
		case 3:
			var interval time.Duration
			interval, err = time.ParseDuration(args[2])
			if err != nil {
				err = errors.New(fmt.Sprintf("invalid interval value: %s", args[2]))
			}
			if interval < 0 {
				err = errors.New("error: interval should not be negative")
			}
			if err == nil {
				subId := args[1]
				err = Start(tgCtx, svcInterests, svcSubs, svcLimits, urlCallbackBase, subId, groupId, interval)
			}
		default:
			err = errors.New(fmt.Sprintf("invalid response: expected 1 or 3 arguments, got %d: %+v", len(args), args))
		}
		return
	}
}

func StartIntervalRequest(tgCtx telebot.Context, interestId string) (err error) {
	_ = tgCtx.Send("Reply a minimum notification interval to the command below, for example `0`, `1s`, `2m` or `3h`:", telebot.ModeMarkdownV2)
	err = tgCtx.Send(ReqStart+" "+interestId, &telebot.ReplyMarkup{
		ForceReply:  true,
		Placeholder: "0",
	})
	return
}

func Start(
	tgCtx telebot.Context,
	svcInterests interests.Service,
	svcSubs subscriptions.Service,
	svcLimits limits.Service,
	urlCallbackBase string,
	interestId string,
	groupId string,
	interval time.Duration,
) (err error) {
	ctx := context.TODO()
	var userId string
	switch tgCtx.Sender() {
	case nil:
		userId = fmt.Sprintf(service.FmtNamePub, tgCtx.Chat().Username) // public channel post has no sender
	default:
		userId = util.SenderToUserId(tgCtx)
	}
	urlCallback := subscriptions.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID, userId)
	err = svcSubs.Subscribe(ctx, interestId, groupId, userId, urlCallback, interval)
	if errors.Is(err, chats.ErrAlreadyExists) {
		// might be not an error, so try to re-link the subscription
		err = svcSubs.Unsubscribe(ctx, interestId, groupId, userId, urlCallback)
		if err != nil {
			urlCallbackOld := subscriptions.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID, "")
			err = svcSubs.Unsubscribe(ctx, interestId, groupId, userId, urlCallbackOld)
		}
		if err == nil {
			err = svcSubs.Subscribe(ctx, interestId, groupId, userId, urlCallback, interval)
		}
	}
	switch {
	case err == nil:
		var subData interest.Data
		subData, err = svcInterests.Read(context.TODO(), groupId, userId, interestId)
		var subDescr string
		switch err {
		case nil:
			subDescr = "named \"" + html.EscapeString(subData.Description) + "\""
		default:
			// it's still ok to follow an interest created by a non-telegram user in Awakari web UI
			subDescr = "id: <code>" + interestId + "</code>"
		}
		err = tgCtx.Send(fmt.Sprintf(MsgFmtChatLinked, subDescr, interval), telebot.ModeHTML, telebot.NoPreview)
	case errors.Is(err, subscriptions.ErrPermitExhausted):
		var l usage.Limit
		l, err = svcLimits.Get(ctx, groupId, userId, usage.SubjectSubscriptions)
		switch err {
		case nil:
			switch {
			case l.Count < 5:
				err = tgCtx.Send(fmt.Sprintf("Subscription count limit reached: %d", l.Count), &telebot.ReplyMarkup{
					InlineKeyboard: [][]telebot.InlineButton{
						{
							telebot.InlineButton{
								Text: "Increase to 5",
								URL:  "https://t.me/tribute/app?startapp=svd8",
							},
						},
					},
				})
			case l.Count < 10:
				err = tgCtx.Send(fmt.Sprintf("Subscription count limit reached: %d", l.Count), &telebot.ReplyMarkup{
					InlineKeyboard: [][]telebot.InlineButton{
						{
							telebot.InlineButton{
								Text: "Increase to 10",
								URL:  "https://t.me/tribute/app?startapp=sv5Q",
							},
						},
					},
				})
			case l.Count < 20:
				err = tgCtx.Send(fmt.Sprintf("Subscription count limit reached: %d", l.Count), &telebot.ReplyMarkup{
					InlineKeyboard: [][]telebot.InlineButton{
						{
							telebot.InlineButton{
								Text: "Increase to 20",
								URL:  "https://t.me/tribute/app?startapp=svaR",
							},
						},
					},
				})
			default:
				err = tgCtx.Send(fmt.Sprintf("Subscription count limit reached"))
			}
		default:
			_ = tgCtx.Send(fmt.Sprintf("Subscription count limit reached"))
		}
	default:
		err = tgCtx.Send("Unexpected failure", telebot.ModeHTML, telebot.NoPreview)
	}
	return
}
