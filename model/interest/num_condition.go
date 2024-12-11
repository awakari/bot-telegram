package interest

type NumberCondition interface {
	KeyCondition
	GetOperation() NumOp
	GetValue() float64
}

type numCond struct {
	kc  KeyCondition
	op  NumOp
	val float64
}

func NewNumberCondition(kc KeyCondition, op NumOp, val float64) NumberCondition {
	return numCond{
		kc:  kc,
		op:  op,
		val: val,
	}
}

func (nc numCond) IsNot() bool {
	return nc.kc.IsNot()
}

func (nc numCond) GetKey() string {
	return nc.kc.GetKey()
}

func (nc numCond) GetOperation() NumOp {
	return nc.op
}

func (nc numCond) GetValue() float64 {
	return nc.val
}
