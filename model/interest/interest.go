package interest

import (
	"time"
)

type Data struct {

	// Condition represents the certain criteria to select the Subscription for the further routing.
	// It's immutable once the Subscription is created.
	Condition Condition

	// Description is a human-readable subscription description.
	Description string

	// Enabled defines whether subscription is active and may be used to deliver a message.
	Enabled bool

	// Expires defines a deadline when subscription becomes disabled regardless the Enabled value.
	Expires time.Time

	Created time.Time

	Updated time.Time

	Public bool

	Followers int64
}
