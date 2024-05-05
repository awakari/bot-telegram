package chats

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

var dbUri = os.Getenv("DB_URI_TEST_MONGO")

func TestNewStorage(t *testing.T) {
	//
	collName := fmt.Sprintf("chats-test-%d", time.Now().UnixMicro())
	dbCfg := config.ChatsDbConfig{
		Uri:  dbUri,
		Name: "bot-telegram",
	}
	dbCfg.Table.Name = collName
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	assert.NotNil(t, s)
	assert.Nil(t, err)
	//
	clear(ctx, t, s.(storageMongo))
}

func clear(ctx context.Context, t *testing.T, sm storageMongo) {
	require.Nil(t, sm.coll.Drop(ctx))
	require.Nil(t, sm.Close())
}

func TestStorageMongo_Create(t *testing.T) {
	//
	collName := fmt.Sprintf("chats-test-%d", time.Now().UnixMicro())
	dbCfg := config.ChatsDbConfig{
		Uri:  dbUri,
		Name: "bot-telegram",
	}
	dbCfg.Table.Name = collName
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.NotNil(t, s)
	require.Nil(t, err)
	sm := s.(storageMongo)
	defer clear(ctx, t, sm)
	//
	preExisting := Chat{
		Id:    -123,
		SubId: "sub0",
	}
	err = s.LinkSubscription(ctx, preExisting)
	require.Nil(t, err)
	//
	cases := map[string]struct {
		chat Chat
		err  error
	}{
		"ok": {
			chat: Chat{
				Id:    234,
				SubId: "sub1",
			},
		},
		"already exists - same subscription": {
			chat: Chat{
				Id:    345,
				SubId: "sub0",
			},
			err: ErrAlreadyExists,
		},
		"already exists - same chat id": {
			chat: Chat{
				Id:    -123,
				SubId: "sub1",
			},
			err: ErrAlreadyExists,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			err = s.LinkSubscription(ctx, c.chat)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_Delete(t *testing.T) {
	//
	collName := fmt.Sprintf("chats-test-%d", time.Now().UnixMicro())
	dbCfg := config.ChatsDbConfig{
		Uri:  dbUri,
		Name: "bot-telegram",
	}
	dbCfg.Table.Name = collName
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.NotNil(t, s)
	require.Nil(t, err)
	sm := s.(storageMongo)
	defer clear(ctx, t, sm)
	//
	preExisting0 := Chat{
		Id:    -123,
		SubId: "sub0",
	}
	err = s.LinkSubscription(ctx, preExisting0)
	require.Nil(t, err)
	//
	preExisting1 := Chat{
		Id:    -123,
		SubId: "sub1",
	}
	err = s.LinkSubscription(ctx, preExisting1)
	require.Nil(t, err)
	//
	cases := map[string]struct {
		id    int64
		count int64
		err   error
	}{
		"ok": {
			id:    -123,
			count: 2,
		},
		"not found is ok": {
			id: 234,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			count, err := s.Delete(ctx, c.id)
			assert.Equal(t, c.count, count)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestStorageMongo_GetBatch(t *testing.T) {
	//
	collName := fmt.Sprintf("chats-test-%d", time.Now().UnixMicro())
	dbCfg := config.ChatsDbConfig{
		Uri:  dbUri,
		Name: "bot-telegram",
	}
	dbCfg.Table.Name = collName
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.NotNil(t, s)
	require.Nil(t, err)
	sm := s.(storageMongo)
	defer clear(ctx, t, sm)
	//
	cases := map[string]struct {
		idRange  uint32
		idIndex  uint32
		stored   []Chat
		selected []Chat
	}{
		"ok": {
			idRange: 2,
			idIndex: 1,
			stored: []Chat{
				{
					Id:      -1001875128866,
					SubId:   "sub1",
					GroupId: "group1",
					UserId:  "user1",
				},
				{
					Id:      -1001778619305,
					SubId:   "sub2",
					GroupId: "group2",
					UserId:  "user2",
				},
				{
					Id:      -1001733378662,
					SubId:   "sub3",
					GroupId: "group3",
					UserId:  "user3",
				},
			},
			selected: []Chat{
				{
					Id:      -1001778619305,
					SubId:   "sub2",
					GroupId: "group2",
					UserId:  "user2",
				},
			},
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			for _, chat := range c.stored {
				err = s.LinkSubscription(ctx, chat)
				require.Nil(t, err)
			}
			var selected []Chat
			selected, err = s.GetBatch(ctx, c.idIndex, c.idRange, 10, "")
			assert.Equal(t, c.selected, selected)
			assert.Nil(t, err)
		})
	}
}

func TestStorageMongo_Count(t *testing.T) {
	//
	collName := fmt.Sprintf("chats-test-%d", time.Now().UnixMicro())
	dbCfg := config.ChatsDbConfig{
		Uri:  dbUri,
		Name: "bot-telegram",
	}
	dbCfg.Table.Name = collName
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	s, err := NewStorage(ctx, dbCfg)
	require.NotNil(t, s)
	require.Nil(t, err)
	sm := s.(storageMongo)
	defer clear(ctx, t, sm)
	//
	cases := map[string]struct {
		stored []Chat
		out    int64
		err    error
	}{
		"ok": {
			stored: []Chat{
				{
					Id:      -1001875128866,
					SubId:   "sub1",
					GroupId: "group1",
					UserId:  "user1",
				},
				{
					Id:      -1001778619305,
					SubId:   "sub2",
					GroupId: "group2",
					UserId:  "user2",
				},
				{
					Id:      -1001733378662,
					SubId:   "sub3",
					GroupId: "group3",
					UserId:  "user3",
				},
			},
			out: 3,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			for _, chat := range c.stored {
				err = s.LinkSubscription(ctx, chat)
				require.Nil(t, err)
			}
			count, err := s.Count(ctx)
			assert.Equal(t, c.out, count)
			assert.Nil(t, err)
		})
	}
}
