package subscriptions

import (
	"context"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"strings"
	"unicode/utf8"
)

const CmdList = "list"
const subListLimit = 256 // TODO: implement the proper pagination

func ListHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		var subIds []string
		subIds, err = awakariClient.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
		if err == nil {
			var sub subscription.Data
			for _, subId := range subIds {
				sub, err = awakariClient.ReadSubscription(groupIdCtx, userId, subId)
				if err != nil {
					break
				}
				m := &telebot.ReplyMarkup{}
				m.Inline(m.Row(
					telebot.Btn{
						Text: "🔗 Link Chat",
						URL:  fmt.Sprintf("https://t.me/AwakariSubscriptionsBot?startgroup=%s", subId),
					},
					telebot.Btn{
						Text: "🔎 Details",
						Data: fmt.Sprintf("%s %s", CmdDetails, subId),
					},
					telebot.Btn{
						Text: "❌ Delete",
						Data: fmt.Sprintf("%s %s", CmdDelete, subId),
					},
				))
				err = ctx.Send(fmt.Sprintf("<pre>%s</pre>", fixLenString(sub.Description, 31)), m, telebot.ModeHTML)
			}
		}
		return
	}
}

func fixLenString(s string, l int) string {
	if len(s) <= l {
		return padString(s, l)
	}
	// truncate, ensure we don't split a UTF-8 character in the middle.
	for i := l - 3; i > 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i] + "..."
		}
	}
	return ""
}

func padString(input string, length int) string {
	padLength := length - len(input)
	if padLength > 0 {
		// Pad the input string with spaces up to the specified length.
		return input + strings.Repeat("_", padLength)
	}
	return input
}
