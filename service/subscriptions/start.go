package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

const msgFmtChatLinked = "Linked the subscription \"%s\" to this chat. New matching messages will appear here."
const msgFmtRenameFail = "Unable to rename the chat. Please rename it manually to: <pre>%s</pre>."

func Start(
	tgCtx telebot.Context,
	log *slog.Logger,
	clientAwk api.Client,
	chatStor chats.Storage,
	groupId string,
	msgFmt messages.Format,
) (err error) {
	subId := tgCtx.Data()
	if subId == "" {
		err = errors.New("subscription id argument is missing")
	}
	var userId string
	var chat chats.Chat
	if err == nil {
		userId = strconv.FormatInt(tgCtx.Sender().ID, 10)
		chat.Key = chats.Key{
			Id:    tgCtx.Chat().ID,
			SubId: subId,
		}
		chat.GroupId = groupId
		chat.UserId = userId
		chat.State = chats.StateActive
		chat.Expires = time.Now().UTC().Add(chats.ReaderTtl)
		err = chatStor.Create(context.TODO(), chat)
	}
	var subData subscription.Data
	if err == nil {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		subData, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
	}
	if err == nil {
		err = tgCtx.Send(fmt.Sprintf(msgFmtChatLinked, subData.Description))
	}
	if err == nil {
		_ = tgCtx.Bot().SetGroupDescription(tgCtx.Chat(), fmt.Sprintf("Subscription id: %s", subId))
		err = tgCtx.Bot().SetGroupTitle(tgCtx.Chat(), subData.Description)
	}
	if err != nil {
		err = tgCtx.Send(fmt.Sprintf(msgFmtRenameFail, subData.Description), telebot.ModeHTML)
	}
	if err == nil {
		r := chats.NewReader(tgCtx, clientAwk, chatStor, chat.Key, groupId, userId, msgFmt)
		go r.Run(context.Background(), log)
	}
	return
}
