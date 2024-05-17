package reader

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

func (sl serviceLogging) CreateCallback(ctx context.Context, subId, url string) (err error) {
	err = sl.svc.CreateCallback(ctx, subId, url)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("reader.CreateCallback(%s, %s): err=%s", subId, url, err))
	return
}

func (sl serviceLogging) GetCallback(ctx context.Context, subId string) (cb Callback, err error) {
	cb, err = sl.svc.GetCallback(ctx, subId)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("reader.GetCallback(%s): %+v, err=%s", subId, cb, err))
	return
}

func (sl serviceLogging) DeleteCallback(ctx context.Context, subId, url string) (err error) {
	err = sl.svc.DeleteCallback(ctx, subId, url)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("reader.DeleteCallback(%s, %s): err=%s", subId, url, err))
	return
}

func (sl serviceLogging) logLevel(err error) (lvl slog.Level) {
	switch err {
	case nil:
		lvl = slog.LevelInfo
	default:
		lvl = slog.LevelError
	}
	return
}
