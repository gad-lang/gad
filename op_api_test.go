package gad

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// addOnlyObject implements only the per-operator ObjectWithAddBinOperator API,
// exercising the generated binOpObject dispatcher (op_api.go).
type addOnlyObject struct {
	ObjectImpl
	n Int
}

func (a addOnlyObject) BinOpAdd(_ *VM, right Object) (Object, error) {
	if r, ok := right.(Int); ok {
		return a.n + r, nil
	}
	return nil, ErrInvalidOperator
}

func TestBinOpObjectDispatch(t *testing.T) {
	left := addOnlyObject{n: 5}

	// the implemented operator is dispatched to BinOpAdd.
	ret, err, handled := binOpObject(nil, TBinaryOperatorAdd, left, Int(3))
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, Int(8), ret)

	// an operator the type does not implement is not handled (caller falls back
	// to BinaryOperatorHandler).
	_, _, handled = binOpObject(nil, TBinaryOperatorSub, left, Int(3))
	require.False(t, handled)
}
