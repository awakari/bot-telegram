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
	"time"
)

const CmdStart = "sub_start"
const MsgFmtChatLinked = "Linked the subscription \"%s\" to this chat. " +
	"New results will appear here. Min interval: %s. " +
	"To manage own subscriptions use the <a href=\"https://awakari.com/login.html\" target=\"blank\">app</a>."

var deliveryIntervalRows = [][]string{
	{
		"1s", "1m", "5m", "15m",
	},
	{
		"1h", "6h", "12h", "1d",
	},
}

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
			err = requestDeliveryInterval(tgCtx, subId)
		case 2:
			subId := args[0]
			minIntervalStr := args[1]
			var minInterval time.Duration
			minInterval, err = time.ParseDuration(minIntervalStr)
			switch err {
			case nil:
				err = start(tgCtx, clientAwk, svcReader, urlCallbackBase, subId, groupId, minInterval)
			default:
				err = errors.New(fmt.Sprintf("failed to parse min delivery interval: %s", err))
			}
		default:
			err = errors.New(fmt.Sprintf("invalid response: expected 1 or 2 arguments, got %d", len(args)))
		}
		return
	}
}

func requestDeliveryInterval(tgCtx telebot.Context, subId string) (err error) {
	m := &telebot.ReplyMarkup{}
	var rows []telebot.Row
	for _, diRow := range deliveryIntervalRows {
		var rowBtns []telebot.Btn
		for _, di := range diRow {
			btn := telebot.Btn{
				Text: di,
				Data: fmt.Sprintf("%s %s %s", CmdStart, subId, di),
			}
			rowBtns = append(rowBtns, btn)
		}
		row := m.Row(rowBtns...)
		rows = append(rows, row)
	}
	m.Inline(rows...)
	err = tgCtx.Send("Choose the minimum interval for the message delivery for this subscription:", m)
	return
}

func start(
	tgCtx telebot.Context,
	clientAwk api.Client,
	svcReader reader.Service,
	urlCallbackBase string,
	subId string,
	groupId string,
	minInterval time.Duration,
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
		err = tgCtx.Send(fmt.Sprintf(MsgFmtChatLinked, subData.Description, minInterval), telebot.ModeHTML, telebot.NoPreview)
	}
	return
}
