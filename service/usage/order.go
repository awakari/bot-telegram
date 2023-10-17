package usage

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/model/usage"
)

type OrderLimit struct {
	TimeDays uint32        `json:"timeDays"`
	Count    uint32        `json:"count"`
	Subject  usage.Subject `json:"subject"`
}

const orderLimitTimeDaysMin = 7
const orderLimitTimeDaysMax = 3652
const orderLimitCountMin = 2

var errInvalidOrder = errors.New("invalid order")

func (o OrderLimit) validate() (err error) {
	if err == nil && (o.TimeDays < orderLimitTimeDaysMin || o.TimeDays > orderLimitTimeDaysMax) {
		err = fmt.Errorf(
			"%w: limit duration is %d days, should be in the range [%d; %d]",
			errInvalidOrder,
			o.TimeDays,
			orderLimitTimeDaysMin,
			orderLimitTimeDaysMax,
		)
	}
	if err == nil && o.Count < orderLimitCountMin {
		err = fmt.Errorf(
			"%w: count is %d, should be greater or equal to %d",
			errInvalidOrder,
			o.Count,
			orderLimitCountMin,
		)
	}
	if err == nil {
		switch o.Subject {
		case usage.SubjectPublishEvents: // ok
		case usage.SubjectSubscriptions: // ok
		default:
			err = fmt.Errorf("%w: unknown subject %s", errInvalidOrder, o.Subject)
		}
	}
	return
}
