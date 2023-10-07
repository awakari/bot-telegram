package chats

type State int

const (
	StateUndefined State = iota
	StateActive
	StateInactive
)

func (s State) String() string {
	return [...]string{
		"StateUndefined",
		"StateActive",
		"StateInactive",
	}[s]
}
