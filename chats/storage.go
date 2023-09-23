package chats

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	io.Closer
	Create(ctx context.Context, c Chat) (err error)
	Update(ctx context.Context, c Chat) (err error)
	Delete(ctx context.Context, k Key) (err error)
	ActivateNext(ctx context.Context, expiresNext time.Time) (c Chat, err error)
}
