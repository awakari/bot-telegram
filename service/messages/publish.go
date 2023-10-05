package messages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/messages"
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
const msgLimitReached = `Message daily publishing limit reached. 
Payment is required to proceed.
The message is saved for 1 week.`
const pricePublish = 1.0
const msgFmtPublishMissing = "message to publish is missing: %s"
const msgFmtRunOnceFailed = "failed to publish event: %s, cause: %s, retrying in: %s"

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
	paymentProviderToken string,
	kbd *telebot.ReplyMarkup,
) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		sender := tgCtx.Sender()
		userId := strconv.FormatInt(sender.ID, 10)
		w, err := clientAwk.OpenMessagesWriter(groupIdCtx, userId)
		var evt *pb.CloudEvent
		if err == nil {
			defer w.Close()
			evt = toCloudEvent(sender, tgCtx.Message(), args[1])
			err = publish(tgCtx, w, evt, svcMsgs, paymentProviderToken, kbd)
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

func PublishCustomHandlerFunc(
	clientAwk api.Client,
	groupId string,
	svcMsgs messages.Service,
	paymentProviderToken string,
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
			err = publish(tgCtx, w, &evt, svcMsgs, paymentProviderToken, nil)
		}
		return
	}
}

func publish(
	tgCtx telebot.Context,
	w model.Writer[*pb.CloudEvent],
	evt *pb.CloudEvent,
	svcMsgs messages.Service,
	paymentProviderToken string,
	kbd *telebot.ReplyMarkup,
) (err error) {
	var ackCount uint32
	ackCount, err = w.WriteBatch([]*pb.CloudEvent{evt})
	switch {
	case ackCount == 0 && errors.Is(err, limits.ErrReached):
		ackCount, err = publishPaid(tgCtx, evt, svcMsgs, paymentProviderToken, kbd)
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

func publishPaid(
	tgCtx telebot.Context,
	evt *pb.CloudEvent,
	svcMsgs messages.Service,
	paymentProviderToken string,
	kbd *telebot.ReplyMarkup,
) (ackCount uint32, err error) {
	ackCount, err = svcMsgs.PutBatch(context.TODO(), []*pb.CloudEvent{evt})
	if ackCount == 1 {
		if kbd == nil {
			_ = tgCtx.Send(msgLimitReached, telebot.ModeHTML)
		} else {
			_ = tgCtx.Send(msgLimitReached, telebot.ModeHTML, kbd)
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
				Currency:    service.Currency,
				Prices: []telebot.Price{
					{
						Label:  label,
						Amount: int(pricePublish * service.SubCurrencyFactor),
					},
				},
				Token: paymentProviderToken,
				Total: int(pricePublish * service.SubCurrencyFactor),
			}
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
	}
	return
}

func PublishPreCheckout(svcMsgs messages.Service) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		ctx, cancel := context.WithTimeout(context.TODO(), service.PreCheckoutTimeout)
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

func PublishPayment(svcMsgs messages.Service, clientAwk api.Client, groupId string, log *slog.Logger) service.ArgHandlerFunc {
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
			ew := evtWriter{
				e: evts[0],
				w: w,
			}
			err = backoff.RetryNotify(ew.runOnce, b, func(err error, d time.Duration) {
				log.Warn(fmt.Sprintf(msgFmtRunOnceFailed, evtId, err, d))
			})
		}
		if err == nil {
			_ = tgCtx.Send(fmt.Sprintf(msgFmtPublished, evtId), telebot.ModeHTML)
			_, err = svcMsgs.DeleteBatch(context.TODO(), []string{evtId})
		}
		return
	}
}
