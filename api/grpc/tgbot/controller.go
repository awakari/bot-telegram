package tgbot

import (
	"context"
	"encoding/json"
	tgverifier "github.com/electrofocus/telegram-auth-verifier"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Controller interface {
	ServiceServer
}

type controller struct {
	secretToken []byte
}

func NewController(secretToken []byte) Controller {
	return controller{
		secretToken: secretToken,
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
