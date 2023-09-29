package subscriptions

import (
	"context"
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

const CmdDetails = "details"

const CmdDescription = "description"
const CmdEnabled = "enabled"

func DetailsHandlerFunc(awakariClient api.Client, groupId string) func(ctx telebot.Context, args ...string) (err error) {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var sd subscription.Data
		sd, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			mDescription := &telebot.ReplyMarkup{}
			mDescription.Row(telebot.Btn{
				Text: "✏ Edit",
				Data: fmt.Sprintf("%s %s", CmdDescription, subId),
			})
			_ = tgCtx.Send(fmt.Sprintf("Description: <pre>%s</pre>", sd.Description), mDescription, telebot.ModeHTML)
			mEnabled := &telebot.ReplyMarkup{}
			mEnabled.Row(telebot.Btn{
				Text: "✏ Edit",
				Data: fmt.Sprintf("%s %s", CmdEnabled, subId),
			})
			_ = tgCtx.Send(fmt.Sprintf("Enabled: <pre>%t</pre>", sd.Enabled), mEnabled, telebot.ModeHTML)
			condJsonTxt := protojson.Format(encodeCondition(sd.Condition))
			_ = tgCtx.Send(fmt.Sprintf("Condition:\n<pre>%s</pre>", condJsonTxt), telebot.ModeHTML)
		}
		return
	}
}

func encodeCondition(src condition.Condition) (dst *subscriptions.Condition) {
	dst = &subscriptions.Condition{
		Not: src.IsNot(),
	}
	switch c := src.(type) {
	case condition.GroupCondition:
		var dstGroup []*subscriptions.Condition
		for _, childSrc := range c.GetGroup() {
			childDst := encodeCondition(childSrc)
			dstGroup = append(dstGroup, childDst)
		}
		dst.Cond = &subscriptions.Condition_Gc{
			Gc: &subscriptions.GroupCondition{
				Logic: subscriptions.GroupLogic(c.GetLogic()),
				Group: dstGroup,
			},
		}
	case condition.TextCondition:
		dst.Cond = &subscriptions.Condition_Tc{
			Tc: &subscriptions.TextCondition{
				Key:   c.GetKey(),
				Term:  c.GetTerm(),
				Exact: c.IsExact(),
			},
		}
	case condition.NumberCondition:
		dstOp := encodeNumOp(c.GetOperation())
		dst.Cond = &subscriptions.Condition_Nc{
			Nc: &subscriptions.NumberCondition{
				Key: c.GetKey(),
				Op:  dstOp,
				Val: c.GetValue(),
			},
		}
	}
	return
}

func encodeNumOp(src condition.NumOp) (dst subscriptions.Operation) {
	switch src {
	case condition.NumOpGt:
		dst = subscriptions.Operation_Gt
	case condition.NumOpGte:
		dst = subscriptions.Operation_Gte
	case condition.NumOpEq:
		dst = subscriptions.Operation_Eq
	case condition.NumOpLte:
		dst = subscriptions.Operation_Lte
	case condition.NumOpLt:
		dst = subscriptions.Operation_Lt
	default:
		dst = subscriptions.Operation_Undefined
	}
	return
}
