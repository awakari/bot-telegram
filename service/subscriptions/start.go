package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/interests"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/model/interest"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"gopkg.in/telebot.v3"
	"html"
	"time"
)

const CmdStart = "sub_start"
const ReqStart = "sub_start"
const MsgFmtChatLinked = "Following the interest %s in this chat. " +
	"New results will appear here with a minimum interval of %s. " +
	"To manage own interests use the <a href=\"https://awakari.com/login.html\" target=\"blank\">app</a>."

func StartHandler(
	svcInterests interests.Service,
	svcReader reader.Service,
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
				err = Start(tgCtx, svcInterests, svcReader, urlCallbackBase, subId, groupId, interval)
			}
		default:
			err = errors.New(fmt.Sprintf("invalid response: expected 1 or 3 arguments, got %d: %+v", len(args), args))
		}
		return
	}
}

func StartIntervalRequest(tgCtx telebot.Context, interestId string) (err error) {
	_ = tgCtx.Send("Reply a minimum notification interval, for example `0`, `1s`, `2m` or `3h`:", telebot.ModeMarkdownV2)
	err = tgCtx.Send(ReqStart+" "+interestId, &telebot.ReplyMarkup{
		ForceReply:  true,
		Placeholder: "0",
	})
	return
}

func Start(
	tgCtx telebot.Context,
	svcInterests interests.Service,
	svcReader reader.Service,
	urlCallbackBase string,
	subId string,
	groupId string,
	interval time.Duration,
) (err error) {
	ctx := context.TODO()
	var userId string
	switch tgCtx.Sender() {
	case nil:
		userId = fmt.Sprintf(service.FmtNamePub, tgCtx.Chat().Username) // public channel post has no sender
	default:
		userId = fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
	}
	urlCallback := reader.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID)
	err = svcReader.CreateCallback(ctx, subId, urlCallback, interval)
	switch {
	case errors.Is(err, chats.ErrAlreadyExists):
		// might be not an error, so try to re-link the subscription
		err = svcReader.DeleteCallback(ctx, subId, urlCallback)
		if err == nil {
			err = svcReader.CreateCallback(ctx, subId, urlCallback, interval)
		}
	}
	var subData interest.Data
	if err == nil {
		subData, err = svcInterests.Read(context.TODO(), groupId, userId, subId)
	}
	var subDescr string
	switch err {
	case nil:
		subDescr = "named \"" + html.EscapeString(subData.Description) + "\""
	default:
		// it's still ok to follow an interest created by a non-telegram user in Awakari web UI
		subDescr = "id: <code>" + subId + "</code>"
	}
	err = tgCtx.Send(fmt.Sprintf(MsgFmtChatLinked, subDescr, interval), telebot.ModeHTML, telebot.NoPreview)
	return
}
