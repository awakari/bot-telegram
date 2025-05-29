package subject

import (
	"fmt"
	"github.com/awakari/bot-telegram/model/usage"
)

func Encode(src usage.Subject) (dst Subject, err error) {
	switch src {
	case usage.SubjectInterests:
		dst = Subject_Interests
	case usage.SubjectPublishHourly:
		dst = Subject_PublishHourly
	case usage.SubjectPublishDaily:
		dst = Subject_PublishDaily
	case usage.SubjectInterestsPublic:
		dst = Subject_InterestsPublic
	case usage.SubjectSubscriptions:
		dst = Subject_Subscriptions
	default:
		err = fmt.Errorf("invalid subject: %s", src)
	}
	return
}
