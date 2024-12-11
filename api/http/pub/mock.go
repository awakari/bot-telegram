package pub

import (
	"context"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
)

type mock struct {
	chEvt chan<- *pb.CloudEvent
}

func NewMock(chEvt chan<- *pb.CloudEvent) Service {
	return mock{
		chEvt: chEvt,
	}
}

func (m mock) Publish(ctx context.Context, evt *pb.CloudEvent, groupId, userId string) (err error) {
	m.chEvt <- evt
	return
}
