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
	"time"
)

type storageMongo struct {
	conn *mongo.Client
	db   *mongo.Database
	coll *mongo.Collection
}

type chatRec struct {
	Id      int64     `bson:"id"`
	SubId   string    `bson:"subId"`
	GroupId string    `bson:"groupId"`
	UserId  string    `bson:"userId"`
	State   int       `bson:"state"`
	Expires time.Time `bson:"expires"`
}

const attrId = "id"
const attrSubId = "subId"
const attrGroupId = "groupId"
const attrUserId = "userId"
const attrState = "state"
const attrExpires = "expires"

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
			SetUnique(true),
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
	{
		Keys: bson.D{
			{
				Key:   attrState,
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
				Key:   attrExpires,
				Value: 1,
			},
		},
		Options: options.
			Index().
			SetUnique(false),
	},
}
var optsActivateNext = options.
	FindOneAndUpdate().
	SetSort(bson.D{
		{
			Key:   attrSubId,
			Value: 1,
		},
	}).
	SetReturnDocument(options.After)

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

func (sm storageMongo) Create(ctx context.Context, c Chat) (err error) {
	rec := chatRec{
		Id:      c.Key.Id,
		SubId:   c.Key.SubId,
		GroupId: c.GroupId,
		UserId:  c.UserId,
		State:   int(c.State),
		Expires: c.Expires,
	}
	_, err = sm.coll.InsertOne(ctx, rec)
	err = decodeMongoError(err)
	return
}

func (sm storageMongo) Update(ctx context.Context, c Chat) (err error) {
	q := bson.M{
		attrId:    c.Key.Id,
		attrSubId: c.Key.SubId,
	}
	u := bson.M{
		"$set": bson.M{
			attrState:   c.State,
			attrExpires: c.Expires,
		},
	}
	var result *mongo.UpdateResult
	result, err = sm.coll.UpdateOne(ctx, q, u)
	if err == nil {
		if result.MatchedCount < 1 {
			err = fmt.Errorf("%w: %+v", ErrNotFound, c.Key)
		}
	} else {
		err = decodeMongoError(err)
	}
	return
}

func (sm storageMongo) Delete(ctx context.Context, id int64) (err error) {
	q := bson.M{
		attrId: id,
	}
	_, err = sm.coll.DeleteOne(ctx, q)
	err = decodeMongoError(err)
	return
}

func (sm storageMongo) ActivateNext(ctx context.Context, expiresNext time.Time) (c Chat, err error) {
	q := bson.M{
		"$or": []bson.M{
			{
				attrState: StateInactive,
			},
			{
				attrExpires: bson.M{
					"$lt": time.Now().UTC(),
				},
			},
		},
	}
	u := bson.M{
		"$set": bson.M{
			attrState:   StateActive,
			attrExpires: expiresNext,
		},
	}
	result := sm.coll.FindOneAndUpdate(ctx, q, u, optsActivateNext)
	err = result.Err()
	var rec chatRec
	if err == nil {
		err = result.Decode(&rec)
	}
	if err == nil {
		c.Key.Id = rec.Id
		c.Key.SubId = rec.SubId
		c.GroupId = rec.GroupId
		c.UserId = rec.UserId
		c.State = State(rec.State)
		c.Expires = rec.Expires
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
