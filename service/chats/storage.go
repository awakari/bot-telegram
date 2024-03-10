package chats

import (
	"context"
	"io"
)

type Storage interface {
	io.Closer
	LinkSubscription(ctx context.Context, c Chat) (err error)
	GetSubscriptionLink(ctx context.Context, subId string) (c Chat, err error)
	UnlinkSubscription(ctx context.Context, subId string) (err error)
	Delete(ctx context.Context, id int64) (count int64, err error)
	GetBatch(ctx context.Context, idRem, idDiv uint32, limit uint32, cursor int64) (page []Chat, err error)
	Count(ctx context.Context) (count int64, err error)
}
