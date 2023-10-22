package telegram

import (
	"context"
	"fmt"
	"log/slog"
)

type serviceLogging struct {
	svc Service
	log *slog.Logger
}

func NewServiceLogging(svc Service, log *slog.Logger) Service {
	return serviceLogging{
		svc: svc,
		log: log,
	}
}

func (sl serviceLogging) List(ctx context.Context, limit uint32, cursor int64) (page []int64, err error) {
	page, err = sl.svc.List(ctx, limit, cursor)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("api.grpc.source.telegram.List(%d, %d): %d", limit, cursor, len(page)))
	default:
		sl.log.Error(fmt.Sprintf("api.grpc.source.telegram.List(%d, %d): %s", limit, cursor, err))
	}
	return
}
