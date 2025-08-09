package tgbot

import (
	"context"
	"github.com/awakari/bot-telegram/api/http/subscriptions"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/bytedance/sonic"
	tgverifier "github.com/electrofocus/telegram-auth-verifier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/telebot.v3"
	"log/slog"
)

type Controller interface {
	ServiceServer
}

type controller struct {
	secretToken     []byte
	cp              messages.ChanPostHandler
	svcSubs         subscriptions.Service
	urlCallbackBase string
	log             *slog.Logger
	tgBot           *telebot.Bot
	msgFmt          messages.Format
}

func NewController(
	secretToken []byte,
	cp messages.ChanPostHandler,
	svcSubs subscriptions.Service,
	urlCallbackBase string,
	log *slog.Logger,
	tgBot *telebot.Bot,
	msgFmt messages.Format,
) Controller {
	return controller{
		secretToken:     secretToken,
		cp:              cp,
		svcSubs:         svcSubs,
		urlCallbackBase: urlCallbackBase,
		log:             log,
		tgBot:           tgBot,
		msgFmt:          msgFmt,
	}
}

func (c controller) Authenticate(ctx context.Context, req *AuthenticateRequest) (resp *AuthenticateResponse, err error) {
	resp = &AuthenticateResponse{}
	var creds tgverifier.Credentials
	err = sonic.Unmarshal(req.Data, &creds)
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

func encodeError(src error) (dst error) {
	switch {
	case src == nil:
	default:
		dst = status.Error(codes.Unknown, src.Error())
	}
	return
}
