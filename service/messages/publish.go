package messages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/messages"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/limits"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

const ReqMsgPub = "msg_pub"
const PurposePublish = "msg_pub"
const attrKeyMsgId = "tgmessageid"
const attrValSpecVersion = "1.0"
const msgBusy = "Busy, please retry later"
const msgFmtPublished = "Message published, id: <pre>%s</pre>"
const msgLimitReached = "Message daily publishing limit reached. Payment is required to proceed. The message is being kept for 1 week"
const msgFmtPublishMissing = "message to publish is missing: %s"
const msgFmtRunOnceFailed = "failed to publish event: %s, cause: %s, retrying in: %s"

// file attrs
const attrKeyFileId = "tgfileid"
const attrKeyFileUniqueId = "tgfileuniqueid"
const attrKeyFileMediaDuration = "tgfilemediaduration"
const attrKeyFileImgHeight = "tgfileimgheight"
const attrKeyFileImgWidth = "tgfileimgwidth"
const attrKeyFileType = "tgfiletype"
const attrKeyLatitude = "latitude"
const attrKeyLongitude = "longitude"

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
	clientAwk api.Client,
	groupId string,
	svcMsgs messages.Service,
	cfgPayment config.PaymentConfig,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
		sender := tgCtx.Sender()
		userId := fmt.Sprintf(service.FmtUserId, sender.ID)
		w, err := clientAwk.OpenMessagesWriter(groupIdCtx, userId)
		evt := pb.CloudEvent{
			Id:          uuid.NewString(),
			Source:      "@AwakariBot",
			SpecVersion: attrValSpecVersion,
			Type:        "com.github.awakari.bot-telegram.v1",
		}
		if err == nil {
			defer w.Close()
			err = toCloudEvent(tgCtx.Message(), args[1], &evt)
		}
		if err == nil {
			err = publish(tgCtx, w, &evt, svcMsgs, cfgPayment)
		}
		return
	}
}

