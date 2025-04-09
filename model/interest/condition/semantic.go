package condition

type SemanticCondition interface {
	Condition
	Query() string
}

type semCond struct {
	cond  Condition
	id    string
	query string
}

func NewSemanticCondition(cond Condition, id string, query string) SemanticCondition {
	return &semCond{
		cond:  cond,
		id:    id,
		query: query,
	}
}

func (sc *semCond) IsNot() bool {
	return sc.cond.IsNot()
}

func (sc *semCond) GetId() string {
	return sc.id
}

func (sc *semCond) Query() string {
	return sc.query
}
