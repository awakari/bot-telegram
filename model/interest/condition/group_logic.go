package condition

type GroupLogic int

const (
	GroupLogicAnd = iota
	GroupLogicOr
	GroupLogicXor
)

func (gl GroupLogic) String() string {
	return [...]string{
		"And",
		"Or",
		"Xor",
	}[gl]
}
