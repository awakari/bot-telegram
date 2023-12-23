package auth

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

func (c controller) Validate(ctx context.Context, req *ValidateRequest) (resp *ValidateResponse, err error) {
	resp = &ValidateResponse{}
	var creds tgverifier.Credentials
	err = json.Unmarshal(req.AuthData, &creds)
	if err == nil {
		err = creds.Verify(c.secretToken)
	}
	if err != nil {
		err = status.Error(codes.Unauthenticated, err.Error())
	}
	return
}
