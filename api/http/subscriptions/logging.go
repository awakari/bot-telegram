package subscriptions

import (
	"context"
	"fmt"
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

func (sl serviceLogging) Subscribe(ctx context.Context, interestId, groupId, userId, url string, interval time.Duration) (err error) {
	err = sl.svc.Subscribe(ctx, interestId, groupId, userId, url, interval)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("subscriptions.Subscribe(%s, %s, %s): err=%s", interestId, url, interval, err))
	return
}

func (sl serviceLogging) Subscription(ctx context.Context, interestId, groupId, userId, url string) (cb Subscription, err error) {
	cb, err = sl.svc.Subscription(ctx, interestId, groupId, userId, url)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("subscriptions.Subscription(%s, %s): %+v, err=%s", interestId, url, cb, err))
	return
}

func (sl serviceLogging) Unsubscribe(ctx context.Context, interestId, groupId, userId, url string) (err error) {
	err = sl.svc.Unsubscribe(ctx, interestId, groupId, userId, url)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("subscriptions.Unsubscribe(%s, %s): err=%s", interestId, url, err))
	return
}

func (sl serviceLogging) InterestsByUrl(ctx context.Context, groupId, userId string, limit uint32, url, cursor string) (page []string, err error) {
	page, err = sl.svc.InterestsByUrl(ctx, groupId, userId, limit, url, cursor)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("subscriptions.InterestsByUrl(%s, %s, %d, %s, %s): %d, err=%s", groupId, userId, limit, url, cursor, len(page), err))
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
