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
		Key: Key{
			Id:    -123,
			SubId: "sub0",
		},
		State:   StateActive,
		Expires: time.Now().Add(time.Hour),
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
				Key: Key{
					Id:    234,
					SubId: "sub1",
				},
				State:   StateActive,
				Expires: time.Now().Add(time.Hour),
			},
		},
		"already exists - same subscription": {
			chat: Chat{
				Key: Key{
					Id:    345,
					SubId: "sub0",
				},
				State:   StateActive,
				Expires: time.Now().Add(time.Hour),
			},
			err: ErrAlreadyExists,
		},
		"already exists - same chat id": {
			chat: Chat{
				Key: Key{
					Id:    -123,
					SubId: "sub1",
				},
				State:   StateActive,
				Expires: time.Now().Add(time.Hour),
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

func TestStorageMongo_Update(t *testing.T) {
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
		Key: Key{
			Id:    -123,
			SubId: "sub0",
		},
		State:   StateActive,
		Expires: time.Now().Add(time.Hour),
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
				Key: Key{
					Id:    -123,
					SubId: "sub0",
				},
				State:   StateInactive,
				Expires: time.Now().Add(time.Hour),
			},
		},
		"not found - different subscription": {
			chat: Chat{
				Key: Key{
					Id:    -123,
					SubId: "sub1",
				},
				State:   StateInactive,
				Expires: time.Now().Add(time.Hour),
			},
			err: ErrNotFound,
		},
		"not found - different chat id": {
			chat: Chat{
				Key: Key{
					Id:    234,
					SubId: "sub0",
				},
				State:   StateInactive,
				Expires: time.Now().Add(time.Hour),
			},
			err: ErrNotFound,
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			err = s.UpdateSubscriptionLink(ctx, c.chat)
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
		Key: Key{
			Id:    -123,
			SubId: "sub0",
		},
		State:   StateActive,
		Expires: time.Now().Add(time.Hour),
	}
	err = s.LinkSubscription(ctx, preExisting0)
	require.Nil(t, err)
	//
	preExisting1 := Chat{
		Key: Key{
			Id:    -123,
			SubId: "sub1",
		},
		State:   StateInactive,
		Expires: time.Now().Add(time.Hour),
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

func TestStorageMongo_ActivateNext(t *testing.T) {
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
		stored    []Chat
		activated []Chat
	}{
		"ok": {
			stored: []Chat{
				{
					Key: Key{
						Id:    1,
						SubId: "sub1",
					},
					GroupId: "group1",
					UserId:  "user1",
					State:   StateActive,
					Expires: time.Now().Add(time.Hour),
				},
				{
					Key: Key{
						Id:    2,
						SubId: "sub2",
					},
					GroupId: "group2",
					UserId:  "user2",
					State:   StateInactive,
					Expires: time.Now().Add(time.Hour),
				},
				{
					Key: Key{
						Id:    3,
						SubId: "sub3",
					},
					GroupId: "group3",
					UserId:  "user3",
					State:   StateActive,
					Expires: time.Now(),
				},
			},
			activated: []Chat{
				{
					Key: Key{
						Id:    2,
						SubId: "sub2",
					},
					GroupId: "group2",
					UserId:  "user2",
					State:   StateActive,
				},
				{
					Key: Key{
						Id:    3,
						SubId: "sub3",
					},
					GroupId: "group3",
					UserId:  "user3",
					State:   StateActive,
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
			var actual Chat
			for _, expected := range c.activated {
				actual, err = s.ActivateNext(ctx, time.Now().Add(time.Hour))
				assert.Equal(t, expected.Key, actual.Key)
				assert.Equal(t, expected.GroupId, actual.GroupId)
				assert.Equal(t, expected.UserId, actual.UserId)
				assert.Equal(t, expected.State, StateActive)
				assert.True(t, actual.Expires.After(time.Now()))
				assert.Nil(t, err)
			}
			actual, err = s.ActivateNext(ctx, time.Now().Add(time.Hour))
			assert.Zero(t, actual)
			assert.ErrorIs(t, err, ErrNotFound)
		})
	}
}
