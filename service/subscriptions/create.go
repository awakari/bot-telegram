package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/http/reader"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/api/grpc/limits"
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/awakari/client-sdk-go/model/subscription"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"regexp"
	"strings"
)

const limitGroupOrCondChildrenCount = 4
const minTextCondTermsLength = 3
const maxTextCondTermsLength = 256

const ReqSubCreate = "sub_create"
const msgSubCreate = "Following a simple text interest. " +
	"Reply a name followed by keywords to the next message. Example:\n" +
	"<pre>Wishlist1 tesla iphone</pre>"
const msgSubCreated = "If you want to read it in another chat, unlink it first using the <pre>/start</pre> command."

var errCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text interest")
var errInvalidCondition = errors.New("invalid interest condition")
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

func CreateBasicReplyHandlerFunc(
	clientAwk api.Client,
	groupId string,
	svcReader reader.Service,
	urlCallbackBase string,
) service.ArgHandlerFunc {
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
		var subId string
		if err == nil {
			subId, err = create(tgCtx, clientAwk, groupId, sd)
		}
		if err == nil {
			err = Start(tgCtx, clientAwk, svcReader, urlCallbackBase, subId, groupId)
		} else {
			err = fmt.Errorf("failed to register the interest:\n%w", err)
		}
		if err == nil {
			err = tgCtx.Send(msgSubCreated, telebot.ModeHTML)
		} else {
			err = fmt.Errorf("failed to follow the interest to this chat:\n%w", err)
		}
		return
	}
}

func create(tgCtx telebot.Context, clientAwk api.Client, groupId string, sd subscription.Data) (id string, err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, groupId)
	userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
	id, err = clientAwk.CreateSubscription(groupIdCtx, userId, sd)
	switch {
	case errors.Is(err, limits.ErrReached):
		err = fmt.Errorf("%w, consider to request to increase your limit", errLimitReached)
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
		err = errors.New("invalid interest:\nempty description")
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
				"%w:\nchildren condition count for the group condition with \"Or\" logic is %d, limit is %d,\nconsider to follow an additional interest instead",
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
		if lenTerms < minTextCondTermsLength || lenTerms > maxTextCondTermsLength {
			err = fmt.Errorf(
				"%w:\ntext condition terms length is %d, should be [%d, %d]",
				errInvalidCondition,
				lenTerms,
				minTextCondTermsLength,
				maxTextCondTermsLength,
			)
		}
	}
	return
}
