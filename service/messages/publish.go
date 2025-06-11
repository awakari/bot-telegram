package messages

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/pub"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/model"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/util"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/segmentio/ksuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/telebot.v3"
	"html"
	"strconv"
)

const ReqMsgPub = "msg_pub"
const PurposePublish = "msg_pub"
const ceKeyTgMessageId = "tgmessageid"
const attrValSpecVersion = "1.0"
const msgBusy = "Busy, please retry later"
const msgFmtPublished = "Message published, id: <pre>%s</pre>"
const msgLimitReached = "Message daily publishing limit reached. Payment is required to proceed. The message is being kept for 1 week"
const msgFmtPublishMissing = "message to publish is missing: %s"
const msgFmtRunOnceFailed = "failed to publish event: %s, cause: %s, retrying in: %s"

type FileType int32

const (
	FileTypeUndefined FileType = iota
	FileTypeAudio
	FileTypeDocument
	FileTypeImage
	FileTypeVideo
)

var publishBasicMarkup = &telebot.ReplyMarkup{
	ForceReply:  true,
	Placeholder: "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
}

func PublishBasicRequest(tgCtx telebot.Context) (err error) {
	_ = tgCtx.Send("Reply with your message to publish:")
	err = tgCtx.Send(ReqMsgPub, publishBasicMarkup)
	return
}

func PublishBasicReplyHandlerFunc(
	svcPub pub.Service,
	groupId string,
	cfg config.Config,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		userId := util.SenderToUserId(tgCtx)
		evt := pb.CloudEvent{
			Id:          ksuid.New().String(),
			Source:      "https://t.me/" + tgCtx.Chat().Username,
			SpecVersion: attrValSpecVersion,
			Type:        cfg.Api.Messages.Type,
		}
		if err == nil {
			err = toCloudEvent(tgCtx.Message(), args[1], &evt)
		}
		if err == nil {
			err = publish(tgCtx, svcPub, &evt, groupId, userId)
		}
		return
	}
}

func toCloudEvent(msg *telebot.Message, txt string, evt *pb.CloudEvent) (err error) {
	evt.Attributes = map[string]*pb.CloudEventAttributeValue{
		ceKeyTgMessageId: {
			Attr: &pb.CloudEventAttributeValue_CeString{
				CeString: strconv.Itoa(msg.ID),
			},
		},
		model.CeKeyTime: {
			Attr: &pb.CloudEventAttributeValue_CeTimestamp{
				CeTimestamp: timestamppb.New(msg.Time()),
			},
		},
	}
	switch {
	case txt != "":
		evt.Data = &pb.CloudEvent_TextData{
			TextData: txt,
		}
	case msg.Caption != "":
		evt.Data = &pb.CloudEvent_TextData{
			TextData: msg.Caption,
		}
	}
	if err == nil {
		var f telebot.File
		switch {
		case msg.Audio != nil:
			evt.Attributes[model.CeKeyTgFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeAudio),
				},
			}
			evt.Attributes[model.CeKeyTgFileMediaDuration] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Audio.Duration),
				},
			}
			f = msg.Audio.File
		case msg.Document != nil:
			evt.Attributes[model.CeKeyTgFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeDocument),
				},
			}
			f = msg.Document.File
		case msg.Location != nil:
			evt.Attributes[model.CeKeyLatitude] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: fmt.Sprintf("%f", msg.Location.Lat),
				},
			}
			evt.Attributes[model.CeKeyLongitude] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: fmt.Sprintf("%f", msg.Location.Lng),
				},
			}
		case msg.Photo != nil:
			evt.Attributes[model.CeKeyTgFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeImage),
				},
			}
			evt.Attributes[model.CeKeyTgFileImgHeight] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Photo.Height),
				},
			}
			evt.Attributes[model.CeKeyTgFileImgWidth] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Photo.Width),
				},
			}
			f = msg.Photo.File
		case msg.Video != nil:
			evt.Attributes[model.CeKeyTgFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeVideo),
				},
			}
			evt.Attributes[model.CeKeyTgFileMediaDuration] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Video.Duration),
				},
			}
			evt.Attributes[model.CeKeyTgFileImgHeight] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Video.Height),
				},
			}
			evt.Attributes[model.CeKeyTgFileImgWidth] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Video.Width),
				},
			}
			f = msg.Video.File
		}
		if f.FileID != "" {
			evt.Attributes[model.CeKeyTgFileId] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: f.FileID,
				},
			}
		}
		if f.UniqueID != "" {
			evt.Attributes[model.CeKeyTgFileUniqueId] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: f.UniqueID,
				},
			}
		}
	}
	return
}

func publish(
	tgCtx telebot.Context,
	svcPub pub.Service,
	evt *pb.CloudEvent,
	groupId, userId string,
) (err error) {
	err = svcPub.Publish(context.TODO(), evt, groupId, userId)
	switch {
	case errors.Is(err, pub.ErrLimitReached):
		err = errors.New(fmt.Sprintf("Message daily publishing limit reached. Consider to increase."))
	default:
		err = tgCtx.Send(fmt.Sprintf(msgFmtPublished, html.EscapeString(evt.Id)), telebot.ModeHTML)
	}
	return
}
