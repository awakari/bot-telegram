package messages

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/limits"
	"github.com/awakari/client-sdk-go/api/grpc/permits"
	"github.com/awakari/client-sdk-go/api/grpc/resolver"
	"github.com/awakari/client-sdk-go/model"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/segmentio/ksuid"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"io"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type Channel struct {
	LastUpdate time.Time
	Link       string
}

type ChanFilter struct {
	Pattern string
}

type ChanPostHandler struct {
	ClientAwk api.Client
	GroupId   string
	Log       *slog.Logger
	Writers   map[string]model.Writer[*pb.CloudEvent]
	Channels  map[string]time.Time
	ChansLock *sync.Mutex
	CfgMsgs   config.MessagesConfig
}

const tagNoBot = "#nobot"

var errNoAck = errors.New("event was not accepted")

func (cp ChanPostHandler) Publish(tgCtx telebot.Context) (err error) {

	tgMsg := tgCtx.Message()
	ch := tgCtx.Chat()
	chanUserName := ch.Username

	var txt string
	switch {
	case tgMsg.Text != "":
		txt = tgMsg.Text
	case tgMsg.Caption != "":
		txt = tgMsg.Caption
	}
	for _, w := range strings.Split(txt, " ") {
		if w == tagNoBot {
			cp.Log.Warn(fmt.Sprintf("Channel %s (%d) post %d contains the %s tag, skipping", chanUserName, ch.ID, tgMsg.ID, tagNoBot))
			return
		}
	}

	chanUserId := fmt.Sprintf("@%s", chanUserName)
	evt := pb.CloudEvent{
		Id:          ksuid.New().String(),
		Source:      fmt.Sprintf("https://t.me/%s", chanUserName),
		SpecVersion: attrValSpecVersion,
		Type:        cp.CfgMsgs.Type,
	}
	err = toCloudEvent(tgMsg, tgCtx.Text(), &evt)
	if err == nil {
		err = cp.getWriterAndPublish(chanUserId, &evt)
		if err != nil {
			// retry with a backoff
			b := backoff.NewExponentialBackOff()
			b.InitialInterval = 100 * time.Millisecond
			b.MaxElapsedTime = 10 * time.Second
			err = backoff.RetryNotify(
				func() error {
					return cp.getWriterAndPublish(chanUserId, &evt)
				},
				b,
				func(err error, d time.Duration) {
					cp.Log.Warn(fmt.Sprintf("Failed to write event %s, cause: %s, retrying in %s...", evt.Id, err, d))
				},
			)
		}
	}
	return
}

func (cp ChanPostHandler) List(ctx context.Context, filter ChanFilter, limit uint32, cursor string, order Order) (page []Channel, err error) {
	//
	var count uint32
	var p *regexp.Regexp
	if filter.Pattern != "" {
		p, err = regexp.Compile(filter.Pattern)
	}
	//
	cp.ChansLock.Lock()
	defer cp.ChansLock.Unlock()
	var chansSorted []string
	for l, _ := range cp.Channels {
		chansSorted = append(chansSorted, l)
	}
	switch order {
	case OrderDesc:
		sort.Slice(chansSorted, func(i, j int) bool {
			return chansSorted[i] > chansSorted[j]
		})
	default:
		sort.Strings(chansSorted)
	}
	for _, l := range chansSorted {
		t := cp.Channels[l]
		if count == limit {
			break
		}
		if cursor != "" {
			switch order {
			case OrderDesc:
				if strings.Compare(l, cursor) >= 0 {
					continue
				}
			default:
				if strings.Compare(l, cursor) <= 0 {
					continue
				}
			}
		}
		if p != nil && !p.MatchString(l) {
			continue
		}
		page = append(page, Channel{
			LastUpdate: t,
			Link:       l,
		})
		count++
	}
	return
}

func (cp ChanPostHandler) Close() {
	cp.ChansLock.Lock()
	defer cp.ChansLock.Unlock()
	for _, w := range cp.Writers {
		_ = w.Close()
	}
	clear(cp.Writers)
}

