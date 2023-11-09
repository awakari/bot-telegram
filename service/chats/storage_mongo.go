package chats

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type storageMongo struct {
	conn *mongo.Client
	db   *mongo.Database
	coll *mongo.Collection
}

type Chat struct {
	Id      int64  `bson:"id"`
	SubId   string `bson:"subId"`
	GroupId string `bson:"groupId"`
	UserId  string `bson:"userId"`
}

const attrId = "id"
const attrSubId = "subId"
const attrGroupId = "groupId"
const attrUserId = "userId"

var optsSrvApi = options.ServerAPI(options.ServerAPIVersion1)
var indices = []mongo.IndexModel{
	{
		Keys: bson.D{
			{
				Key:   attrId,
				Value: 1,
			},
		},
		Options: options.
			Index().
			SetUnique(false),
	},
	{
		Keys: bson.D{
			{
				Key:   attrSubId,
				Value: 1,
			},
		},
		Options: options.
			Index().
			SetUnique(true),
	},
}
var projGet = bson.D{
	{
		Key:   attrId,
		Value: 1,
	},
	{
		Key:   attrGroupId,
		Value: 1,
	},
	{
		Key:   attrUserId,
		Value: 1,
	},
}
var optsGet = options.
	FindOne().
	SetShowRecordID(false).
	SetProjection(projGet)
var sortGetBatch = bson.D{
	{
		Key:   attrId,
		Value: 1,
	},
}
var projGetBatch = bson.D{
	{
		Key:   attrId,
		Value: 1,
	},
	{
		Key:   attrSubId,
		Value: 1,
	},
	{
		Key:   attrGroupId,
		Value: 1,
	},
	{
		Key:   attrUserId,
		Value: 1,
	},
}

func NewStorage(ctx context.Context, cfgDb config.ChatsDbConfig) (s Storage, err error) {
	clientOpts := options.
		Client().
		ApplyURI(cfgDb.Uri).
		SetServerAPIOptions(optsSrvApi)
	if cfgDb.Tls.Enabled {
		clientOpts = clientOpts.SetTLSConfig(&tls.Config{InsecureSkipVerify: cfgDb.Tls.Insecure})
	}
	if len(cfgDb.UserName) > 0 {
		auth := options.Credential{
			Username:    cfgDb.UserName,
			Password:    cfgDb.Password,
			PasswordSet: len(cfgDb.Password) > 0,
		}
		clientOpts = clientOpts.SetAuth(auth)
	}
	conn, err := mongo.Connect(ctx, clientOpts)
	var sm storageMongo
	if err == nil {
		db := conn.Database(cfgDb.Name)
		coll := db.Collection(cfgDb.Table.Name)
		sm.conn = conn
		sm.db = db
		sm.coll = coll
		_, err = sm.ensureIndices(ctx)
	}
	if err == nil {
		s = sm
	}
	return
}

func (sm storageMongo) ensureIndices(ctx context.Context) ([]string, error) {
	return sm.coll.Indexes().CreateMany(ctx, indices)
}

func (sm storageMongo) Close() error {
	return sm.conn.Disconnect(context.TODO())
}

func (sm storageMongo) LinkSubscription(ctx context.Context, c Chat) (err error) {
	_, err = sm.coll.InsertOne(ctx, c)
	err = decodeMongoError(err)
	return
}

func (sm storageMongo) GetSubscriptionLink(ctx context.Context, subId string) (c Chat, err error) {
	q := bson.M{
		attrSubId: subId,
	}
	var result *mongo.SingleResult
	result = sm.coll.FindOne(ctx, q, optsGet)
	err = result.Err()
	if err == nil {
		err = result.Decode(&c)
	}
	return
}

func (sm storageMongo) UnlinkSubscription(ctx context.Context, subId string) (err error) {
	q := bson.M{
		attrSubId: subId,
	}
	_, err = sm.coll.DeleteOne(ctx, q)
	err = decodeMongoError(err)
	return
}

func (sm storageMongo) Delete(ctx context.Context, id int64) (count int64, err error) {
	fmt.Printf("delete all persisted chat %d links\n", id)
	q := bson.M{
		attrId: id,
	}
	var result *mongo.DeleteResult
	result, err = sm.coll.DeleteMany(ctx, q)
	if result != nil {
		count = result.DeletedCount
	}
	err = decodeMongoError(err)
	return
}

func (sm storageMongo) GetBatch(ctx context.Context, idRem, idDiv uint32, limit uint32, cursor int64) (page []Chat, err error) {
	q := bson.M{
		attrId: bson.M{
			"$gt": cursor,
			"$mod": bson.A{
				idDiv,
				-int32(idRem),
			},
		},
	}
	optsList := options.
		Find().
		SetLimit(int64(limit)).
		SetShowRecordID(false).
		SetSort(sortGetBatch).
		SetProjection(projGetBatch)
	var cur *mongo.Cursor
	cur, err = sm.coll.Find(ctx, q, optsList)
	if err == nil {
		var rec Chat
		for cur.Next(ctx) {
			err = errors.Join(err, cur.Decode(&rec))
			if err == nil {
				page = append(page, rec)
			}
		}
	}
	err = decodeMongoError(err)
	return
}

func decodeMongoError(src error) (dst error) {
	switch {
	case src == nil:
	case mongo.IsDuplicateKeyError(src):
		dst = fmt.Errorf("%w: %s", ErrAlreadyExists, src)
	case errors.Is(src, mongo.ErrNoDocuments):
		dst = ErrNotFound
	default:
		dst = fmt.Errorf("%w: %s", ErrInternal, src)
	}
	return
}
