package subscriptions

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

const CmdDetails = "details"

func DetailsHandlerFunc(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var sd subscription.Data
		sd, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			// id, delete and condition button
			m := &telebot.ReplyMarkup{}
			condJsonUrl := base64.URLEncoding.EncodeToString([]byte(protojson.Format(encodeCondition(sd.Condition))))
			m.Inline(m.Row(
				telebot.Btn{
					Text: "❌ Delete",
					Data: fmt.Sprintf("%s %s", CmdDelete, subId),
				},
				telebot.Btn{
					Text: "🔎 🔍 Condition",
					WebApp: &telebot.WebApp{
						URL: fmt.Sprintf("https://awakari.app/sub-cond.html?cond=%s", condJsonUrl),
					},
				},
			))
			_ = tgCtx.Send(fmt.Sprintf("Subscription: %s", subId), m)
			// description: change
			m = &telebot.ReplyMarkup{}
			m.Inline(m.Row(telebot.Btn{
				Text: "Change",
				Data: fmt.Sprintf("%s %s", CmdDescription, subId),
			}))
			_ = tgCtx.Send(fmt.Sprintf("Description: %s", sd.Description), m)
			// expires: extend
			m = &telebot.ReplyMarkup{}
			var expires string
			switch {
			case sd.Expires.IsZero():
				expires = "never"
			default:
				expires = sd.Expires.Format(time.RFC3339)
				m.Inline(m.Row(telebot.Btn{
					Text: "▲ Extend",
					Data: fmt.Sprintf("%s %s", CmdExtend, subId),
				}))
			}
			_ = tgCtx.Send(fmt.Sprintf("Expires: %s", expires), m)
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
