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
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"log/slog"
	"strconv"
	"time"
)

const ReqMsgPubBasic = "msg_pub_basic"
const PurposePublish = "msg_pub"
const attrKeyAuthor = "author"
const attrKeyMsgId = "telegrammessageid"
const attrValType = "com.github.awakari.bot-telegram"
const attrValSpecVersion = "1.0"
const fmtLinkUser = "tg://user?id=%d"
const fmtUserName = "%s %s"
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
	_ = tgCtx.Send("Reply with a text")
	err = tgCtx.Send(ReqMsgPubBasic, publishBasicMarkup)
	return
}

func PublishBasicReplyHandlerFunc(
	clientAwk api.Client,
	groupId string,
	svcMsgs messages.Service,
	cfgPayment config.PaymentConfig,
	kbd *telebot.ReplyMarkup,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		sender := tgCtx.Sender()
		userId := strconv.FormatInt(sender.ID, 10)
		w, err := clientAwk.OpenMessagesWriter(groupIdCtx, userId)
		var evt pb.CloudEvent
		if err == nil {
			defer w.Close()
			err = toCloudEvent(sender, tgCtx.Message(), args[1], &evt)
		}
		if err == nil {
			err = publish(tgCtx, w, &evt, svcMsgs, cfgPayment, kbd)
		}
		return
	}
}

func toCloudEvent(sender *telebot.User, msg *telebot.Message, txt string, evt *pb.CloudEvent) (err error) {
	evt.Id = uuid.NewString()
	evt.Source = fmt.Sprintf(fmtLinkUser, sender.ID)
	evt.SpecVersion = attrValSpecVersion
	evt.Type = attrValType
	evt.Attributes = map[string]*pb.CloudEventAttributeValue{
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
	}
	if txt != "" {
		evt.Data = &pb.CloudEvent_TextData{
			TextData: txt,
		}
	}
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
		evt.Data = &pb.CloudEvent_TextData{
			TextData: msg.Audio.Caption,
		}
		f = msg.Audio.File
	case msg.Document != nil:
		evt.Attributes[attrKeyFileType] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeInteger{
				CeInteger: int32(FileTypeDocument),
			},
		}
		evt.Data = &pb.CloudEvent_TextData{
			TextData: msg.Document.Caption,
		}
		f = msg.Document.File
	case msg.Photo != nil:
		evt.Attributes[attrKeyFileType] = &pb.CloudEventAttributeValue{
			Attr: &pb.CloudEventAttributeValue_CeInteger{
				CeInteger: int32(FileTypeImage),
			},
		}
		evt.Data = &pb.CloudEvent_TextData{
			TextData: msg.Photo.Caption,
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
		evt.Data = &pb.CloudEvent_TextData{
			TextData: msg.Video.Caption,
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
	if evt.Data == nil {
		err = errors.New("message text or file caption is empty")
	}
	return
}

func PublishCustomHandlerFunc(
	clientAwk api.Client,
	groupId string,
	svcMsgs messages.Service,
	cfgPayment config.PaymentConfig,
) service.ArgHandlerFunc {
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
		if err == nil {
			evt.Source = fmt.Sprintf(fmtLinkUser, tgCtx.Sender().ID)
			evt.SpecVersion = attrValSpecVersion
			evt.Type = attrValType
			err = publish(tgCtx, w, &evt, svcMsgs, cfgPayment, nil)
		}
		return
	}
}

func publish(
	tgCtx telebot.Context,
	w model.Writer[*pb.CloudEvent],
	evt *pb.CloudEvent,
	svcMsgs messages.Service,
	cfgPayment config.PaymentConfig,
	kbd *telebot.ReplyMarkup,
) (err error) {
	var ackCount uint32
	ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
	switch {
	case ackCount == 0 && errors.Is(err, limits.ErrReached):
		ackCount, err = publishInvoice(tgCtx, evt, svcMsgs, cfgPayment, kbd)
	case ackCount == 1:
		if kbd == nil {
			err = tgCtx.Send(fmt.Sprintf(msgFmtPublished, evt.Id), telebot.ModeHTML)
		} else {
			err = tgCtx.Send(fmt.Sprintf(msgFmtPublished, evt.Id), telebot.ModeHTML, kbd)
		}
	}
	if err == nil {
		switch ackCount {
		case 0:
			if kbd == nil {
				err = tgCtx.Send(msgBusy)
			} else {
				err = tgCtx.Send(msgBusy, kbd)
			}
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
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
			userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
			w, err = clientAwk.OpenMessagesWriter(groupIdCtx, userId)
		}
		if err == nil {
			defer w.Close()
			b := backoff.NewExponentialBackOff()
			b.InitialInterval = cfgBackoff.Init
			b.Multiplier = cfgBackoff.Factor
			b.MaxElapsedTime = cfgBackoff.LimitTotal
			ew := evtWriter{
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
