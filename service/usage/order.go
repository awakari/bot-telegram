package usage

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/model/usage"
	"time"
)

type OrderLimit struct {
	Expires time.Time     `json:"expires"`
	Count   uint32        `json:"count"`
	Subject usage.Subject `json:"subject"`
}

const orderLimitCountMinMsgs = 11
const orderLimitCountMinSubs = 2
const orderLimitCountMaxSubs = 100000
const orderLimitCountMaxMsgs = 1000

var errInvalidOrder = errors.New("invalid order")

func (ol OrderLimit) validate() (err error) {
	if err == nil && ol.Expires.Before(time.Now()) {
		err = fmt.Errorf(
			"%w: new expiration date %s is in past",
			errInvalidOrder,
			ol.Expires.Format(time.RFC3339),
		)
	}
	if err == nil {
		switch ol.Subject {
		case usage.SubjectPublishEvents:
			if ol.Count < orderLimitCountMinMsgs || ol.Count > orderLimitCountMaxMsgs {
				err = fmt.Errorf(
					"%w: count is %d, should be in the range [%d; %d]",
					errInvalidOrder,
					ol.Count,
					orderLimitCountMinMsgs,
					orderLimitCountMaxMsgs,
				)
			}
		case usage.SubjectSubscriptions:
			if ol.Count < orderLimitCountMinSubs || ol.Count > orderLimitCountMaxSubs {
				err = fmt.Errorf(
					"%w: count is %d, should be in the range [%d; %d]",
					errInvalidOrder,
					ol.Count,
					orderLimitCountMinSubs,
					orderLimitCountMaxSubs,
				)
			}
		default:
			err = fmt.Errorf("%w: unknown subject %s", errInvalidOrder, ol.Subject)
		}
	}
	return
}
