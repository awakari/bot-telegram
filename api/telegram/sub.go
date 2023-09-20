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

type SubscriptionHandlers struct {
	Client  api.Client
	GroupId string
}

const CmdPrefixSubCreateSimplePrefix = "/sub"
const argSep = " "
const subListLimit = 10 // TODO: implement the proper pagination later

var ErrCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text subscription")

var whiteSpaceRegex = regexp.MustCompile(`\p{Zs}+`)

func (h SubscriptionHandlers) CreateTextSubscription(ctx telebot.Context) (err error) {
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
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", h.GroupId)
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
		subId, err = h.Client.CreateSubscription(groupIdCtx, userId, subData)
	}
	if err == nil {
		err = ctx.Send(fmt.Sprintf("Created the new simple subscription, id:\n<pre>%s</pre>", subId))
	} else {
		err = fmt.Errorf("failed to create the subscription: %w", err)
	}
	return
}

func (h SubscriptionHandlers) ListMySubscriptions(ctx telebot.Context) (err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", h.GroupId)
	userId := strconv.FormatInt(ctx.Sender().ID, 10)
	var subIds []string
	subIds, err = h.Client.SearchSubscriptions(groupIdCtx, userId, subListLimit, "")
	m := &telebot.ReplyMarkup{}
	var rows []telebot.Row
	if err == nil {
		var sub subscription.Data
		for _, subId := range subIds {
			sub, err = h.Client.ReadSubscription(groupIdCtx, userId, subId)
			if err != nil {
				break
			}
			row := m.Row(telebot.Btn{
				Unique: subId,
				Text:   sub.Description,
				Data:   subId,
			})
			rows = append(rows, row)
		}
	}
	m.Inline(rows...)
	err = ctx.Send(msgStartGroup, m)
	return
}
