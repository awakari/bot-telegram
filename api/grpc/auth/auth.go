package auth

import (
	"context"
	"github.com/awakari/bot-telegram/model"
	"google.golang.org/grpc/metadata"
)

func SetOutgoingAuthInfo(src context.Context, groupId, userId string) (dst context.Context) {
	dst = metadata.AppendToOutgoingContext(src, model.KeyGroupId, groupId, model.KeyUserId, userId)
	return
}
