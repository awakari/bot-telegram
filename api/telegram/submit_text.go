package telegram

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
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

func SubmitTextHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		sender := tgCtx.Sender()
		userId := strconv.FormatInt(sender.ID, 10)
		w, err := awakariClient.OpenMessagesWriter(groupIdCtx, userId)
		var ackCount uint32
		if err == nil {
			defer w.Close()
			evt := convertMessage(sender, tgCtx.Update(), tgCtx.Message(), tgCtx.Text())
			ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
		}
		if err == nil {
			switch ackCount {
			case 0:
				err = tgCtx.Send("Busy, please retry later")
			}
		}
		return
	}
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
