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

func TestBinOpObjectInDispatch(t *testing.T) {
	// `a in b` dispatches on the right operand (the container) via BinOpIn.
	ret, err, handled := binOpObject(nil, TBinaryOperatorIn, Int(2), Array{Int(1), Int(2), Int(3)})
	require.True(t, handled)
	require.NoError(t, err)
	require.Equal(t, True, ret)

	ret, _, handled = binOpObject(nil, TBinaryOperatorIn, Int(9), Array{Int(1), Int(2)})
	require.True(t, handled)
	require.Equal(t, False, ret)

	// a non-container right operand is not handled.
	_, _, handled = binOpObject(nil, TBinaryOperatorIn, Int(1), Int(2))
	require.False(t, handled)
}
