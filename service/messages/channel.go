package messages

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/pub"
	"github.com/awakari/bot-telegram/config"
	"github.com/cenkalti/backoff/v4"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/segmentio/ksuid"
	"gopkg.in/telebot.v3"
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
	SvcPub    pub.Service
	GroupId   string
	Log       *slog.Logger
	Channels  map[string]time.Time
	ChansLock *sync.Mutex
	CfgMsgs   config.MessagesConfig
}

const tagNoBot = "#nobot"

func (cp ChanPostHandler) Publish(tgCtx telebot.Context, chanUserName string) (err error) {

	tgMsg := tgCtx.Message()
	ch := tgCtx.Chat()

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
		err = cp.SvcPub.Publish(context.TODO(), &evt, cp.GroupId, chanUserId)
		if err != nil {
			// retry with a backoff
			b := backoff.NewExponentialBackOff()
			b.InitialInterval = 100 * time.Millisecond
			b.MaxElapsedTime = 10 * time.Second
			err = backoff.RetryNotify(
				func() error {
					return cp.SvcPub.Publish(context.TODO(), &evt, cp.GroupId, chanUserId)
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
