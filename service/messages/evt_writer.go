package messages

import (
	"errors"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
)

type evtWriter struct {
	e *pb.CloudEvent
	w model.Writer[*pb.CloudEvent]
}

var errBusy = errors.New("busy")

func (ew evtWriter) runOnce() (err error) {
	var ackCount uint32
	ackCount, err = ew.w.WriteBatch([]*pb.CloudEvent{ew.e})
	if err == nil {
		if ackCount < 1 {
			err = errBusy
		}
	}
	return
}
