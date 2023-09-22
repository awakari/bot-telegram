package events

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const readBatchSize = 16

func ViewInboxHandlerFunc(awakariClient api.Client, groupId string) telegram.ArgHandlerFunc {
	return func(ctx telebot.Context, args ...string) (err error) {
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		subId := args[0]
		go readAndSendEventsLoop(ctx, awakariClient, groupId, userId, subId)
		return
	}
}

func readAndSendEventsLoop(ctx telebot.Context, awakariClient api.Client, groupId, userId, subId string) {
	for {
		err := readAndSentEventsOnce(ctx, awakariClient, groupId, userId, subId)
		if err != nil {
			_ = ctx.Send(fmt.Sprintf("Failed to open the events stream, try again later: %s", err.Error()))
			break // TODO backoff instead
		}
	}
}

func readAndSentEventsOnce(ctx telebot.Context, awakariClient api.Client, groupId, userId, subId string) (err error) {
	groupIdCtx, cancel := context.WithTimeout(context.TODO(), 1*time.Minute)
	defer cancel()
	groupIdCtx = metadata.AppendToOutgoingContext(groupIdCtx, "x-awakari-group-id", groupId)
	r, err := awakariClient.OpenMessagesReader(groupIdCtx, userId, subId, readBatchSize)
	if err == nil {
		defer r.Close()
		fmt.Printf("New reader for subscription %s\n", subId)
		readAndSendEvents(ctx, r)
		fmt.Printf("Done reader for subscription %s\n", subId)
	}
	return
}

func readAndSendEvents(ctx telebot.Context, r model.Reader[[]*pb.CloudEvent]) {
	for {
		evts, err := r.Read()
		if len(evts) > 0 {
			sendEvents(ctx, evts)
		}
		if err != nil {
			break
		}
	}
}

func sendEvents(ctx telebot.Context, evts []*pb.CloudEvent) {
	for _, evt := range evts {
		_ = ctx.Send(evt.String())
	}
}
