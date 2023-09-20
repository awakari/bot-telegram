package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/grpc/metadata"
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
