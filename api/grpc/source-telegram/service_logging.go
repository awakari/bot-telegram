package source_telegram

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

func (sl serviceLogging) Login(ctx context.Context, code int64, replicaIdx uint) (success bool, err error) {
	success, err = sl.svc.Login(ctx, code, replicaIdx)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("api.grpc.source-telegram.Login(%d. %d): %t", code, replicaIdx, success))
	default:
		sl.log.Error(fmt.Sprintf("api.grpc.source-telegram.Login(%d, %d): %s", code, replicaIdx, err))
	}
	return
}
