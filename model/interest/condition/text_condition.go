package condition

type (
	TextCondition interface {
		KeyCondition
		GetTerm() string
		IsExact() bool
	}

	textCondition struct {
		KeyCondition KeyCondition
		Term         string
		Exact        bool
	}
)

func NewTextCondition(kc KeyCondition, pattern string, exact bool) TextCondition {
	return textCondition{
		KeyCondition: kc,
		Term:         pattern,
		Exact:        exact,
	}
}

func (tc textCondition) IsNot() bool {
	return tc.KeyCondition.IsNot()
}

func (tc textCondition) GetKey() string {
	return tc.KeyCondition.GetKey()
}

func (tc textCondition) GetTerm() string {
	return tc.Term
}

func (tc textCondition) IsExact() bool {
	return tc.Exact
}
