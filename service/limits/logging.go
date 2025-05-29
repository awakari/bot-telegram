package limits

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/model/usage"
	"github.com/awakari/bot-telegram/util"
	"log/slog"
	"time"
)

type logging struct {
	svc Service
	log *slog.Logger
}

func NewLogging(svc Service, log *slog.Logger) Service {
	return logging{
		svc: svc,
		log: log,
	}
}

func (sl logging) Get(ctx context.Context, groupId, userId string, subj usage.Subject) (l usage.Limit, err error) {
	l, err = sl.svc.Get(ctx, groupId, userId, subj)
	sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("limits.Get(%s, %s, %s): %v, err=%s", groupId, userId, subj, l, err))
	return
}

func (sl logging) Set(ctx context.Context, groupId, userId string, subj usage.Subject, count int64, expires time.Time) (err error) {
	err = sl.svc.Set(ctx, groupId, userId, subj, count, expires)
	sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("limits.Set(%s, %s, %s, %d, %s): err=%s", groupId, userId, subj, count, expires, err))
	return
}

func (sl logging) Delete(ctx context.Context, groupId, userId string, subjs ...usage.Subject) (err error) {
	err = sl.svc.Delete(ctx, groupId, userId, subjs...)
	sl.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("limits.Delete(%s, %s, %s): err=%s", groupId, userId, subjs, err))
	return
}
