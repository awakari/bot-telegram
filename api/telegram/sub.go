package telegram

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

type Subscriptions struct {
	Client  api.Client
	GroupId string
}

const CmdPrefixSubCreateSimplePrefix = "/sub"
const argSep = " "

var ErrCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text subscription")

var whiteSpaceRegex = regexp.MustCompile(`\p{Zs}+`)

func (s Subscriptions) CreateTextSubscription(ctx telebot.Context) (err error) {
	txt := ctx.Text()
	argStr, ok := strings.CutPrefix(txt, CmdPrefixSubCreateSimplePrefix+" ")
	if !ok {
		err = ErrCreateSubNotEnoughArgs
	}
	var args []string
	if err == nil {
		argStr = whiteSpaceRegex.ReplaceAllString(argStr, argSep)
		args = strings.SplitN(argStr, argSep, 2)
	}
	if len(args) < 2 {
		err = ErrCreateSubNotEnoughArgs
	}
	var subId string
	if err == nil {
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", s.GroupId)
		userId := strconv.FormatInt(ctx.Sender().ID, 10)
		name := args[0]
		keywords := args[1]
		subData := subscription.Data{
			Condition: condition.
				NewBuilder().
				AnyOfWords(keywords).
				BuildTextCondition(),
			Description: name,
			Enabled:     true,
		}
		subId, err = s.Client.CreateSubscription(groupIdCtx, userId, subData)
	}
	if err == nil {
		err = ctx.Send(fmt.Sprintf("Created the new simple subscription, id: %s", subId))
	} else {
		_ = ctx.Send(fmt.Sprintf("Failed to create the subscription, err: %s", err.Error()))
	}
	return
}
