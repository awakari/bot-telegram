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

var errCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text subscription")

var whiteSpaceRegex = regexp.MustCompile(`\p{Zs}+`)
var msgFmtSubCreated = `Created the new simple subscription, id:
<pre>%s</pre>
Next, you can go to a group with the bot and select this subscription by name to receive the matching messages.`

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
		var subId string
		if err == nil {
			groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
			userId := strconv.FormatInt(ctx.Sender().ID, 10)
			name := args[0]
			keywords := args[1]
			subData := subscription.Data{
				Condition: condition.NewBuilder().
					AnyOfWords(keywords).
					BuildTextCondition(),
				Description: name,
				Enabled:     true,
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

func CreateCustomHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) error {
	return func(ctx telebot.Context, args ...string) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		data := args[0]
		// TODO: fix this double decoding/encoding of the payload with copy paste decode code
		var req subscriptions.CreateRequest
		err = protojson.Unmarshal([]byte(data), &req)
		var cond condition.Condition
		if err == nil {
			cond, err = decodeCondition(req.Cond)
		}
		//
		var subId string
		if err == nil {
			subId, err = awakariClient.CreateSubscription(groupIdCtx, userId, subscription.Data{
				Condition:   cond,
				Description: req.Description,
				Enabled:     req.Enabled,
			})
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
