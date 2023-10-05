package messages

import (
	"context"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type clientMock struct {
}

func newClientMock() ServiceClient {
	return clientMock{}
}

func (cm clientMock) PutBatch(ctx context.Context, req *PutBatchRequest, opts ...grpc.CallOption) (resp *PutBatchResponse, err error) {
	resp = &PutBatchResponse{}
	for _, msg := range req.Msgs {
		switch msg.Id {
		case "messages_fail":
			err = status.Error(codes.Internal, "internal failure")
		case "messages_conflict":
			err = errConflict
		}
		if err != nil {
			if err == errConflict {
				err = nil
				resp.AckCount++
			}
			break
		}
		resp.AckCount++
	}
	if resp.AckCount > 2 {
		resp.AckCount = 2
	}
	return
}

func (cm clientMock) GetBatch(ctx context.Context, req *GetBatchRequest, opts ...grpc.CallOption) (resp *GetBatchResponse, err error) {
	resp = &GetBatchResponse{}
	for _, id := range req.Ids {
		switch id {
		case "fail":
			err = status.Error(codes.Internal, ErrInternal.Error())
		case "missing":
		case "timeout":
			err = status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
		default:
			resp.Msgs = append(resp.Msgs, &pb.CloudEvent{Id: id})
		}
		if err != nil {
			resp.Msgs = []*pb.CloudEvent{}
			break
		}
	}
	return
}

func (cm clientMock) DeleteBatch(ctx context.Context, req *DeleteBatchRequest, opts ...grpc.CallOption) (resp *DeleteBatchResponse, err error) {
	resp = &DeleteBatchResponse{}
	for _, id := range req.Ids {
		switch id {
		case "fail":
			err = status.Error(codes.Internal, ErrInternal.Error())
		case "missing":
		case "timeout":
			err = status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
		default:
			resp.AckCount++
		}
		if err != nil {
			break
		}
	}
	return
}
