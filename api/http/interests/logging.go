package interests

import (
	"context"
	"fmt"
	apiGrpc "github.com/awakari/bot-telegram/api/grpc/interests"
	"github.com/awakari/bot-telegram/model/interest"
	"github.com/awakari/bot-telegram/util"
	"log/slog"
)

type logging struct {
	svc Service
	log *slog.Logger
}

func NewLogging(svc Service, log *slog.Logger) Service {
	return logging{
		svc: svc,
		log: log,
	}
}

func (l logging) Create(ctx context.Context, groupId, userId string, subData interest.Data) (id string, err error) {
	id, err = l.svc.Create(ctx, groupId, userId, subData)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("interests.Create(%s, %s, %v): %s, %s", groupId, userId, subData, id, err))
	return
}

func (l logging) Read(ctx context.Context, groupId, userId, subId string) (subData interest.Data, err error) {
	subData, err = l.svc.Read(ctx, groupId, userId, subId)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("interests.Read(%s, %s, %s): %v, %s", groupId, userId, subId, subData, err))
	return
}

func (l logging) Delete(ctx context.Context, groupId, userId, subId string) (err error) {
	err = l.svc.Delete(ctx, groupId, userId, subId)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("interests.Delete(%s, %s, %s): %s", groupId, userId, subId, err))
	return
}

func (l logging) Search(ctx context.Context, groupId, userId string, q interest.Query, cursor interest.Cursor) (page []*apiGrpc.Interest, err error) {
	page, err = l.svc.Search(ctx, groupId, userId, q, cursor)
	l.log.Log(ctx, util.LogLevel(err), fmt.Sprintf("interests.Search(%s, %s, %v, %v): %d, %s", groupId, userId, q, cursor, len(page), err))
	return
}
