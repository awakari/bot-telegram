package chats

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	io.Closer
	LinkSubscription(ctx context.Context, c Chat) (err error)
	GetSubscriptionLink(ctx context.Context, subId string) (c Chat, err error)
	UpdateSubscriptionLink(ctx context.Context, c Chat) (err error)
	UnlinkSubscription(ctx context.Context, k Key) (err error)
	Delete(ctx context.Context, id int64) (count int64, err error)
	ActivateNext(ctx context.Context, expiresNext time.Time) (c Chat, err error)
}
