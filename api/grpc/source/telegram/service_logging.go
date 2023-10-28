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

func (sl serviceLogging) Create(ctx context.Context, ch *Channel) (err error) {
	err = sl.svc.Create(ctx, ch)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("api.grpc.source.telegram.Create(%+v): ok", ch))
	default:
		sl.log.Error(fmt.Sprintf("api.grpc.source.telegram.Create(%+v): %s", ch, err))
	}
	return
}

func (sl serviceLogging) Read(ctx context.Context, link string) (ch *Channel, err error) {
	ch, err = sl.svc.Read(ctx, link)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("api.grpc.source.telegram.Read(%s): %+v", link, ch))
	default:
		sl.log.Error(fmt.Sprintf("api.grpc.source.telegram.Create(%s): %s", link, err))
	}
	return
}

func (sl serviceLogging) Delete(ctx context.Context, link string) (err error) {
	err = sl.svc.Delete(ctx, link)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("api.grpc.source.telegram.Delete(%s): ok", link))
	default:
		sl.log.Error(fmt.Sprintf("api.grpc.source.telegram.Delete(%s): %s", link, err))
	}
	return
}

func (sl serviceLogging) List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []*Channel, err error) {
	page, err = sl.svc.List(ctx, filter, limit, cursor)
	switch err {
	case nil:
		sl.log.Debug(fmt.Sprintf("api.grpc.source.telegram.List(%+v, %d, %s): %d", filter, limit, cursor, len(page)))
	default:
		sl.log.Error(fmt.Sprintf("api.grpc.source.telegram.List(%+v, %d, %s): %s", filter, limit, cursor, err))
	}
	return
}
