package subscriptions

import (
	"context"
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

const msgFmtDetails = `Subscription Details:
Id: <pre>%s</pre>
Description: <pre>%s</pre>
Expires: <pre>%s</pre>
Condition:
<pre>%s</pre>`

func DetailsHandlerFunc(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		var sd subscription.Data
		sd, err = clientAwk.ReadSubscription(groupIdCtx, userId, subId)
		if err == nil {
			m := &telebot.ReplyMarkup{}
			rows := []telebot.Row{
				m.Row(
					telebot.Btn{
						Text: "🔗 Link Chat",
						URL:  fmt.Sprintf("https://t.me/AwakariSubscriptionsBot?startgroup=%s", subId),
					},
					telebot.Btn{
						Text: "❌ Delete",
						Data: fmt.Sprintf("%s %s", CmdDelete, subId),
					},
				),
			}
			btns := []telebot.Btn{
				{
					Text: "🏷 Describe",
					Data: fmt.Sprintf("%s %s", CmdDescription, subId),
				},
			}
			switch sd.Enabled {
			case false:
				btns = append(btns, telebot.Btn{
					Text: "🔉 Enable",
					Data: fmt.Sprintf("%s %s", CmdEnable, subId),
				})
			default:
				btns = append(btns, telebot.Btn{
					Text: "🔇 Disable",
					Data: fmt.Sprintf("%s %s", CmdDisable, subId),
				})
			}
			rows = append(rows, m.Row(btns...))
			m.Inline(rows...)
			condJsonTxt := protojson.Format(encodeCondition(sd.Condition))
			var expires string
			switch {
			case sd.Expires.IsZero():
				expires = "never"
			default:
				expires = sd.Expires.Format(time.RFC3339)
			}
			_ = tgCtx.Send(fmt.Sprintf(msgFmtDetails, subId, sd.Description, expires, condJsonTxt), m, telebot.ModeHTML)
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
