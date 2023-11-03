package subscriptions

import (
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
)

type ConditionHandler struct {
	ClientAwk api.Client
	GroupId   string
}

func (ch ConditionHandler) Update(tgCtx telebot.Context, args ...string) (err error) {
	condProto := &subscriptions.Condition{}
	err = convertConditionJsonToProto(args[0], condProto)
	//var cond condition.Condition
	//if err == nil {
	//    cond, err = decodeCondition(condProto)
	//}
	//
	//if err == nil {
	//    ch.ClientAwk.UpdateSubscription()
	//}
	return
}

func convertConditionJsonToProto(condJson string, condProto *subscriptions.Condition) (err error) {
	err = protojson.Unmarshal([]byte(condJson), condProto)
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
