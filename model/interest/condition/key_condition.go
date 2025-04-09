package condition

type (
	KeyCondition interface {
		Condition
		GetKey() string
	}

	keyCondition struct {
		Condition Condition
		Key       string
		Id        string
	}
)

func NewKeyCondition(c Condition, k string) KeyCondition {
	return keyCondition{
		Condition: c,
		Key:       k,
	}
}

func (kc keyCondition) IsNot() bool {
	return kc.Condition.IsNot()
}

func (kc keyCondition) GetKey() string {
	return kc.Key
}

func (kc keyCondition) GetId() string {
	return kc.Id
}
