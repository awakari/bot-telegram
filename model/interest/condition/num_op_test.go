package condition

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOp_String(t *testing.T) {
	assert.Equal(t, "Undefined", NumOpUndefined.String())
	assert.Equal(t, "Gte", NumOpGte.String())
	assert.Equal(t, "Gt", NumOpGt.String())
	assert.Equal(t, "Lt", NumOpLt.String())
	assert.Equal(t, "Lte", NumOpLte.String())
	assert.Equal(t, "Eq", NumOpEq.String())
}

func TestOp_Int(t *testing.T) {
	assert.Equal(t, 0, int(NumOpUndefined))
	assert.Equal(t, 1, int(NumOpGt))
	assert.Equal(t, 2, int(NumOpGte))
	assert.Equal(t, 3, int(NumOpEq))
	assert.Equal(t, 4, int(NumOpLte))
	assert.Equal(t, 5, int(NumOpLt))
}
