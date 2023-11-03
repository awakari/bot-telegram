package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"strconv"
)

type ConditionHandler struct {
	ClientAwk api.Client
	GroupId   string
}

type subPayload struct {
	Id string `json:"id"`
}

var jsonToProtoOpts = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

func (ch ConditionHandler) Update(tgCtx telebot.Context, args ...string) (err error) {
	payload := []byte(args[0])
	var sp subPayload
	err = json.Unmarshal(payload, &sp)
	condProto := &subscriptions.Condition{}
	if err == nil {
		err = convertConditionJsonToProto(payload, condProto)
	}
	var newCond condition.Condition
	if err == nil {
		newCond, err = decodeCondition(condProto)
	}
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "X-Awakari-Group-Id", ch.GroupId)
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	subId := sp.Id
	var sd subscription.Data
	if err == nil {
		sd, err = ch.ClientAwk.ReadSubscription(groupIdCtx, userId, subId)
	}
	if err == nil {
		sd.Condition = newCond
		fmt.Printf("update subscription condition: %+v\n", sd.Condition)
		err = ch.ClientAwk.UpdateSubscription(groupIdCtx, userId, subId, sd)
	}
	if err == nil {
		_ = tgCtx.Send("Subscription updated.")
	}
	return
}

func convertConditionJsonToProto(in []byte, out *subscriptions.Condition) (err error) {
	err = jsonToProtoOpts.Unmarshal(in, out)
	return
}

func decodeCondition(src *subscriptions.Condition) (dst condition.Condition, err error) {
	gc, nc, tc := src.GetGc(), src.GetNc(), src.GetTc()
	switch {
	case gc != nil:
		var group []condition.Condition
		var childDst condition.Condition
		for _, childSrc := range gc.Group {
			childDst, err = decodeCondition(childSrc)
			if err != nil {
				break
			}
			group = append(group, childDst)
		}
		if err == nil {
			dst = condition.NewGroupCondition(
				condition.NewCondition(src.Not),
				condition.GroupLogic(gc.GetLogic()),
				group,
			)
		}
	case nc != nil:
		dstOp := decodeNumOp(nc.Op)
		dst = condition.NewNumberCondition(
			condition.NewKeyCondition(condition.NewCondition(src.Not), nc.GetKey()),
			dstOp, nc.Val,
		)
	case tc != nil:
		dst = condition.NewTextCondition(
			condition.NewKeyCondition(condition.NewCondition(src.Not), tc.GetKey()),
			tc.GetTerm(), tc.GetExact(),
		)
	default:
		err = fmt.Errorf("unsupported condition type: %+v", src)
	}
	return
}
