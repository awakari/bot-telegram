package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
)

const CmdStart = "sub_start"
const MsgFmtChatLinked = "Following the interest \"%s\" in this chat. " +
	"New results will appear here. " +
	"To manage own interests use the <a href=\"https://awakari.com/login.html\" target=\"blank\">app</a>."

func Start(
	clientAwk api.Client,
	svcReader reader.Service,
	urlCallbackBase string,
	groupId string,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		switch len(args) {
		case 1: // ask for min delivery interval
			subId := args[0]
			err = start(tgCtx, clientAwk, svcReader, urlCallbackBase, subId, groupId)
		default:
			err = errors.New(fmt.Sprintf("invalid response: expected 1 or 2 arguments, got %d", len(args)))
		}
		return
	}
}

func start(
	tgCtx telebot.Context,
	clientAwk api.Client,
	svcReader reader.Service,
	urlCallbackBase string,
	subId string,
	groupId string,
) (err error) {
	ctx := context.TODO()
	userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
	urlCallback := reader.MakeCallbackUrl(urlCallbackBase, tgCtx.Chat().ID)
	if err == nil {
		err = svcReader.CreateCallback(ctx, subId, urlCallback)
		switch {
		case errors.Is(err, chats.ErrAlreadyExists):
			// might be not an error, so try to re-link the subscription
			err = svcReader.DeleteCallback(ctx, subId, urlCallback)
			if err == nil {
				err = svcReader.CreateCallback(ctx, subId, urlCallback)
			}
		}
	}
	var subData subscription.Data
	if err == nil {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		subData, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
	}
	if err == nil {
		err = tgCtx.Send(fmt.Sprintf(MsgFmtChatLinked, subData.Description), telebot.ModeHTML, telebot.NoPreview)
	}
	return
}
