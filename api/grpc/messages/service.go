package messages

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	PutBatch(ctx context.Context, msgs []*pb.CloudEvent) (count uint32, err error)
	GetBatch(ctx context.Context, ids []string) (msgs []*pb.CloudEvent, err error)
	DeleteBatch(ctx context.Context, ids []string) (count uint32, err error)
}

type service struct {
	client ServiceClient
}

// ErrInternal indicates some unexpected internal failure.
var ErrInternal = errors.New("messages: internal failure")

// errConflict indicates the message already exists and can not be written
var errConflict = errors.New("messages: write conflict")

func NewService(client ServiceClient) Service {
	return service{
		client: client,
	}
}

func (svc service) PutBatch(ctx context.Context, msgs []*pb.CloudEvent) (count uint32, err error) {
	req := PutBatchRequest{
		Msgs: msgs,
	}
	var resp *PutBatchResponse
	resp, err = svc.client.PutBatch(ctx, &req)
	if resp != nil {
		count = resp.AckCount
	}
	err = decodeError(err)
	return
}

func (svc service) GetBatch(ctx context.Context, ids []string) (msgs []*pb.CloudEvent, err error) {
	req := GetBatchRequest{
		Ids: ids,
	}
	var resp *GetBatchResponse
	resp, err = svc.client.GetBatch(ctx, &req)
	if resp != nil {
		msgs = resp.Msgs
	}
	err = decodeError(err)
	return
}

func (svc service) DeleteBatch(ctx context.Context, ids []string) (count uint32, err error) {
	req := DeleteBatchRequest{
		Ids: ids,
	}
	var resp *DeleteBatchResponse
	resp, err = svc.client.DeleteBatch(ctx, &req)
	if resp != nil {
		count = resp.AckCount
	}
	err = decodeError(err)
	return
}

func decodeError(src error) (dst error) {
	switch {
	case status.Code(src) == codes.OK:
		dst = nil
	case status.Code(src) == codes.DeadlineExceeded:
		dst = context.DeadlineExceeded
	default:
		dst = fmt.Errorf("%w: %s", ErrInternal, src)
	}
	return
}