func toCloudEvent(msg *telebot.Message, txt string, evt *pb.CloudEvent) (err error) {
	evt.Attributes = map[string]*pb.CloudEventAttributeValue{
		attrKeyMsgId: {
			Attr: &pb.CloudEventAttributeValue_CeString{
				CeString: strconv.Itoa(msg.ID),
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
			evt.Attributes[attrKeyFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeAudio),
				},
			}
			evt.Attributes[attrKeyFileMediaDuration] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Audio.Duration),
				},
			}
			f = msg.Audio.File
		case msg.Document != nil:
			evt.Attributes[attrKeyFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeDocument),
				},
			}
			f = msg.Document.File
		case msg.Location != nil:
			evt.Attributes[attrKeyLatitude] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: fmt.Sprintf("%f", msg.Location.Lat),
				},
			}
			evt.Attributes[attrKeyLongitude] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: fmt.Sprintf("%f", msg.Location.Lng),
				},
			}
		case msg.Photo != nil:
			evt.Attributes[attrKeyFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeImage),
				},
			}
			evt.Attributes[attrKeyFileImgHeight] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Photo.Height),
				},
			}
			evt.Attributes[attrKeyFileImgWidth] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Photo.Width),
				},
			}
			f = msg.Photo.File
		case msg.Video != nil:
			evt.Attributes[attrKeyFileType] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(FileTypeVideo),
				},
			}
			evt.Attributes[attrKeyFileMediaDuration] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Video.Duration),
				},
			}
			evt.Attributes[attrKeyFileImgHeight] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Video.Height),
				},
			}
			evt.Attributes[attrKeyFileImgWidth] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeInteger{
					CeInteger: int32(msg.Video.Width),
				},
			}
			f = msg.Video.File
		}
		if f.FileID != "" {
			evt.Attributes[attrKeyFileId] = &pb.CloudEventAttributeValue{
				Attr: &pb.CloudEventAttributeValue_CeString{
					CeString: f.FileID,
				},
			}
		}
		if f.UniqueID != "" {
			evt.Attributes[attrKeyFileUniqueId] = &pb.CloudEventAttributeValue{
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
	w model.Writer[*pb.CloudEvent],
	evt *pb.CloudEvent,
	svcMsgs messages.Service,
	cfgPayment config.PaymentConfig,
) (err error) {
	var ackCount uint32
	ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
	switch {
	case ackCount == 0 && errors.Is(err, limits.ErrReached):
		// ackCount, err = publishInvoice(tgCtx, evt, svcMsgs, cfgPayment, kbd)
		err = errors.New(fmt.Sprintf("Message daily publishing limit reached. Consider to donate and increase your limit."))
	case ackCount == 1:
		err = tgCtx.Send(fmt.Sprintf(msgFmtPublished, evt.Id), telebot.ModeHTML)
	}
	if err == nil {
		switch ackCount {
		case 0:
			err = tgCtx.Send(msgBusy)
		}
	}
	return
}

func publishInvoice(
	tgCtx telebot.Context,
	evt *pb.CloudEvent,
	svcMsgs messages.Service,
	cfgPayment config.PaymentConfig,
	kbd *telebot.ReplyMarkup,
) (ackCount uint32, err error) {
	ackCount, err = svcMsgs.PutBatch(context.TODO(), []*pb.CloudEvent{evt})
	if ackCount == 1 {
		if kbd == nil {
			_ = tgCtx.Send(msgLimitReached)
		} else {
			_ = tgCtx.Send(msgLimitReached, kbd)
		}
		var orderData []byte
		orderData, err = json.Marshal(service.Order{
			Purpose: PurposePublish,
			Payload: evt.Id,
		})
		if err == nil {
			label := fmt.Sprintf("Publish Message %s", evt.Id)
			invoice := telebot.Invoice{
				Start:       evt.Id,
				Title:       "Publish Message",
				Description: label,
				Payload:     string(orderData),
				Currency:    cfgPayment.Currency.Code,
				Prices: []telebot.Price{
					{
						Label:  label,
						Amount: int(cfgPayment.Price.MessagePublishing.Extra * cfgPayment.Currency.SubFactor),
					},
				},
				Token: cfgPayment.Provider.Token,
				Total: int(cfgPayment.Price.MessagePublishing.Extra * cfgPayment.Currency.SubFactor),
			}
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
	}
	return
}

func PublishPreCheckout(svcMsgs messages.Service, cfgPayment config.PaymentConfig) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		ctx, cancel := context.WithTimeout(context.TODO(), cfgPayment.PreCheckout.Timeout)
		defer cancel()
		evtId := args[0]
		var evts []*pb.CloudEvent
		evts, err = svcMsgs.GetBatch(ctx, []string{evtId})
		switch {
		case err != nil:
			err = tgCtx.Accept(err.Error())
		case len(evts) == 0:
			err = tgCtx.Accept(fmt.Sprintf(msgFmtPublishMissing, evtId))
		default:
			err = tgCtx.Accept()
		}
		return
	}
}

func PublishPaid(
	svcMsgs messages.Service,
	clientAwk api.Client,
	groupId string,
	log *slog.Logger,
	cfgBackoff config.BackoffConfig,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		evtId := args[0]
		var evts []*pb.CloudEvent
		evts, err = svcMsgs.GetBatch(context.TODO(), []string{evtId})
		if err == nil {
			if len(evts) == 0 {
				err = fmt.Errorf(msgFmtPublishMissing, evtId)
			}
		}
		var w model.Writer[*pb.CloudEvent]
		if err == nil {
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
			userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
			w, err = clientAwk.OpenMessagesWriter(groupIdCtx, userId)
		}
		if err == nil {
			defer w.Close()
			b := backoff.NewExponentialBackOff()
			b.InitialInterval = cfgBackoff.Init
			b.Multiplier = cfgBackoff.Factor
			b.MaxElapsedTime = cfgBackoff.LimitTotal
			ew := writer{
				e: evts[0],
				w: w,
			}
			err = backoff.RetryNotify(ew.runOnce, b, func(err error, d time.Duration) {
				log.Warn(fmt.Sprintf(msgFmtRunOnceFailed, evtId, err, d))
				if d > 1*time.Second {
					_ = tgCtx.Send("Publishing the message, please wait...")
				}
			})
		}
		if err == nil {
			_ = tgCtx.Send(fmt.Sprintf(msgFmtPublished, evtId), telebot.ModeHTML)
			_, err = svcMsgs.DeleteBatch(context.TODO(), []string{evtId})
		}
		return
	}
}
