package condition

type Condition interface {
	IsNot() bool
}

type condition struct {
	Not bool
}

func NewCondition(not bool) Condition {
	return condition{
		Not: not,
	}
}

func (c condition) IsNot() bool {
	return c.Not
}
