package messages

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"strconv"
)

const ReqMsgPubBasic = "msg_pub_basic"

const attrKeyAuthor = "author"
const attrKeyMsgId = "telegrammessageid"
const attrKeyUpdId = "telegramupdateid"
const attrValType = "com.github.awakari.bot-telegram"
const attrValSpecVersion = "1.0"
const fmtLinkUser = "tg://user?id=%d"
const fmtUserName = "%s %s"

var publishBasicMarkup = &telebot.ReplyMarkup{
	ForceReply:  true,
	Placeholder: "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
}

func PublishBasicRequest(tgCtx telebot.Context) (err error) {
	_ = tgCtx.Send("Reply with a text")
	err = tgCtx.Send(ReqMsgPubBasic, publishBasicMarkup)
	return
}

func PublishBasicReplyHandlerFunc(clientAwk api.Client, groupId string, kbd *telebot.ReplyMarkup) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		sender := tgCtx.Sender()
		userId := strconv.FormatInt(sender.ID, 10)
		w, err := clientAwk.OpenMessagesWriter(groupIdCtx, userId)
		var ackCount uint32
		var evt *pb.CloudEvent
		if err == nil {
			defer w.Close()
			evt = toCloudEvent(sender, tgCtx.Message(), args[1])
			ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
		}
		if err == nil {
			switch ackCount {
			case 1:
				err = tgCtx.Send(fmt.Sprintf("Message published, id: <pre>%s</pre>", evt.Id), kbd, telebot.ModeHTML)
			default:
				err = tgCtx.Send("Busy, please retry later", kbd)
			}
		}
		return
	}
}

func toCloudEvent(sender *telebot.User, msg *telebot.Message, txt string) (evt *pb.CloudEvent) {
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

func PublishCustomHandlerFunc(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		data := args[0]
		var w model.Writer[*pb.CloudEvent]
		var evt pb.CloudEvent
		w, err = clientAwk.OpenMessagesWriter(groupIdCtx, userId)
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
