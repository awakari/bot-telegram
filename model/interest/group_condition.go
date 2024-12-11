package interest

type (
	GroupCondition interface {
		Condition
		GetLogic() (logic GroupLogic)
		GetGroup() (group []Condition)
	}

	groupCondition struct {
		Condition Condition
		Logic     GroupLogic
		Group     []Condition
	}
)

func NewGroupCondition(c Condition, logic GroupLogic, group []Condition) GroupCondition {
	return groupCondition{
		Condition: c,
		Logic:     logic,
		Group:     group,
	}
}

func (gc groupCondition) IsNot() bool {
	return gc.Condition.IsNot()
}

func (gc groupCondition) GetLogic() (logic GroupLogic) {
	return gc.Logic
}

func (gc groupCondition) GetGroup() (group []Condition) {
	return gc.Group
}
