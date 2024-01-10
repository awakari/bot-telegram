package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"log/slog"
)

const CmdStart = "sub_start"
const msgFmtChatLinked = "Linked the subscription \"%s\" to this chat. New matching messages will appear here."

func Start(
	log *slog.Logger,
	clientAwk api.Client,
	chatStor chats.Storage,
	groupId string,
	msgFmt messages.Format,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		if subId == "" {
			err = errors.New("subscription id argument is missing")
		}
		var userId string
		var chat chats.Chat
		if err == nil {
			userId = fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
			chat.Id = tgCtx.Chat().ID
		}
		if err == nil {
			chat.SubId = subId
			chat.GroupId = groupId
			chat.UserId = userId
			err = chatStor.LinkSubscription(context.TODO(), chat)
			switch {
			case errors.Is(err, chats.ErrAlreadyExists):
				err = errors.New("the chat is already linked to a subscription, try to use another group chat")
			}
		}
		var subData subscription.Data
		if err == nil {
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
			subData, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
		}
		if err == nil {
			err = tgCtx.Send(fmt.Sprintf(msgFmtChatLinked, subData.Description), telebot.ModeHTML)
		}
		if err == nil {
			r := chats.NewReader(tgCtx, clientAwk, chatStor, chat.Id, chat.SubId, groupId, userId, msgFmt)
			go r.Run(context.Background(), log)
		}
		return
	}
}
