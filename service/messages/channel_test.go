package messages

import (
	"context"
	"github.com/stretchr/testify/assert"
	"regexp/syntax"
	"sync"
	"testing"
	"time"
)

func TestChanPostHandler_List(t *testing.T) {
	cp := ChanPostHandler{
		Channels: map[string]time.Time{
			"@a":    time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
			"@bb":   time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
			"@ccc":  time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
			"@dddd": time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
		},
		ChansLock: &sync.Mutex{},
	}
	cases := map[string]struct {
		filter ChanFilter
		limit  uint32
		cursor string
		order  Order
		page   []Channel
		err    string
	}{
		"default": {
			limit: 10,
			page: []Channel{
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@a",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@bb",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@ccc",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@dddd",
				},
			},
		},
		"filter": {
			filter: ChanFilter{
				Pattern: "d",
			},
			limit: 10,
			page: []Channel{
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@dddd",
				},
			},
		},
		"filter err": {
			filter: ChanFilter{
				Pattern: "[ ]\\K(?<!\\d )",
			},
			limit: 10,
			page: []Channel{
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@a",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@bb",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@ccc",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@dddd",
				},
			},
			err: syntax.ErrInvalidEscape.String(),
		},
		"limit=2": {
			limit: 2,
			page: []Channel{
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@a",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@bb",
				},
			},
		},
		"cursor": {
			limit:  10,
			cursor: "@bb",
			page: []Channel{
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@ccc",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@dddd",
				},
			},
		},
		"order": {
			limit: 10,
			order: OrderDesc,
			page: []Channel{
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@dddd",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@ccc",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@bb",
				},
				{
					LastUpdate: time.Date(2023, 12, 29, 13, 22, 10, 0, time.UTC),
					Link:       "@a",
				},
			},
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			page, err := cp.List(context.TODO(), c.filter, c.limit, c.cursor, c.order)
			assert.Equal(t, c.page, page)
			if c.err != "" {
				assert.ErrorContains(t, err, c.err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
