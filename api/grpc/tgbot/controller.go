package tgbot

import (
	"context"
	"encoding/json"
	"github.com/awakari/bot-telegram/service/messages"
	tgverifier "github.com/electrofocus/telegram-auth-verifier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Controller interface {
	ServiceServer
}

type controller struct {
	secretToken     []byte
	chanPostHandler messages.ChanPostHandler
}

func NewController(secretToken []byte, chanPostHandler messages.ChanPostHandler) Controller {
	return controller{
		secretToken:     secretToken,
		chanPostHandler: chanPostHandler,
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

	return
}