func (cp ChanPostHandler) getWriterAndPublish(chanUserId string, evt *pb.CloudEvent) (err error) {
	var w model.Writer[*pb.CloudEvent]
	if err == nil {
		w, err = cp.getWriter(chanUserId)
	}
	if err == nil {
		err = cp.publish(w, evt)
		switch {
		case err == nil:
		case errors.Is(err, limits.ErrUnavailable):
			fallthrough
		case errors.Is(err, permits.ErrUnavailable):
			fallthrough
		case errors.Is(err, resolver.ErrUnavailable):
			fallthrough
		case errors.Is(err, io.EOF):
			cp.Log.Warn(fmt.Sprintf("Closing the failing writer stream for %s before retrying, cause: %s", chanUserId, err))
			cp.ChansLock.Lock()
			defer cp.ChansLock.Unlock()
			_ = w.Close()
			delete(cp.Writers, chanUserId)
		default:
			cp.Log.Error(fmt.Sprintf("Failed to publish event %s from channel %s, cause: %s", evt.Id, chanUserId, err))
		}
	}
	return
}

func (cp ChanPostHandler) getWriter(userId string) (w model.Writer[*pb.CloudEvent], err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, cp.GroupId)
	cp.ChansLock.Lock()
	defer cp.ChansLock.Unlock()
	var wExists bool
	w, wExists = cp.Writers[userId]
	if !wExists {
		w, err = cp.ClientAwk.OpenMessagesWriter(groupIdCtx, userId)
		if err == nil {
			cp.Channels[userId] = time.Now().UTC()
			cp.Writers[userId] = w
		}
	}
	return
}

func (cp ChanPostHandler) publish(w model.Writer[*pb.CloudEvent], evt *pb.CloudEvent) (err error) {
	evts := []*pb.CloudEvent{
		evt,
	}
	err = cp.tryWriteEventOnce(w, evts)
	if err != nil {
		// retry with a backoff
		b := backoff.NewExponentialBackOff()
		b.InitialInterval = 100 * time.Millisecond
		switch {
		case errors.Is(err, limits.ErrReached):
			err = nil // avoid the outer retry
			// spawn a shorter backoff just in case if the ResourceExhausted status is spurious, don't block
			b.MaxElapsedTime = 1 * time.Second
			go func() {
				err = backoff.RetryNotify(
					func() error {
						return cp.retryWriteRejectedEvent(w, evts)
					},
					b,
					func(err error, d time.Duration) {
						cp.Log.Warn(fmt.Sprintf("Failed to write event %s, cause: %s, retrying in %s...", evt.Id, err, d))
					},
				)
			}()
		case errors.Is(err, limits.ErrUnavailable) || errors.Is(err, permits.ErrUnavailable) || errors.Is(err, resolver.ErrUnavailable):
			// avoid retrying this before reopening the writer
		default:
			b.MaxElapsedTime = 10 * time.Second
			err = backoff.RetryNotify(
				func() error {
					return cp.tryWriteEventOnce(w, evts)
				},
				b,
				func(err error, d time.Duration) {
					cp.Log.Warn(fmt.Sprintf("Failed to write event %s, cause: %s, retrying in %s...", evt.Id, err, d))
				},
			)
		}
	}
	return
}

func (cp ChanPostHandler) retryWriteRejectedEvent(w model.Writer[*pb.CloudEvent], evts []*pb.CloudEvent) (err error) {
	var ackCount uint32
	ackCount, err = w.WriteBatch(evts)
	if err == nil && ackCount < 1 {
		err = errNoAck // it's an error to retry
	}
	if !errors.Is(err, limits.ErrReached) {
		cp.Log.Debug(fmt.Sprintf("Dropping the rejected event %s from %s, cause: %s", evts[0].Id, evts[0].Source, err))
		err = nil // stop retrying
	}
	return
}

func (cp ChanPostHandler) tryWriteEventOnce(w model.Writer[*pb.CloudEvent], evts []*pb.CloudEvent) (err error) {
	var ackCount uint32
	ackCount, err = w.WriteBatch(evts)
	if err == nil && ackCount < 1 {
		err = errNoAck // it's an error to retry
	}
	return
}
