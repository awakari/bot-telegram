package subscriptions

import (
	"context"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service"
	"gopkg.in/telebot.v3"
)

const CmdStop = "sub_stop"

func Stop(svcReader reader.Service, urlCallbackBase string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		ctx := context.TODO()
		subId := args[0]
		urlCallback := reader.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID)
		_, err = svcReader.GetCallback(ctx, subId, urlCallback)
		if err == nil {
			err = svcReader.DeleteCallback(ctx, subId, urlCallback)
		}
		if err == nil {
			_ = tgCtx.Send("Stopped following the interest in this chat")
		}
		return
	}
}
