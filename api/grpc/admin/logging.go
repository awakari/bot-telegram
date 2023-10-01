package admin

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api/grpc/subject"
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

func (sl serviceLogging) SetLimits(ctx context.Context, groupId, userId string, subj subject.Subject, count int64) (err error) {
	err = sl.svc.SetLimits(ctx, groupId, userId, subj, count)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("admin.SetLimits(groupId=%s, userId=%s, subj=%s, count=%d)", groupId, userId, subj, count))
	default:
		sl.log.Error(fmt.Sprintf("admin.SetLimits(groupId=%s, userId=%s, subj=%s, count=%d): %s", groupId, userId, subj, count, err))
	}
	return
}
