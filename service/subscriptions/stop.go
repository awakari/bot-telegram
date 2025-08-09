package subscriptions

import (
	"context"
	"github.com/awakari/bot-telegram/api/http/subscriptions"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/util"
	"gopkg.in/telebot.v3"
)

const CmdStop = "sub_stop"

func Stop(svcSubs subscriptions.Service, urlCallbackBase, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		ctx := context.TODO()
		interestId := args[0]
		userId := util.SenderToUserId(tgCtx)
		urlCallback := subscriptions.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID, userId)
		_, err = svcSubs.Subscription(ctx, interestId, groupId, userId, urlCallback)
		switch err {
		case nil:
			if err == nil {
				err = svcSubs.Unsubscribe(ctx, interestId, groupId, userId, urlCallback)
			}
		default:
			urlCallbackOld := subscriptions.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID, "")
			_, err = svcSubs.Subscription(ctx, interestId, groupId, userId, urlCallbackOld)
			if err == nil {
				err = svcSubs.Unsubscribe(ctx, interestId, groupId, userId, urlCallbackOld)
			}
		}
		if err == nil {
			_ = tgCtx.Send("Unsubscribed from the interest in this chat")
		}
		return
	}
}
