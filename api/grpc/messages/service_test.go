package messages

import (
	"context"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
)

func TestService_PutBatch(t *testing.T) {
	svc := NewService(newClientMock())
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc = NewServiceLogging(svc, log)
	cases := map[string]struct {
		msgs     []*pb.CloudEvent
		ackCount uint32
		err      error
	}{
		"1 => ack 1": {
			msgs: []*pb.CloudEvent{
				{
					Id: "msg0",
				},
			},
			ackCount: 1,
		},
		"3 => ack 2": {
			msgs: []*pb.CloudEvent{
				{
					Id: "msg0",
				},
				{
					Id: "msg1",
				},
				{
					Id: "msg2",
				},
			},
			ackCount: 2,
		},
		"messages_fail": {
			msgs: []*pb.CloudEvent{
				{
					Id: "messages_fail",
				},
				{
					Id: "msg1",
				},
			},
			err: ErrInternal,
		},
		"messages_conflict": {
			msgs: []*pb.CloudEvent{
				{
					Id: "msg0",
				},
				{
					Id: "messages_conflict",
				},
			},
			ackCount: 2,
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			ackCount, err := svc.PutBatch(context.TODO(), c.msgs)
			assert.Equal(t, c.ackCount, ackCount)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestService_GetBatch(t *testing.T) {
	svc := NewService(newClientMock())
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc = NewServiceLogging(svc, log)
	cases := map[string]struct {
		ids  []string
		msgs []*pb.CloudEvent
		err  error
		err1 error
	}{
		"ok": {
			ids: []string{
				"msg0",
				"missing",
				"msg2",
			},
			msgs: []*pb.CloudEvent{
				{
					Id: "msg0",
				},
				{
					Id: "msg2",
				},
			},
		},
		"fail": {
			ids: []string{
				"msg0",
				"fail",
				"msg2",
			},
			err: ErrInternal,
		},
		"timeout": {
			ids: []string{
				"msg0",
				"msg1",
				"timeout",
			},
			err: context.DeadlineExceeded,
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			var msgs []*pb.CloudEvent
			msgs, err := svc.GetBatch(context.TODO(), c.ids)
			assert.Equal(t, len(c.msgs), len(msgs))
			for i, expectedMsg := range c.msgs {
				assert.Equal(t, expectedMsg.Id, msgs[i].Id)
			}
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestService_DeleteBatch(t *testing.T) {
	svc := NewService(newClientMock())
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc = NewServiceLogging(svc, log)
	cases := map[string]struct {
		ids      []string
		ackCount uint32
		err      error
		err1     error
	}{
		"ok": {
			ids: []string{
				"msg0",
				"missing",
				"msg2",
			},
			ackCount: 2,
		},
		"fail": {
			ids: []string{
				"msg0",
				"fail",
				"msg2",
			},
			ackCount: 1,
			err:      ErrInternal,
		},
		"timeout": {
			ids: []string{
				"msg0",
				"msg1",
				"timeout",
			},
			ackCount: 2,
			err:      context.DeadlineExceeded,
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			ackCount, err := svc.DeleteBatch(context.TODO(), c.ids)
			assert.Equal(t, c.ackCount, ackCount)
			assert.ErrorIs(t, err, c.err)
		})
	}
}
