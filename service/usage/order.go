package usage

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/model/usage"
	"time"
)

type OrderExtend struct {
	Expires time.Time     `json:"expires"`
	Count   uint32        `json:"count"`
	Subject usage.Subject `json:"subject"`
}

const orderLimitTimeDaysMin = 1
const orderLimitTimeDaysMax = 3652
const orderLimitCountMinMsgs = 2
const orderLimitCountMinSubs = 10
const orderLimitCountMaxSubs = 8333
const orderLimitCountMaxMsgs = 8333 // = $9999.6

var errInvalidOrder = errors.New("invalid order")

func (oe OrderExtend) validate() (err error) {
	if err == nil && oe.Expires.Before(time.Now()) {
		err = fmt.Errorf(
			"%w: new expiration date %s is in past",
			errInvalidOrder,
			oe.Expires.Format(time.RFC3339),
		)
	}
	if err == nil {
		switch oe.Subject {
		case usage.SubjectPublishEvents:
			if oe.Count < orderLimitCountMinMsgs || oe.Count > orderLimitCountMaxMsgs {
				err = fmt.Errorf(
					"%w: count is %d, should be in the range [%d; %d]",
					errInvalidOrder,
					oe.Count,
					orderLimitCountMinMsgs,
					orderLimitCountMaxMsgs,
				)
			}
		case usage.SubjectSubscriptions:
			if oe.Count < orderLimitCountMaxSubs || oe.Count > orderLimitCountMaxSubs {
				err = fmt.Errorf(
					"%w: count is %d, should be in the range [%d; %d]",
					errInvalidOrder,
					oe.Count,
					orderLimitCountMinSubs,
					orderLimitCountMaxSubs,
				)
			}
		default:
			err = fmt.Errorf("%w: unknown subject %s", errInvalidOrder, oe.Subject)
		}
	}
	return
}
