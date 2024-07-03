package chats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	apiHttpReader "github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/client-sdk-go/api"
	"github.com/bytedance/sonic/utf8"
	"github.com/cenkalti/backoff/v4"
	ceProto "github.com/cloudevents/sdk-go/binding/format/protobuf/v2"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	ce "github.com/cloudevents/sdk-go/v2/event"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Handler interface {
	Confirm(ctx *gin.Context)
	DeliverMessages(ctx *gin.Context)
}

type handler struct {
	topicPrefixBase string
	format          messages.Format
	urlCallbackBase string
	svcReader       apiHttpReader.Service
	tgBot           *telebot.Bot
	clientAwk       api.Client
	groupId         string
}

const keyHubChallenge = "hub.challenge"
const keyHubTopic = "hub.topic"
const linkSelfSuffix = ">; rel=\"self\""
const keyAckCount = "X-Ack-Count"

func NewHandler(
	topicPrefixBase string,
	format messages.Format,
	urlCallbackBase string,
	svcReader apiHttpReader.Service,
	tgBot *telebot.Bot,
	clientAwk api.Client,
	groupId string,
) Handler {
	return handler{
		topicPrefixBase: topicPrefixBase,
		format:          format,
		urlCallbackBase: urlCallbackBase,
		svcReader:       svcReader,
		tgBot:           tgBot,
		clientAwk:       clientAwk,
		groupId:         groupId,
	}
}

func (h handler) Confirm(ctx *gin.Context) {
	topic := ctx.Query(keyHubTopic)
	challenge := ctx.Query(keyHubChallenge)
	if strings.HasPrefix(topic, h.topicPrefixBase+"/sub/"+apiHttpReader.FmtJson) {
		ctx.String(http.StatusOK, challenge)
	} else {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", topic))
	}
	return
}

func (h handler) DeliverMessages(ctx *gin.Context) {

	var topic string
	for k, vals := range ctx.Request.Header {
		if strings.ToLower(k) == "link" {
			for _, l := range vals {
				if strings.HasSuffix(l, linkSelfSuffix) && len(l) > len(linkSelfSuffix) {
					topic = l[1 : len(l)-len(linkSelfSuffix)]
				}
			}
		}
	}
	if topic == "" {
		ctx.String(http.StatusBadRequest, "self link header missing in the request")
		return
	}

	var subId string
	topicParts := strings.Split(topic, "/")
	topicPartsLen := len(topicParts)
	if topicPartsLen > 0 {
		subId = topicParts[topicPartsLen-1]
	}
	if subId == "" {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("invalid self link header value in the request: %s", topic))
		return
	}

	chatIdRaw := ctx.Param("chatId")
	if chatIdRaw == "" {
		ctx.String(http.StatusBadRequest, "chat id parameter is missing in the request URL")
		return
	}
	chatId, err := strconv.ParseInt(chatIdRaw, 10, 64)
	if err != nil {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("chat id parameter is not a valid integer: %s", chatIdRaw))
		return
	}

	var subDescr string
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, h.groupId)
	userId := service.PrefixUserId + chatIdRaw
	sub, _ := h.clientAwk.ReadSubscription(groupIdCtx, userId, subId)
	subDescr = sub.Description

	defer ctx.Request.Body.Close()
	var evts []*ce.Event
	err = json.NewDecoder(ctx.Request.Body).Decode(&evts)
	if err != nil {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("failed to deserialize the request payload: %s", err))
		return
	}

	var countAck uint32
	countAck, err = h.deliver(ctx, evts, subId, subDescr, chatId)
	if err == nil || countAck > 0 {
		ctx.Writer.Header().Add(keyAckCount, strconv.FormatUint(uint64(countAck), 10))
		ctx.Status(http.StatusOK)
	} else {
		ctx.String(http.StatusInternalServerError, err.Error())
	}

	return
}

func (h handler) deliver(
	ctx context.Context,
	evts []*ce.Event,
	subId string,
	subDescr string,
	chatId int64,
) (
	countAck uint32,
	err error,
) {
	tgCtx := h.tgBot.NewContext(telebot.Update{
		Message: &telebot.Message{
			Chat: &telebot.Chat{
				ID: chatId,
			},
		},
	})
	for _, evt := range evts {
		var evtProto *pb.CloudEvent
		evtProto, err = ceProto.ToProto(evt)
		var dataTxt string
		if err == nil {
			err = evt.DataAs(&dataTxt)
		}
		if err != nil {
			break
		}
		if err == nil && utf8.ValidateString(dataTxt) {
			evtProto.Data = &pb.CloudEvent_TextData{
				TextData: dataTxt,
			}
		}
		tgMsg := h.format.Convert(evtProto, subId, subDescr, messages.FormatModeHtml)
		err = tgCtx.Send(tgMsg, telebot.ModeHTML)
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
				go h.handleFloodError(ctx, tgCtx, subId, chatId, err.(telebot.FloodError).RetryAfter)
			default:
				errTb := &telebot.Error{}
				if errors.As(err, &errTb) && errTb.Code == 403 {
					fmt.Printf("Bot blocked: %s, removing the chat from the storage", err)
					urlCallback := apiHttpReader.MakeCallbackUrl(h.urlCallbackBase, chatId)
					err = h.svcReader.DeleteCallback(ctx, subId, urlCallback)
					return
				}
				fmt.Printf("Failed to send message %+v to chat %d in HTML mode, cause: %s (%s)\n", tgMsg, chatId, err, reflect.TypeOf(err))
				tgMsg = h.format.Convert(evtProto, subId, subDescr, messages.FormatModePlain)
				err = tgCtx.Send(tgMsg) // fallback: try to re-send as a plain text
			}
		}
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
				go h.handleFloodError(ctx, tgCtx, subId, chatId, err.(telebot.FloodError).RetryAfter)
			default:
				fmt.Printf("Failed to send message %+v in plain text mode, cause: %s\n", tgMsg, err)
				tgMsg = h.format.Convert(evtProto, subId, subDescr, messages.FormatModeRaw)
				err = tgCtx.Send(tgMsg) // fallback: try to re-send as a raw text w/o file attachments
			}
		}
		//
		if err == nil {
			countAck++
		}
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
				go h.handleFloodError(ctx, tgCtx, subId, chatId, err.(telebot.FloodError).RetryAfter)
			default:
				fmt.Printf("FATAL: failed to send message %+v in raw text mode, cause: %s\n", tgMsg, err)
				countAck++ // to skip
			}
			break
		}
	}
	return
}

func (h handler) handleFloodError(ctx context.Context, tgCtx telebot.Context, subId string, chatId int64, retryAfter int) {
	urlCallback := apiHttpReader.MakeCallbackUrl(h.urlCallbackBase, chatId)
	_ = h.svcReader.DeleteCallback(ctx, subId, urlCallback)
	fmt.Printf("High message rate detected for the interest %s\n", subId)
	retryDuration := time.Duration(retryAfter) * time.Second
	time.Sleep(retryDuration)
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = retryDuration
	b.MaxInterval = time.Duration(backoff.DefaultMultiplier * float64(retryDuration))
	_ = backoff.Retry(func() error {
		return tgCtx.Send(
			"⚠ High message rate detected. "+
				"Results streaming stopped to prevent a further flood. "+
				"Typical cause: interest conditions are too vague. "+
				"Review the <a href=\"https://awakari.com/sub-details.html?id="+subId+
				"\">interest</a> and make it more specific. "+
				"Link it back to a chat later using the /start command of the bot.",
			telebot.ModeHTML,
		)
	}, b)

}
