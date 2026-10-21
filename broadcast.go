package gad

import (
	"fmt"

	"github.com/gad-lang/gad/token"
)

// Broadcast is `left .op right` (Julia's broadcast): op applied item by item,
// an array of the results.
//
//   - two arrays (a typed one's items too) of the same length: `a[i] op b[i]`;
//   - an array and a value: the value with every item (`[1, 2] .+ 1`);
//   - two values: `left op right`.
//
// One level only, as Julia: an item that is an array takes op as a whole
// (`[[1, 2]] .< [[1, 3]]` is `[true]`); nested descends into nested arrays,
// broadcasting at every level (`[1, [2]] .+ 1` with nested is `[2, [3]]`).
func Broadcast(vm *VM, tok token.Token, left, right Object, nested bool) (Object, error) {
	return vm.broadcast(tok, left, right, nested)
}

func (vm *VM) broadcast(tok token.Token, left, right Object, nested bool) (Object, error) {
	l, lok := broadcastItems(left)
	r, rok := broadcastItems(right)
	if !lok && !rok {
		return vm.broadcastOp(tok, left, right)
	}
	if lok && rok && len(l) != len(r) {
		return nil, ErrArgument.NewError(fmt.Sprintf(
			".%s: arrays of different lengths (%d and %d)", tok.String(), len(l), len(r)))
	}
	n := len(l)
	if !lok {
		n = len(r)
	}
	out := make(Array, n)
	for i := range out {
		a, b := left, right
		if lok {
			a = l[i]
		}
		if rok {
			b = r[i]
		}
		var (
			v   Object
			err error
		)
		if nested {
			v, err = vm.broadcast(tok, a, b, true)
		} else {
			v, err = vm.broadcastOp(tok, a, b)
		}
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// broadcastItems are the items o broadcasts over: an array's, a typed
// array's.
func broadcastItems(o Object) (Array, bool) {
	switch t := o.(type) {
	case Array:
		return t, true
	case *TypedArray:
		return t.Items, true
	}
	return nil, false
}

// broadcastOp is `a tok b` of two items, as the binary operator is compiled:
// `==`/`!=` by Equal, `!==` the negation of `===`, the others through
// gad.binOp (the user overloads too).
func (vm *VM) broadcastOp(tok token.Token, a, b Object) (Object, error) {
	switch tok {
	case token.Equal:
		return Bool(a.Equal(b)), nil
	case token.NotEqual:
		return Bool(!a.Equal(b)), nil
	case token.NotSame:
		v, err := vm.callBinaryOp(token.Same, a, b)
		if err != nil {
			return nil, err
		}
		return Bool(v.IsFalsy()), nil
	}
	return vm.callBinaryOp(tok, a, b)
}
