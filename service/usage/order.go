package usage

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/model/usage"
)

type Order struct {
	Limit OrderLimit `json:"limit"`
	Price OrderPrice `json:"price"`
}

type OrderLimit struct {
	TimeDays uint32        `json:"timeDays"`
	Count    uint32        `json:"count"`
	Subject  usage.Subject `json:"subject"`
}

type OrderPrice struct {
	Unit  string  `json:"unit"`
	Total float64 `json:"total"`
}

const orderLimitTimeDaysMin = 7
const orderLimitTimeDaysMax = 3652
const orderLimitCountMin = 2

var errInvalidOrder = errors.New("invalid order")

func (o Order) validate() (err error) {
	if err == nil && (o.Limit.TimeDays < orderLimitTimeDaysMin || o.Limit.TimeDays > orderLimitTimeDaysMax) {
		err = fmt.Errorf(
			"%w: limit duration is %d days, should be in the range [%d; %d]",
			errInvalidOrder,
			o.Limit.TimeDays,
			orderLimitTimeDaysMin,
			orderLimitTimeDaysMax,
		)
	}
	if err == nil && o.Limit.Count < orderLimitCountMin {
		err = fmt.Errorf(
			"%w: count is %d, should be greater or equal to %d",
			errInvalidOrder,
			o.Limit.Count,
			orderLimitCountMin,
		)
	}
	if err == nil {
		switch o.Limit.Subject {
		case usage.SubjectPublishEvents: // ok
		case usage.SubjectSubscriptions: // ok
		default:
			err = fmt.Errorf("%w: unknown subject %s", errInvalidOrder, o.Limit.Subject)
		}
	}
	if err == nil && o.Price.Total <= 0 {
		err = fmt.Errorf("%w: non-positive total price %f", errInvalidOrder, o.Price.Total)
	}
	return
}
