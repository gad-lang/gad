// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import "bytes"

// The `===` (Same) operator is strict: unlike `==`, it does not coerce between
// numeric kinds, so the operands must be the same concrete type and equal value
// (`1 === 1u` is false, `1 == 1u` is true). These implement
// ObjectWithSameBinOperator for the primitive value types; types without a
// BinOpSame fall back to reflect (primitives) or address identity.

func (o Int) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Int)
	return Bool(ok && o == r), nil
}

func (o Uint) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Uint)
	return Bool(ok && o == r), nil
}

func (o Float) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Float)
	return Bool(ok && o == r), nil
}

func (o Char) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Char)
	return Bool(ok && o == r), nil
}

func (o Bool) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Bool)
	return Bool(ok && o == r), nil
}

func (o Flag) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Flag)
	return Bool(ok && o == r), nil
}

func (o Str) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Str)
	return Bool(ok && o == r), nil
}

func (o RawStr) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(RawStr)
	return Bool(ok && o == r), nil
}

func (o Bytes) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Bytes)
	return Bool(ok && bytes.Equal(o, r)), nil
}

func (o Decimal) BinOpSame(_ *VM, right Object) (Object, error) {
	r, ok := right.(Decimal)
	return Bool(ok && o.ToGo().Equal(r.ToGo())), nil
}

func (o *NilType) BinOpSame(_ *VM, right Object) (Object, error) {
	_, ok := right.(*NilType)
	return Bool(ok), nil
}
