package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Controller interface {
	ServiceServer
}

type controller struct {
	secretKey []byte
}

func NewController(secretToken string) Controller {
	s := sha256.Sum256([]byte(secretToken))
	var secretKey []byte
	for _, b := range s {
		secretKey = append(secretKey, b)
	}
	return controller{
		secretKey: secretKey,
	}
}

func (c controller) Validate(ctx context.Context, req *ValidateRequest) (resp *ValidateResponse, err error) {
	resp = &ValidateResponse{}
	h := hmac.New(sha256.New, c.secretKey)
	h.Write([]byte(req.DataCheckString))
	hh := hex.EncodeToString(h.Sum(nil))
	if req.Hash != hh {
		err = status.Error(codes.Unauthenticated, fmt.Sprintf("hash mismatch: expected %s, calculated %s", req.Hash, hh))
	}
	return
}
