package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"strings"
)

const CmdPrefixSubCreateSimplePrefix = "/sub"
const argSep = " "
const limitRootGroupOrCondChildrenCount = 4
const limitTextCondTermsLength = 256

var errCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text subscription")
var errInvalidCondition = errors.New("invalid subscription condition")

var whiteSpaceRegex = regexp.MustCompile(`\p{Zs}+`)
var msgFmtSubCreated = `Subscription created, next: 
1. Create a group chat for the created subscription. 
2. <a href="https://t.me/AwakariSubscriptionsBot?startgroup=%s">Link</a> the subscription to the group.`

func CreateSimpleHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		txt := ctx.Text()
		argStr, ok := strings.CutPrefix(txt, CmdPrefixSubCreateSimplePrefix+" ")
		if !ok {
			err = errCreateSubNotEnoughArgs
		}
		var args []string
		if err == nil {
			argStr = whiteSpaceRegex.ReplaceAllString(argStr, argSep)
			args = strings.SplitN(argStr, argSep, 2)
		}
		if len(args) < 2 {
			err = errCreateSubNotEnoughArgs
		}
		var subData subscription.Data
		if err == nil {
			name := args[0]
			keywords := args[1]
			subData.Condition = condition.NewBuilder().
				AnyOfWords(keywords).
				BuildTextCondition()
			subData.Description = name
			subData.Enabled = true
			err = validateCondition(subData.Condition, true)
		}
		var subId string
		if err == nil {
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
			userId := strconv.FormatInt(ctx.Sender().ID, 10)
			subId, err = awakariClient.CreateSubscription(groupIdCtx, userId, subData)
		}
		if err == nil {
			err = ctx.Send(fmt.Sprintf(msgFmtSubCreated, subId), telebot.ModeHTML)
		} else {
			err = fmt.Errorf("failed to create the subscription: %w", err)
		}
		return
	}
}

func CreateCustomHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) (err error) {
	return func(ctx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		data := args[0]
		// TODO: maybe fix this double decoding/encoding of the payload with copy paste decode code
		var req subscriptions.CreateRequest
		err = protojson.Unmarshal([]byte(data), &req)
		var cond condition.Condition
		if err == nil {
			cond, err = decodeCondition(req.Cond)
		}
		//
		var subId string
		if err == nil {
			subData := subscription.Data{
				Condition:   cond,
				Description: req.Description,
				Enabled:     req.Enabled,
			}
			subId, err = awakariClient.CreateSubscription(groupIdCtx, userId, subData)
		}
		if err == nil {
			err = ctx.Send(fmt.Sprintf(msgFmtSubCreated, subId), telebot.ModeHTML)
		} else {
			err = fmt.Errorf("failed to create the subscription: %w", err)
		}
		return
	}
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

func decodeNumOp(src subscriptions.Operation) (dst condition.NumOp) {
	switch src {
	case subscriptions.Operation_Gt:
		dst = condition.NumOpGt
	case subscriptions.Operation_Gte:
		dst = condition.NumOpGte
	case subscriptions.Operation_Eq:
		dst = condition.NumOpEq
	case subscriptions.Operation_Lte:
		dst = condition.NumOpLte
	case subscriptions.Operation_Lt:
		dst = condition.NumOpLt
	default:
		dst = condition.NumOpUndefined
	}
	return
}

func validateCondition(cond condition.Condition, root bool) (err error) {
	switch tc := cond.(type) {
	case condition.GroupCondition:
		children := tc.GetGroup()
		countChildren := len(children)
		if root && tc.GetLogic() == condition.GroupLogicOr && countChildren > limitRootGroupOrCondChildrenCount {
			err = fmt.Errorf(
				"%w: root group condition with logic \"Or\" child condition count is %d, limit is %d,\nconsider to use an additional subscription",
				errInvalidCondition,
				countChildren,
				limitRootGroupOrCondChildrenCount,
			)
		} else {
			for _, child := range children {
				err = validateCondition(child, false)
				if err != nil {
					break
				}
			}
		}
	case condition.TextCondition:
		lenTerms := len(tc.GetTerm())
		if lenTerms > limitTextCondTermsLength {
			err = fmt.Errorf(
				"%w: text condition terms length is %d, limit is %d,\nconsider to use an additional subscription",
				errInvalidCondition,
				lenTerms,
				limitRootGroupOrCondChildrenCount,
			)
		}
	}
	return
}
