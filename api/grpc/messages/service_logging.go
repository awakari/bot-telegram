package messages

import (
	"context"
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"log/slog"
)

type svcLogging struct {
	svc Service
	log *slog.Logger
}

func NewServiceLogging(svc Service, log *slog.Logger) Service {
	return svcLogging{
		svc: svc,
		log: log,
	}
}

func (sl svcLogging) PutBatch(ctx context.Context, msgs []*pb.CloudEvent) (count uint32, err error) {
	count, err = sl.svc.PutBatch(ctx, msgs)
	ll := sl.logLevel(err)
	var msgIds []string
	for _, msg := range msgs {
		msgIds = append(msgIds, msg.Id)
	}
	sl.log.Log(ctx, ll, fmt.Sprintf("messages.PutBatch(ids=%+v): ack=%d, err=%s", msgIds, count, err))
	return
}

func (sl svcLogging) GetBatch(ctx context.Context, ids []string) (msgs []*pb.CloudEvent, err error) {
	msgs, err = sl.svc.GetBatch(ctx, ids)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("messages.GetBatch(%d): %d, err=%s", len(ids), len(msgs), err))
	return
}

func (sl svcLogging) DeleteBatch(ctx context.Context, ids []string) (count uint32, err error) {
	count, err = sl.svc.DeleteBatch(ctx, ids)
	ll := sl.logLevel(err)
	sl.log.Log(ctx, ll, fmt.Sprintf("messages.DeleteBatch(%d): ack=%d, err=%s", len(ids), count, err))
	return
}

func (sl svcLogging) logLevel(err error) (lvl slog.Level) {
	switch err {
	case nil:
		lvl = slog.LevelInfo
	default:
		lvl = slog.LevelError
	}
	return
}
