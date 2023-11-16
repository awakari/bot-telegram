package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/limits"
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/telebot.v3"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const limitGroupOrCondChildrenCount = 4
const limitTextCondTermsLength = 256
const expiresDefaultDuration = time.Hour * 24 * usage.ExpiresDefaultDays // ~ month

const ReqSubCreate = "sub_create"
const msgSubCreate = "Creating a basic subscription with a single text matching condition. " +
	"Reply a name followed by keywords to the next message. Example:\n" +
	"<pre>Wishlist1 tesla iphone</pre>"
const msgSubCreated = `Subscription created, next: 
1. Create a target group chat to receive the matching messages. 
2. Invite @AwakariBot to the group.
3. Select the subscription in the group from the list.`

var errCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text subscription")
var errInvalidCondition = errors.New("invalid subscription condition")
var errLimitReached = errors.New("limit reached")
var whiteSpaceRegex = regexp.MustCompile(`\p{Zs}+`)

func CreateBasicRequest(tgCtx telebot.Context) (err error) {
	_ = tgCtx.Send(msgSubCreate, telebot.ModeHTML)
	m := &telebot.ReplyMarkup{
		ForceReply:  true,
		Placeholder: "name keyword1 keyword2 ...",
	}
	err = tgCtx.Send(ReqSubCreate, m)
	return
}

func CreateBasicReplyHandlerFunc(clientAwk api.Client, groupId string, kbd *telebot.ReplyMarkup) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		if len(args) < 2 {
			err = errCreateSubNotEnoughArgs
		}
		args = strings.SplitN(whiteSpaceRegex.ReplaceAllString(args[1], " "), " ", 2)
		if len(args) < 2 {
			err = errCreateSubNotEnoughArgs
		}
		var sd subscription.Data
		if err == nil {
			name := args[0]
			keywords := args[1]
			sd.Condition = condition.NewBuilder().
				AnyOfWords(keywords).
				BuildTextCondition()
			sd.Description = name
			sd.Enabled = true
			err = validateSubscriptionData(sd)
		}
		if err == nil {
			_, err = create(tgCtx, clientAwk, groupId, sd)
		}
		if err == nil {
			err = tgCtx.Send(msgSubCreated, kbd, telebot.ModeHTML)
		} else {
			err = fmt.Errorf("failed to create the subscription:\n%w", err)
		}
		return
	}
}

func CreateCustomHandlerFunc(clientAwk api.Client, groupId string) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		data := args[0]
		// TODO: maybe fix this double decoding/encoding of the payload with copy paste decode code
		var req subscriptions.CreateRequest
		err = protojson.Unmarshal([]byte(data), &req)
		var cond condition.Condition
		if err == nil {
			cond, err = decodeCondition(req.Cond)
		}
		//
		var sd subscription.Data
		if err == nil {
			sd.Condition = cond
			sd.Description = req.Description
			sd.Enabled = req.Enabled
			err = validateSubscriptionData(sd)
		}
		if err == nil {
			_, err = create(tgCtx, clientAwk, groupId, sd)
		}
		if err == nil {
			err = tgCtx.Send(msgSubCreated, telebot.ModeHTML)
		} else {
			err = fmt.Errorf("failed to create the subscription:\n%w", err)
		}
		return
	}
}

func create(tgCtx telebot.Context, clientAwk api.Client, groupId string, sd subscription.Data) (id string, err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	//
	// TODO: use the below code only when payments are connected
	//var existingIds []string
	//existingIds, err = clientAwk.SearchSubscriptions(groupIdCtx, userId, 1, "")
	//if err == nil {
	//	switch len(existingIds) {
	//	case 0: // leave expires = 0 (means "never") when user has no subscriptions
	//	default:
	//		sd.Expires = time.Now().Add(expiresDefaultDuration) // expire in a fixed period
	//	}
	//}
	//
	if err == nil {
		id, err = clientAwk.CreateSubscription(groupIdCtx, userId, sd)
		switch {
		case errors.Is(err, limits.ErrReached):
			err = fmt.Errorf(
				"%w, consider to donate and increase limit using the button \"%s\" in the main keyboard",
				errLimitReached,
				service.LabelSubs,
			)
		}
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

func validateSubscriptionData(sd subscription.Data) (err error) {
	if sd.Description == "" {
		err = errors.New("invalid subscription:\nempty description")
	}
	if err == nil {
		err = validateCondition(sd.Condition)
	}
	return err
}

func validateCondition(cond condition.Condition) (err error) {
	switch tc := cond.(type) {
	case condition.GroupCondition:
		children := tc.GetGroup()
		countChildren := len(children)
		if tc.GetLogic() == condition.GroupLogicOr && countChildren > limitGroupOrCondChildrenCount {
			err = fmt.Errorf(
				"%w:\nchildren condition count for the group condition with \"Or\" logic is %d, limit is %d,\nconsider to use an additional subscription instead",
				errInvalidCondition,
				countChildren,
				limitGroupOrCondChildrenCount,
			)
		} else {
			for _, child := range children {
				err = validateCondition(child)
				if err != nil {
					break
				}
			}
		}
	case condition.TextCondition:
		lenTerms := len(tc.GetTerm())
		if lenTerms > limitTextCondTermsLength {
			err = fmt.Errorf(
				"%w:\ntext condition terms length is %d, limit is %d,\nconsider to use an additional subscription",
				errInvalidCondition,
				lenTerms,
				limitGroupOrCondChildrenCount,
			)
		}
	}
	return
}
