package telegram

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"strconv"
)

const attrKeyAuthor = "author"
const attrKeyMsgId = "telegrammessageid"
const attrKeyUpdId = "telegramupdateid"
const attrValType = "com.github.awakari.bot-telegram"
const attrValSpecVersion = "1.0"
const fmtLinkUser = "tg://user?id=%d"
const fmtUserName = "%s %s"

func SubmitCustomHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		data := args[0]
		var w model.Writer[*pb.CloudEvent]
		var evt pb.CloudEvent
		w, err = awakariClient.OpenMessagesWriter(groupIdCtx, userId)
		if err == nil {
			defer w.Close()
			err = protojson.Unmarshal([]byte(data), &evt)
		}
		var ackCount uint32
		if err == nil {
			evt.Source = fmt.Sprintf(fmtLinkUser, tgCtx.Sender().ID)
			evt.SpecVersion = attrValSpecVersion
			evt.Type = attrValType
			ackCount, err = w.WriteBatch([]*pb.CloudEvent{&evt})
		}
		if err == nil {
			switch ackCount {
			case 1:
				err = tgCtx.Send(fmt.Sprintf("Message published, id: <pre>%s</pre>", evt.Id), telebot.ModeHTML)
			default:
				err = tgCtx.Send("Busy, please retry later")
			}
		}
		return
	}
}

func SubmitText(tgCtx telebot.Context, awakariClient api.Client, groupId string) (err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
	sender := tgCtx.Sender()
	userId := strconv.FormatInt(sender.ID, 10)
	w, err := awakariClient.OpenMessagesWriter(groupIdCtx, userId)
	var ackCount uint32
	var evt *pb.CloudEvent
	if err == nil {
		defer w.Close()
		evt = convertMessage(sender, tgCtx.Update(), tgCtx.Message(), tgCtx.Text())
		ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
	}
	if err == nil {
		switch ackCount {
		case 1:
			err = tgCtx.Send(fmt.Sprintf("Message published, id: <pre>%s</pre>", evt.Id), telebot.ModeHTML)
		default:
			err = tgCtx.Send("Busy, please retry later")
		}
	}
	return
}

func convertMessage(sender *telebot.User, upd telebot.Update, msg *telebot.Message, txt string) (evt *pb.CloudEvent) {
	evt = &pb.CloudEvent{
		Id:          uuid.NewString(),
		Source:      fmt.Sprintf(fmtLinkUser, sender.ID),
		SpecVersion: attrValSpecVersion,
		Type:        attrValType,
		Attributes: map[string]*pb.CloudEventAttributeValue{
			attrKeyAuthor: {
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: fmt.Sprintf(fmtUserName, sender.FirstName, sender.LastName),
				},
			},
			attrKeyMsgId: {
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: strconv.Itoa(upd.ID),
				},
			},
			attrKeyUpdId: {
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: strconv.Itoa(msg.ID),
				},
			},
		},
		Data: &pb.CloudEvent_TextData{
			TextData: txt,
		},
	}
	return
}
