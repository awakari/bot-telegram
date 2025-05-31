package subscriptions

import (
	"context"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/util"
	"gopkg.in/telebot.v3"
)

const CmdStop = "sub_stop"

func Stop(svcReader reader.Service, urlCallbackBase, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		ctx := context.TODO()
		interestId := args[0]
		userId := util.SenderToUserId(tgCtx)
		urlCallback := reader.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID, userId)
		_, err = svcReader.Subscription(ctx, interestId, groupId, userId, urlCallback)
		switch err {
		case nil:
			if err == nil {
				err = svcReader.Unsubscribe(ctx, interestId, groupId, userId, urlCallback)
			}
		default:
			urlCallbackOld := reader.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID, "")
			_, err = svcReader.Subscription(ctx, interestId, groupId, userId, urlCallbackOld)
			if err == nil {
				err = svcReader.Unsubscribe(ctx, interestId, groupId, userId, urlCallbackOld)
			}
		}
		if err == nil {
			_ = tgCtx.Send("Unsubscribed from the interest in this chat")
		}
		return
	}
}
