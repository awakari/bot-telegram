package condition

type Builder interface {

	// Not makes a resulting Condition negative.
	// May be set for any resulting Condition type.
	Not() Builder

	// All defines the nested Condition set for a resulting GroupCondition with GroupLogicAnd.
	All(children []Condition) Builder

	// Any defines the nested Condition set for a resulting GroupCondition with GroupLogicOr.
	Any(children []Condition) Builder

	// Xor defines the nested Condition set for a resulting GroupCondition with GroupLogicXor.
	Xor(children []Condition) Builder

	// BuildGroupCondition builds a GroupCondition.
	BuildGroupCondition() (c Condition)

	// AttributeKey defines the incoming messages attribute key to match for a resulting KeyCondition.
	// Default is empty string key. For a TextCondition empty key causes the matching against all attribute keys.
	AttributeKey(k string) Builder

	// AnyOfWords defines the text search words (separated by a whitespace) for a resulting TextCondition.
	AnyOfWords(text string) Builder

	// TextEquals enables the exact text matching criteria for a resulting TextCondition.
	TextEquals(text string) Builder

	// BuildTextCondition builds a TextCondition.
	BuildTextCondition() (c Condition)

	// GreaterThan enables the number value matching criteria "x > val" for a NumberCondition.
	GreaterThan(val float64) Builder

	// GreaterThanOrEqual enables the number value matching criteria "x >= val" for a NumberCondition.
	GreaterThanOrEqual(val float64) Builder

	// Equal enables the number value matching criteria "x == val" for a NumberCondition.
	Equal(val float64) Builder

	// LessThanOrEqual enables the number value matching criteria "x <= val" for a NumberCondition.
	LessThanOrEqual(val float64) Builder

	// LessThan enables the number value matching criteria "x < val" for a NumberCondition.
	LessThan(val float64) Builder

	// BuildNumberCondition builds a NumberCondition
	BuildNumberCondition() (c Condition)
}

type builder struct {
	not   bool
	gl    GroupLogic
	gc    []Condition
	key   string
	term  string
	exact bool
	op    NumOp
	val   float64
}

func NewBuilder() Builder {
	return &builder{}
}

func (b *builder) Not() Builder {
	b.not = true
	return b
}

func (b *builder) All(children []Condition) Builder {
	b.gl = GroupLogicAnd
	b.gc = children
	return b
}

func (b *builder) Any(children []Condition) Builder {
	b.gl = GroupLogicOr
	b.gc = children
	return b
}

func (b *builder) Xor(children []Condition) Builder {
	b.gl = GroupLogicXor
	b.gc = children
	return b
}

func (b *builder) AttributeKey(k string) Builder {
	b.key = k
	return b
}

func (b *builder) AnyOfWords(text string) Builder {
	b.term = text
	b.exact = false
	return b
}

func (b *builder) TextEquals(text string) Builder {
	b.term = text
	b.exact = true
	return b
}

func (b *builder) BuildGroupCondition() (c Condition) {
	c = condition{
		Not: b.not,
	}
	c = groupCondition{
		Condition: c,
		Logic:     b.gl,
		Group:     b.gc,
	}
	return
}

func (b *builder) BuildTextCondition() (c Condition) {
	c = condition{
		b.not,
	}
	c = textCondition{
		KeyCondition: keyCondition{
			Condition: c,
			Key:       b.key,
		},
		Term:  b.term,
		Exact: b.exact,
	}
	return
}

func (b *builder) GreaterThan(val float64) Builder {
	b.op = NumOpGt
	b.val = val
	return b
}

func (b *builder) GreaterThanOrEqual(val float64) Builder {
	b.op = NumOpGte
	b.val = val
	return b
}

func (b *builder) Equal(val float64) Builder {
	b.op = NumOpEq
	b.val = val
	return b
}

func (b *builder) LessThanOrEqual(val float64) Builder {
	b.op = NumOpLte
	b.val = val
	return b
}

func (b *builder) LessThan(val float64) Builder {
	b.op = NumOpLt
	b.val = val
	return b
}

func (b *builder) BuildNumberCondition() (c Condition) {
	c = condition{
		b.not,
	}
	c = numCond{
		kc: keyCondition{
			Condition: c,
			Key:       b.key,
		},
		op:  b.op,
		val: b.val,
	}
	return
}
