package interest

type NumOp int

const (
	NumOpUndefined NumOp = iota
	NumOpGt
	NumOpGte
	NumOpEq
	NumOpLte
	NumOpLt
)

func (op NumOp) String() string {
	return [...]string{
		"Undefined",
		"Gt",
		"Gte",
		"Eq",
		"Lte",
		"Lt",
	}[op]
}
