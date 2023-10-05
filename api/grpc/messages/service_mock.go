package messages

import (
	"context"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
)

type serviceMock struct{}

func NewServiceMock() Service {
	return &serviceMock{}
}

func (sm serviceMock) PutBatch(ctx context.Context, msgs []*pb.CloudEvent) (count uint32, err error) {
	for _, msg := range msgs {
		switch msg.Id {
		case "messages_fail":
			err = ErrInternal
		case "messages_conflict":
			err = errConflict
		}
		if err != nil {
			if err == errConflict {
				count++
				err = nil
			}
			break
		}
		count++
	}
	return
}

func (sm serviceMock) GetBatch(ctx context.Context, ids []string) (msgs []*pb.CloudEvent, err error) {
	for _, id := range ids {
		switch id {
		case "fail":
			err = ErrInternal
		case "missing":
		case "timeout":
			err = context.DeadlineExceeded
		default:
			msgs = append(msgs, &pb.CloudEvent{
				Id: id,
				Attributes: map[string]*pb.CloudEventAttributeValue{
					"awakariuserid": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "user0",
						},
					},
				},
			})
		}
		if err != nil {
			break
		}
	}
	return
}

func (sm serviceMock) DeleteBatch(ctx context.Context, ids []string) (count uint32, err error) {
	for _, id := range ids {
		switch id {
		case "fail":
			err = ErrInternal
		case "missing":
		case "timeout":
			err = context.DeadlineExceeded
		default:
			count++
		}
		if err != nil {
			break
		}
	}
	return
}
