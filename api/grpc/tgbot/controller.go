package tgbot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	tgverifier "github.com/electrofocus/telegram-auth-verifier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type Controller interface {
	ServiceServer
}

type controller struct {
	secretToken []byte
	cp          messages.ChanPostHandler
	chatStor    chats.Storage
	log         *slog.Logger
	clientAwk   api.Client
	tgBot       *telebot.Bot
	msgFmt      messages.Format
}

const intervalDefault = 1 * time.Minute

func NewController(
	secretToken []byte,
	cp messages.ChanPostHandler,
	chatStor chats.Storage,
	log *slog.Logger,
	clientAwk api.Client,
	tgBot *telebot.Bot,
	msgFmt messages.Format,
) Controller {
	return controller{
		secretToken: secretToken,
		cp:          cp,
		chatStor:    chatStor,
		log:         log,
		clientAwk:   clientAwk,
		tgBot:       tgBot,
		msgFmt:      msgFmt,
	}
}

func (c controller) Authenticate(ctx context.Context, req *AuthenticateRequest) (resp *AuthenticateResponse, err error) {
	resp = &AuthenticateResponse{}
	var creds tgverifier.Credentials
	err = json.Unmarshal(req.Data, &creds)
	if err == nil {
		err = creds.Verify(c.secretToken)
	}
	if err != nil {
		err = status.Error(codes.Unauthenticated, err.Error())
	}
	return
}

func (c controller) ListChannels(ctx context.Context, req *ListChannelsRequest) (resp *ListChannelsResponse, err error) {
	resp = &ListChannelsResponse{
		Page: []*Channel{},
	}
	var pattern string
	if req.Filter != nil {
		pattern = req.Filter.Pattern
	}
	filter := messages.ChanFilter{
		Pattern: pattern,
	}
	var order messages.Order
	switch req.Order {
	case Order_DESC:
		order = messages.OrderDesc
	default:
		order = messages.OrderAsc
	}
	var page []messages.Channel
	page, err = c.cp.List(ctx, filter, req.Limit, req.Cursor, order)
	if len(page) > 0 {
		for _, ch := range page {
			resp.Page = append(resp.Page, &Channel{
				LastUpdate: timestamppb.New(ch.LastUpdate),
				Link:       ch.Link,
			})
		}
	}
	err = encodeError(err)
	return
}

func (c controller) Subscribe(ctx context.Context, req *SubscribeRequest) (resp *SubscribeResponse, err error) {
	subId := req.SubId
	groupId := req.GroupId
	userId := req.UserId
	if !strings.HasPrefix(userId, service.PrefixUserId) {
		err = status.Error(codes.InvalidArgument, fmt.Sprintf("User id should have prefix: %s, got: %s", service.PrefixUserId, userId))
	}
	var chatId int64
	if err == nil {
		chatId, err = strconv.ParseInt(userId[len(service.PrefixUserId):], 10, 64)
		if err != nil {
			err = status.Error(codes.InvalidArgument, fmt.Sprintf("User id should end with numeric id: %s, %s", userId, err))
		}
	}
	if err == nil {
		chat := chats.Chat{
			Id:          chatId,
			SubId:       subId,
			GroupId:     groupId,
			UserId:      userId,
			MinInterval: intervalDefault,
		}
		err = c.chatStor.LinkSubscription(ctx, chat)
		err = encodeError(err)
	}
	if err == nil {
		u := telebot.Update{
			Message: &telebot.Message{
				Chat: &telebot.Chat{
					ID: chatId,
				},
			},
		}
		tgCtx := c.tgBot.NewContext(u)
		r := chats.NewReader(tgCtx, c.clientAwk, c.chatStor, chatId, subId, groupId, userId, c.msgFmt, intervalDefault)
		go r.Run(context.Background(), c.log)
		groupIdCtx := metadata.AppendToOutgoingContext(ctx, service.KeyGroupId, groupId)
		var d subscription.Data
		d, err = c.clientAwk.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			_ = tgCtx.Send(fmt.Sprintf("New query \"%s\" is linked to this chat", d.Description))
		}
	}
	return
}

func (c controller) Unsubscribe(ctx context.Context, req *UnsubscribeRequest) (resp *UnsubscribeResponse, err error) {
	chats.StopChatReader(req.SubId)
	err = c.chatStor.UnlinkSubscription(ctx, req.SubId)
	err = encodeError(err)
	return
}

func encodeError(src error) (dst error) {
	switch {
	case src == nil:
	default:
		dst = status.Error(codes.Unknown, src.Error())
	}
	return
}
