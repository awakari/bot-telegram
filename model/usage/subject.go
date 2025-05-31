package usage

type Subject int

const (
	SubjectUndefined Subject = iota
	SubjectInterests
	SubjectPublishHourly
	SubjectPublishDaily
	SubjectInterestsPublic
	SubjectSubscriptions
)

func (s Subject) String() string {
	return [...]string{
		"SubjectUndefined",
		"SubjectInterests",
		"SubjectPublishHourly",
		"SubjectPublishDaily",
		"SubjectInterestsPublic",
		"SubjectSubscriptions",
	}[s]
}

func (s Subject) Description() string {
	return [...]string{
		"Undefined",
		"Interests (own)",
		"Publishing (hourly)",
		"Publishing (daily)",
		"Interests Public (flag)",
		"Subscriptions",
	}[s]
}
