package messages

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/limits"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"log/slog"
)

type ChanPostHandler struct {
	ClientAwk api.Client
	GroupId   string
	Log       *slog.Logger
}

func (cp ChanPostHandler) Publish(tgCtx telebot.Context) (err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, cp.GroupId)
	chanUserName := fmt.Sprintf("@%s", tgCtx.Chat().Username)
	w, err := cp.ClientAwk.OpenMessagesWriter(groupIdCtx, chanUserName)
	evt := pb.CloudEvent{
		Id:          uuid.NewString(),
		Source:      chanUserName,
		SpecVersion: attrValSpecVersion,
		Type:        "com.github.awakari.bot-telegram.v1",
	}
	if err == nil {
		defer w.Close()
		err = toCloudEvent(tgCtx.Message(), tgCtx.Text(), &evt)
	}
	if err == nil {
		err = cp.publish(tgCtx, w, &evt)
	}
	return
}

func (cp ChanPostHandler) publish(tgCtx telebot.Context, w model.Writer[*pb.CloudEvent], evt *pb.CloudEvent) (err error) {
	var ackCount uint32
	ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
	switch {
	case ackCount == 0 && errors.Is(err, limits.ErrReached):
		cp.Log.Warn(fmt.Sprintf("Message daily publishing limit reached for channel: @%s", tgCtx.Chat().Username))
	case ackCount == 1:
		cp.Log.Debug(fmt.Sprintf("Message from channel @%s published, event id: %s", tgCtx.Chat().Username, evt.Id))
	}
	if err == nil {
		switch ackCount {
		case 0:
			err = tgCtx.Send(msgBusy)
		}
	}
	return
}
