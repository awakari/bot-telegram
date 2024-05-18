package chats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	apiHttpReader "github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/bytedance/sonic/utf8"
	ceProto "github.com/cloudevents/sdk-go/binding/format/protobuf/v2"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	ce "github.com/cloudevents/sdk-go/v2/event"
	"github.com/gin-gonic/gin"
	"gopkg.in/telebot.v3"
	"net/http"
	"reflect"
	"strconv"
	"strings"
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
) Handler {
	return handler{
		topicPrefixBase: topicPrefixBase,
		format:          format,
		urlCallbackBase: urlCallbackBase,
		svcReader:       svcReader,
		tgBot:           tgBot,
	}
}

func (h handler) Confirm(ctx *gin.Context) {
	topic := ctx.Query(keyHubTopic)
	challenge := ctx.Query(keyHubChallenge)
	if strings.HasSuffix(topic, h.topicPrefixBase+"/"+apiHttpReader.FmtJson) {
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

	defer ctx.Request.Body.Close()
	var evts []*ce.Event
	err = json.NewDecoder(ctx.Request.Body).Decode(&evts)
	if err != nil {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("failed to deserialize the request payload: %s", err))
		return
	}

	var countAck uint32
	countAck, err = h.deliver(ctx, evts, subId, chatId)
	if err == nil || countAck > 0 {
		ctx.Writer.Header().Add(keyAckCount, strconv.FormatUint(uint64(countAck), 10))
		ctx.Status(http.StatusOK)
	} else {
		ctx.String(http.StatusInternalServerError, err.Error())
	}

	return
}

func (h handler) deliver(ctx context.Context, evts []*ce.Event, subId string, chatId int64) (countAck uint32, err error) {
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
		dataTxt := string(evt.Data())
		if utf8.ValidateString(dataTxt) {
			if strings.HasPrefix(dataTxt, "\"") {
				dataTxt = dataTxt[1:]
			}
			if strings.HasSuffix(dataTxt, "\"") {
				dataTxt = dataTxt[:len(dataTxt)-1]
			}
			evtProto.Data = &pb.CloudEvent_TextData{
				TextData: dataTxt,
			}
		}
		if err != nil {
			break
		}
		tgMsg := h.format.Convert(evtProto, subId, messages.FormatModeHtml)
		err = tgCtx.Send(tgMsg, telebot.ModeHTML)
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
			default:
				errTb := &telebot.Error{}
				if errors.As(err, &errTb) && errTb.Code == 403 {
					fmt.Printf("Bot blocked: %s, removing the chat from the storage", err)
					urlCallback := apiHttpReader.MakeCallbackUrl(h.urlCallbackBase, chatId)
					err = h.svcReader.DeleteCallback(ctx, subId, urlCallback)
					return
				}
				fmt.Printf("Failed to send message %+v to chat %d in HTML mode, cause: %s (%s)\n", tgMsg, chatId, err, reflect.TypeOf(err))
				tgMsg = h.format.Convert(evtProto, subId, messages.FormatModePlain)
				err = tgCtx.Send(tgMsg) // fallback: try to re-send as a plain text
			}
		}
		if err != nil {
			switch err.(type) {
			case telebot.FloodError:
			default:
				fmt.Printf("Failed to send message %+v in plain text mode, cause: %s\n", tgMsg, err)
				tgMsg = h.format.Convert(evtProto, subId, messages.FormatModeRaw)
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
			default:
				fmt.Printf("FATAL: failed to send message %+v in raw text mode, cause: %s\n", tgMsg, err)
				countAck++ // to skip
			}
			break
		}
	}
	return
}
