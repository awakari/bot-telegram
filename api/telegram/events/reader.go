package events

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/chats"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/telebot.v3"
	"sync"
	"time"
)

type Reader interface {
	Run(ctx context.Context)
}

const ReaderTtl = 24 * time.Hour
const readBatchSize = 16
const readerBackoff = 1 * time.Minute

var runtimeReaders = make(map[int64]*reader)
var runtimeReadersLock = &sync.Mutex{}

func StopAllReaders() {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	for _, r := range runtimeReaders {
		r.stop = true
	}
}

func StopChatReader(chatId int64) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	r, ok := runtimeReaders[chatId]
	if ok {
		r.stop = true
	}
}

func NewReader(tgCtx telebot.Context, client api.Client, chatStor chats.Storage, chatKey chats.Key, groupId, userId string) Reader {
	return &reader{
		tgCtx:    tgCtx,
		client:   client,
		chatStor: chatStor,
		chatKey:  chatKey,
		groupId:  groupId,
		userId:   userId,
	}
}

type reader struct {
	tgCtx    telebot.Context
	client   api.Client
	chatStor chats.Storage
	chatKey  chats.Key
	groupId  string
	userId   string
	stop     bool
}

func (r *reader) Run(ctx context.Context) {
	r.runtimeRegister(ctx)
	defer r.runtimeUnregister(ctx)
	//
	for !r.stop {
		err := r.runOnce(ctx)
		if err != nil {
			_ = r.tgCtx.Send(
				fmt.Sprintf(
					`unexpected failure: %s,
to recover: try to create a new chat and select the same subscription`,
					err,
				),
			)
			_ = r.chatStor.Delete(ctx, r.chatKey.Id)
			break
		}
	}
}

func (r *reader) runOnce(ctx context.Context) (err error) {
	groupIdCtx, cancel := context.WithTimeout(ctx, ReaderTtl)
	defer cancel()
	groupIdCtx = metadata.AppendToOutgoingContext(groupIdCtx, "x-awakari-group-id", r.groupId)
	var awakariReader model.Reader[[]*pb.CloudEvent]
	awakariReader, err = r.client.OpenMessagesReader(groupIdCtx, r.userId, r.chatKey.SubId, readBatchSize)
	switch err {
	case nil:
		defer awakariReader.Close()
		err = r.deliverEventsReadLoop(ctx, awakariReader)
	default:
		_ = r.tgCtx.Send(fmt.Sprintf("unexpected failure: %s,\nretrying after %s", err, readerBackoff))
		time.Sleep(readerBackoff)
		err = nil
	}
	if err == nil {
		nextChatState := chats.Chat{
			Key:     r.chatKey,
			Expires: time.Now().Add(ReaderTtl),
			State:   chats.StateActive,
		}
		err = r.chatStor.Update(ctx, nextChatState)
	}
	return
}

func (r *reader) deliverEventsReadLoop(ctx context.Context, awakariReader model.Reader[[]*pb.CloudEvent]) (err error) {
	for !r.stop {
		err = r.deliverEventsRead(ctx, awakariReader)
		if err != nil {
			break
		}
	}
	return
}

func (r *reader) deliverEventsRead(ctx context.Context, awakariReader model.Reader[[]*pb.CloudEvent]) (err error) {
	//
	var evts []*pb.CloudEvent
	evts, err = awakariReader.Read()
	//
	switch status.Code(err) {
	case codes.NotFound:
		_ = r.chatStor.Delete(ctx, r.chatKey.Id)
	}
	//
	if len(evts) > 0 {
		_ = r.deliverEvents(evts)
	}
	//
	return err
}

func (r *reader) deliverEvents(evts []*pb.CloudEvent) (err error) {
	for _, evt := range evts {
		err = r.tgCtx.Send(formatHtmlEvent(evt), telebot.ModeHTML)
		if err != nil || r.stop {
			break
		}
	}
	return
}

func formatHtmlEvent(evt *pb.CloudEvent) (txt string) {

	title, titleOk := evt.Attributes["title"]
	if titleOk {
		txt += fmt.Sprintf("<p><b>%s</b></p>", title)
	}

	txt += fmt.Sprintf("<p>From: %s</p>", evt.Attributes["awakarigroupid"])

	urlSrc := evt.Source
	rssItemGuid, rssItemGuidOk := evt.Attributes["rssitemguid"]
	if rssItemGuidOk {
		urlSrc = rssItemGuid.GetCeString()
	}
	txt += fmt.Sprintf("<p><a href=\"%s\">Source Link</a></p>", urlSrc)

	summary, summaryOk := evt.Attributes["summary"]
	if summaryOk {
		txt += fmt.Sprintf("<p>%s</p>", summary)
	}

	txtData := evt.GetTextData()
	switch {
	case txtData != "":
		txt += fmt.Sprintf("<p>%s</p>", txtData)
	}

	urlImg, urlImgOk := evt.Attributes["imageurl"]
	if !urlImgOk {
		urlImg, urlImgOk = evt.Attributes["feedimageurl"]
	}
	if urlImgOk {
		switch {
		case urlImg.GetCeString() != "":
			txt += fmt.Sprintf("<p><a href=\"%s\" alt=\"image\">  </a></p>", urlImg.GetCeString())
		case urlImg.GetCeUri() != "":
			txt += fmt.Sprintf("<p><a href=\"%s\" alt=\"image\">  </a></p>", urlImg.GetCeUri())
		}
	}

	return
}

func (r *reader) runtimeRegister(_ context.Context) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	runtimeReaders[r.chatKey.Id] = r
}

func (r *reader) runtimeUnregister(ctx context.Context) {
	runtimeReadersLock.Lock()
	defer runtimeReadersLock.Unlock()
	delete(runtimeReaders, r.chatKey.Id)
	// try to do the best effort to mark chat as inactive in the chats DB
	_ = r.chatStor.Update(ctx, chats.Chat{
		Key:   r.chatKey,
		State: chats.StateInactive,
	})
}
