package admin

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api/grpc/subject"
	"log/slog"
	"time"
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

func (sl serviceLogging) SetLimits(ctx context.Context, groupId, userId string, subj subject.Subject, count int64, expires time.Time) (err error) {
	err = sl.svc.SetLimits(ctx, groupId, userId, subj, count, expires)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("admin.SetLimits(groupId=%s, userId=%s, subj=%s, count=%d, expires=%s)", groupId, userId, subj, count, expires))
	default:
		sl.log.Error(fmt.Sprintf("admin.SetLimits(groupId=%s, userId=%s, subj=%s, count=%d, expires=%s): %s", groupId, userId, subj, count, expires, err))
	}
	return
}
