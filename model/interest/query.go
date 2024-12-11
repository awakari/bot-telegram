package interest

type Query struct {
	Limit   uint32
	Sort    Sort
	Order   Order
	Pattern string
	Public  bool // include public non-own?
}

type QueryByCondition struct {
	CondId string
	Limit  uint32
}

type Sort int

const (
	SortId Sort = iota
	SortFollowers
)

func (s Sort) String() string {
	return [...]string{
		"Id",
		"Followers",
	}[s]
}

type Order int

const (
	OrderAsc Order = iota
	OrderDesc
)

func (o Order) String() string {
	return [...]string{
		"Asc",
		"Desc",
	}[o]
}
