// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/internal/compat"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// boolInt converts a Bool to 0/1 for arithmetic with the numeric types.
func boolInt(v Bool) Int {
	if v {
		return 1
	}
	return 0
}

// Int represents signed integer values and implements Object interface.
type Int int64

func (o Int) Type() ObjectType {
	return TInt
}

func (o Int) ToString() string {
	return strconv.FormatInt(int64(o), 10)
}

// Equal implements Object interface.
func (o Int) Equal(right Object) bool {
	switch v := right.(type) {
	case Int:
		return o == v
	case Uint:
		return Uint(o) == v
	case Float:
		return Float(o) == v
	case Decimal:
		return DecimalFromInt(o).Equal(v)
	case Char:
		return o == Int(v)
	case Bool:
		if v {
			return o == 1
		}
		return o == 0
	}
	return false
}

// IsFalsy implements Object interface.
func (o Int) IsFalsy() bool { return o == 0 }

// Small integers are boxed into the Object interface once and shared, so common
// arithmetic (loop counters, indices, flags) does not heap-allocate a fresh box
// each time. Int is an immutable value type and Go compares interface values by
// (type, value), so sharing a box is transparent.
const (
	smallIntMin Int = -256
	smallIntMax Int = 1024
)

var smallInts [smallIntMax - smallIntMin + 1]Object

func init() {
	for i := range smallInts {
		smallInts[i] = smallIntMin + Int(i)
	}
}

// intObject returns v as an Object, reusing a shared box for small values.
func intObject(v Int) Object {
	if v >= smallIntMin && v <= smallIntMax {
		return smallInts[v-smallIntMin]
	}
	return v
}

func (o Int) BinOpAdd(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return intObject(o + v), nil
	case Uint:
		return Uint(o).BinOpAdd(vm, right)
	case Float:
		return Float(o).BinOpAdd(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpAdd(vm, right)
	case Char:
		return Char(o) + v, nil
	case Bool:
		return o.BinOpAdd(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpSub(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return intObject(o - v), nil
	case Uint:
		return Uint(o).BinOpSub(vm, right)
	case Float:
		return Float(o).BinOpSub(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpSub(vm, right)
	case Char:
		return Char(o) - v, nil
	case Bool:
		return o.BinOpSub(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpMul(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return intObject(o * v), nil
	case Uint:
		return Uint(o).BinOpMul(vm, right)
	case Float:
		return Float(o).BinOpMul(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpMul(vm, right)
	case Bool:
		return o.BinOpMul(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Mul.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpPow(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return Decimal(decimal.NewFromInt(int64(o)).Pow(decimal.NewFromInt(int64(v)))), nil
	case Float:
		return Float(o).BinOpPow(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpPow(vm, right)
	case Bool:
		return o.BinOpPow(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Pow.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpQuo(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return intObject(o / v), nil
	case Uint:
		return Uint(o).BinOpQuo(vm, right)
	case Float:
		return Float(o).BinOpQuo(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpQuo(vm, right)
	case Bool:
		return o.BinOpQuo(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Quo.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpRem(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return intObject(o % v), nil
	case Uint:
		return Uint(o).BinOpRem(vm, right)
	case Bool:
		return o.BinOpRem(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Rem.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpAnd(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o & v, nil
	case Uint:
		return Uint(o).BinOpAnd(vm, right)
	case Bool:
		return o.BinOpAnd(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.And.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpOr(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o | v, nil
	case Uint:
		return Uint(o).BinOpOr(vm, right)
	case Bool:
		return o.BinOpOr(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Or.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpXor(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o ^ v, nil
	case Uint:
		return Uint(o).BinOpXor(vm, right)
	case Bool:
		return o.BinOpXor(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Xor.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpAndNot(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o &^ v, nil
	case Uint:
		return Uint(o).BinOpAndNot(vm, right)
	case Bool:
		return o.BinOpAndNot(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.AndNot.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpShl(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o << v, nil
	case Uint:
		return Uint(o).BinOpShl(vm, right)
	case Bool:
		return o.BinOpShl(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Shl.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpShr(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o >> v, nil
	case Uint:
		return Uint(o).BinOpShr(vm, right)
	case Bool:
		return o.BinOpShr(vm, boolInt(v))
	}
	return nil, NewOperandTypeError(token.Shr.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpLess(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return Bool(o < v), nil
	case Uint:
		return Uint(o).BinOpLess(vm, right)
	case Float:
		return Float(o).BinOpLess(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpLess(vm, right)
	case Char:
		return Bool(o < Int(v)), nil
	case Bool:
		return o.BinOpLess(vm, boolInt(v))
	case *NilType:
		return False, nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpLessEq(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return Bool(o <= v), nil
	case Uint:
		return Uint(o).BinOpLessEq(vm, right)
	case Float:
		return Float(o).BinOpLessEq(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpLessEq(vm, right)
	case Char:
		return Bool(o <= Int(v)), nil
	case Bool:
		return o.BinOpLessEq(vm, boolInt(v))
	case *NilType:
		return False, nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpGreater(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return Bool(o > v), nil
	case Uint:
		return Uint(o).BinOpGreater(vm, right)
	case Float:
		return Float(o).BinOpGreater(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpGreater(vm, right)
	case Char:
		return Bool(o > Int(v)), nil
	case Bool:
		return o.BinOpGreater(vm, boolInt(v))
	case *NilType:
		return True, nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o Int) BinOpGreaterEq(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return Bool(o >= v), nil
	case Uint:
		return Uint(o).BinOpGreaterEq(vm, right)
	case Float:
		return Float(o).BinOpGreaterEq(vm, right)
	case Decimal:
		return DecimalFromInt(o).BinOpGreaterEq(vm, right)
	case Char:
		return Bool(o >= Int(v)), nil
	case Bool:
		return o.BinOpGreaterEq(vm, boolInt(v))
	case *NilType:
		return True, nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// Format implements fmt.Formatter interface.
func (o Int) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, int64(o))
}

// Uint represents unsigned integer values and implements Object interface.
type Uint uint64

func (o Uint) Type() ObjectType {
	return TUint
}

func (o Uint) ToString() string {
	return strconv.FormatUint(uint64(o), 10)
}

// Equal implements Object interface.
func (o Uint) Equal(right Object) bool {
	switch v := right.(type) {
	case Uint:
		return o == v
	case Int:
		return o == Uint(v)
	case Float:
		return Float(o) == v
	case Decimal:
		return DecimalFromUint(o).Equal(v)
	case Char:
		return o == Uint(v)
	case Bool:
		if v {
			return o == 1
		}
		return o == 0
	}
	return false
}

// IsFalsy implements Object interface.
func (o Uint) IsFalsy() bool { return o == 0 }

func (o Uint) BinOpAdd(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o + v, nil
	case Int:
		return o.BinOpAdd(vm, Uint(v))
	case Float:
		return Float(o).BinOpAdd(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpAdd(vm, right)
	case Char:
		return Char(o) + v, nil
	case Bool:
		return o.BinOpAdd(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpSub(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o - v, nil
	case Int:
		return o.BinOpSub(vm, Uint(v))
	case Float:
		return Float(o).BinOpSub(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpSub(vm, right)
	case Char:
		return Char(o) - v, nil
	case Bool:
		return o.BinOpSub(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpMul(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o * v, nil
	case Int:
		return o.BinOpMul(vm, Uint(v))
	case Float:
		return Float(o).BinOpMul(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpMul(vm, right)
	case Bool:
		return o.BinOpMul(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Mul.String(), o.Type().Name(), right.Type().Name())
}

// Uint has no native power; `uint ** float|decimal` promotes to that type.
func (o Uint) BinOpPow(vm *VM, right Object) (Object, error) {
	switch right.(type) {
	case Float:
		return Float(o).BinOpPow(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpPow(vm, right)
	}
	return nil, NewOperandTypeError(token.Pow.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpQuo(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return o / v, nil
	case Int:
		return o.BinOpQuo(vm, Uint(v))
	case Float:
		return Float(o).BinOpQuo(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpQuo(vm, right)
	case Bool:
		return o.BinOpQuo(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Quo.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpRem(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o % v, nil
	case Int:
		return o.BinOpRem(vm, Uint(v))
	case Bool:
		return o.BinOpRem(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Rem.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpAnd(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o & v, nil
	case Int:
		return o.BinOpAnd(vm, Uint(v))
	case Bool:
		return o.BinOpAnd(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.And.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpOr(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o | v, nil
	case Int:
		return o.BinOpOr(vm, Uint(v))
	case Bool:
		return o.BinOpOr(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Or.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpXor(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o ^ v, nil
	case Int:
		return o.BinOpXor(vm, Uint(v))
	case Bool:
		return o.BinOpXor(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Xor.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpAndNot(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o &^ v, nil
	case Int:
		return o.BinOpAndNot(vm, Uint(v))
	case Bool:
		return o.BinOpAndNot(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.AndNot.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpShl(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o << v, nil
	case Int:
		return o.BinOpShl(vm, Uint(v))
	case Bool:
		return o.BinOpShl(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Shl.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpShr(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return o >> v, nil
	case Int:
		return o.BinOpShr(vm, Uint(v))
	case Bool:
		return o.BinOpShr(vm, Uint(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Shr.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpLess(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return Bool(o < v), nil
	case Int:
		return o.BinOpLess(vm, Uint(v))
	case Float:
		return Float(o).BinOpLess(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpLess(vm, right)
	case Char:
		return Bool(o < Uint(v)), nil
	case Bool:
		return o.BinOpLess(vm, Uint(boolInt(v)))
	case *NilType:
		return False, nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpLessEq(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return Bool(o <= v), nil
	case Int:
		return o.BinOpLessEq(vm, Uint(v))
	case Float:
		return Float(o).BinOpLessEq(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpLessEq(vm, right)
	case Char:
		return Bool(o <= Uint(v)), nil
	case Bool:
		return o.BinOpLessEq(vm, Uint(boolInt(v)))
	case *NilType:
		return False, nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpGreater(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return Bool(o > v), nil
	case Int:
		return o.BinOpGreater(vm, Uint(v))
	case Float:
		return Float(o).BinOpGreater(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpGreater(vm, right)
	case Char:
		return Bool(o > Uint(v)), nil
	case Bool:
		return o.BinOpGreater(vm, Uint(boolInt(v)))
	case *NilType:
		return True, nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o Uint) BinOpGreaterEq(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		return Bool(o >= v), nil
	case Int:
		return o.BinOpGreaterEq(vm, Uint(v))
	case Float:
		return Float(o).BinOpGreaterEq(vm, right)
	case Decimal:
		return DecimalFromUint(o).BinOpGreaterEq(vm, right)
	case Char:
		return Bool(o >= Uint(v)), nil
	case Bool:
		return o.BinOpGreaterEq(vm, Uint(boolInt(v)))
	case *NilType:
		return True, nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// Format implements fmt.Formatter interface.
func (o Uint) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, uint64(o))
}

// Float represents float values and implements Object interface.
type Float float64

var NaN = Float(math.NaN())

func (o Float) Type() ObjectType {
	return TFloat
}

func (o Float) ToString() string {
	return strconv.FormatFloat(float64(o), 'g', -1, 64)
}

// Equal implements Object interface.
func (o Float) Equal(right Object) bool {
	switch v := right.(type) {
	case Float:
		return o == v
	case Int:
		return o == Float(v)
	case Uint:
		return o == Float(v)
	case Decimal:
		return DecimalFromFloat(o).Equal(v)
	case Bool:
		if v {
			return o == 1
		}
		return o == 0
	}
	return false
}

// IsFalsy implements Object interface.
func (o Float) IsFalsy() bool {
	// IEEE 754 says that only NaNs satisfy f != f.
	// See math.IsNan
	f := float64(o)
	return f != f
}

// floatRHS normalizes a Float's right operand: ok=true with a Float for
// Float/Int/Uint/Bool operands; isDecimal=true means promote the whole
// operation to Decimal; otherwise neither (nil or an unsupported type).
func floatRHS(right Object) (f Float, isDecimal, ok bool) {
	switch v := right.(type) {
	case Float:
		return v, false, true
	case Int:
		return Float(v), false, true
	case Uint:
		return Float(v), false, true
	case Bool:
		return Float(boolInt(v)), false, true
	case Decimal:
		return 0, true, false
	}
	return 0, false, false
}

func (o Float) BinOpAdd(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return o + f, nil
	} else if dec {
		return DecimalFromFloat(o).BinOpAdd(vm, right)
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpSub(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return o - f, nil
	} else if dec {
		return DecimalFromFloat(o).BinOpSub(vm, right)
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpMul(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return o * f, nil
	} else if dec {
		return DecimalFromFloat(o).BinOpMul(vm, right)
	}
	return nil, NewOperandTypeError(token.Mul.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpPow(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return Float(math.Pow(float64(o), float64(f))), nil
	} else if dec {
		return DecimalFromFloat(o).BinOpPow(vm, right)
	}
	return nil, NewOperandTypeError(token.Pow.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpQuo(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		if f == 0 {
			return nil, ErrZeroDivision
		}
		return o / f, nil
	} else if dec {
		return DecimalFromFloat(o).BinOpQuo(vm, right)
	}
	return nil, NewOperandTypeError(token.Quo.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpLess(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return Bool(o < f), nil
	} else if dec {
		return DecimalFromFloat(o).BinOpLess(vm, right)
	} else if right == Nil {
		return False, nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpLessEq(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return Bool(o <= f), nil
	} else if dec {
		return DecimalFromFloat(o).BinOpLessEq(vm, right)
	} else if right == Nil {
		return False, nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpGreater(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return Bool(o > f), nil
	} else if dec {
		return DecimalFromFloat(o).BinOpGreater(vm, right)
	} else if right == Nil {
		return True, nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o Float) BinOpGreaterEq(vm *VM, right Object) (Object, error) {
	if f, dec, ok := floatRHS(right); ok {
		return Bool(o >= f), nil
	} else if dec {
		return DecimalFromFloat(o).BinOpGreaterEq(vm, right)
	} else if right == Nil {
		return True, nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// Format implements fmt.Formatter interface.
func (o Float) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, float64(o))
}

// Decimal represents a fixed-point decimal. It is immutable.
// number = value * 10 ^ exp
type Decimal decimal.Decimal

func (o *Decimal) GobDecode(bytes []byte) (err error) {
	var dec decimal.Decimal
	if err = dec.UnmarshalBinary(bytes); err == nil {
		*o = Decimal(dec)
	}
	return
}

func (o Decimal) GobEncode() ([]byte, error) {
	return o.ToGo().MarshalBinary()
}

func (o Decimal) ToGo() decimal.Decimal {
	return decimal.Decimal(o)
}

func (o Decimal) ToInterface() any {
	return decimal.Decimal(o)
}

func (o Decimal) Type() ObjectType {
	return TDecimal
}

func (o Decimal) ToString() string {
	return o.ToGo().String()
}

// Equal implements Object interface.
func (o Decimal) Equal(right Object) bool {
	switch v := right.(type) {
	case Decimal:
		return o.ToGo().Equal(v.ToGo())
	case Int:
		return o.ToGo().Equal(decimal.Decimal(DecimalFromInt(v)))
	case Uint:
		return o.ToGo().Equal(decimal.Decimal(DecimalFromUint(v)))
	case Float:
		return o.ToGo().Equal(decimal.Decimal(DecimalFromFloat(v)))
	case Bool:
		return o.ToGo().IsZero() != bool(v)
	}
	return false
}

// IsFalsy implements Object interface.
func (o Decimal) IsFalsy() bool {
	// IEEE 754 says that only NaNs satisfy f != f.
	// See math.IsNan
	return o.ToGo().IsZero()
}

// decimalRHS converts a Decimal's right operand to a Decimal. ok=false (with
// nil err) means an unsupported type (e.g. nil); a non-nil err is a conversion
// failure (bad Str/Bytes).
func decimalRHS(right Object) (d Decimal, ok bool, err error) {
	switch v := right.(type) {
	case Decimal:
		return v, true, nil
	case Int:
		return DecimalFromInt(v), true, nil
	case Uint:
		return DecimalFromUint(v), true, nil
	case Float:
		return DecimalFromFloat(v), true, nil
	case Char:
		return DecimalFromUint(Uint(v)), true, nil
	case Bool:
		return DecimalFromUint(Uint(boolInt(v))), true, nil
	case Str:
		dd, e := DecimalFromString(v)
		if e != nil {
			return Decimal{}, false, ErrType.NewError(e.Error())
		}
		return dd, true, nil
	case Bytes:
		var dd decimal.Decimal
		if e := dd.UnmarshalBinary(v); e != nil {
			return Decimal{}, false, e
		}
		return Decimal(dd), true, nil
	}
	return Decimal{}, false, nil
}

func (o Decimal) BinOpAdd(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil || !ok {
		return decimalUnsupported(token.Add, o, right, err)
	}
	return Decimal(o.ToGo().Add(d.ToGo())), nil
}

func (o Decimal) BinOpSub(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil || !ok {
		return decimalUnsupported(token.Sub, o, right, err)
	}
	return Decimal(o.ToGo().Sub(d.ToGo())), nil
}

func (o Decimal) BinOpMul(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil || !ok {
		return decimalUnsupported(token.Mul, o, right, err)
	}
	return Decimal(o.ToGo().Mul(d.ToGo())), nil
}

func (o Decimal) BinOpPow(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil || !ok {
		return decimalUnsupported(token.Pow, o, right, err)
	}
	exp := d.ToGo()
	if !exp.IsInteger() {
		// decimal.Pow handles only integer exponents; use float64 for
		// fractional powers (e.g. square roots via `** 0.5`).
		base, _ := o.ToGo().Float64()
		e, _ := exp.Float64()
		return Float(math.Pow(base, e)), nil
	}
	return Decimal(o.ToGo().Pow(exp)), nil
}

func (o Decimal) BinOpQuo(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil || !ok {
		return decimalUnsupported(token.Quo, o, right, err)
	}
	return Decimal(o.ToGo().Div(d.ToGo())), nil
}

func (o Decimal) BinOpLess(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil {
		return nil, err
	}
	if ok {
		return Bool(o.ToGo().LessThan(d.ToGo())), nil
	}
	if right == Nil {
		return False, nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o Decimal) BinOpLessEq(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil {
		return nil, err
	}
	if ok {
		return Bool(o.ToGo().LessThanOrEqual(d.ToGo())), nil
	}
	if right == Nil {
		return False, nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o Decimal) BinOpGreater(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil {
		return nil, err
	}
	if ok {
		return Bool(o.ToGo().GreaterThan(d.ToGo())), nil
	}
	if right == Nil {
		return True, nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o Decimal) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	d, ok, err := decimalRHS(right)
	if err != nil {
		return nil, err
	}
	if ok {
		return Bool(o.ToGo().GreaterThanOrEqual(d.ToGo())), nil
	}
	if right == Nil {
		return True, nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// decimalUnsupported returns a conversion error or an unsupported-operand error
// for a Decimal arithmetic operator.
func decimalUnsupported(tok token.Token, o Decimal, right Object, err error) (Object, error) {
	if err != nil {
		return nil, err
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

// Format implements fmt.Formatter interface.
func (o Decimal) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, o.ToGo())
}

func (o Decimal) ToBytes() (b Bytes, err error) {
	return o.ToGo().MarshalBinary()
}

func (o Decimal) CallName(name string, c Call) (_ Object, err error) {
	switch name {
	case "trunc":
		prec := &Arg{
			Name:          "precision",
			TypeAssertion: TypeAssertionFromTypes(TInt),
		}
		if err = c.Args.Destructure(prec); err != nil {
			return
		}
		return Decimal(decimal.Decimal(o).Truncate(int32(prec.Value.(Int)))), nil
	default:
		return nil, ErrInvalidIndex.NewError(name)
	}
}

func DecimalFromUint(v Uint) Decimal {
	return Decimal(decimal.NewFromBigInt(new(big.Int).SetUint64(uint64(v)), 0))
}

func DecimalFromInt(v Int) Decimal {
	return Decimal(decimal.NewFromInt(int64(v)))
}

func DecimalFromFloat(v Float) Decimal {
	return Decimal(decimal.NewFromFloat(float64(v)))
}

func DecimalFromString(v Str) (Decimal, error) {
	r, err := decimal.NewFromString(string(v))
	return Decimal(r), err
}

func MustDecimalFromString(v Str) Decimal {
	r, _ := decimal.NewFromString(string(v))
	return Decimal(r)
}

var DecimalZero = Decimal(decimal.Zero)

// Char represents a rune and implements Object interface.
type Char rune

func (o Char) Type() ObjectType {
	return TChar
}

func (o Char) ToString() string {
	return string(o)
}

// Equal implements Object interface.
func (o Char) Equal(right Object) bool {
	switch v := right.(type) {
	case Char:
		return o == v
	case Int:
		return Int(o) == v
	case Uint:
		return Uint(o) == v
	case Float:
		return Float(o) == v
	case Bool:
		if v {
			return o == 1
		}
		return o == 0
	}
	return false
}

// IsFalsy implements Object interface.
func (o Char) IsFalsy() bool { return o == 0 }

func (o Char) BinOpAdd(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o + v, nil
	case Int:
		return o + Char(v), nil
	case Uint:
		return o + Char(v), nil
	case Bool:
		return o.BinOpAdd(vm, Char(boolInt(v)))
	case Str:
		var sb strings.Builder
		sb.Grow(len(v) + 4)
		sb.WriteRune(rune(o))
		sb.WriteString(string(v))
		return Str(sb.String()), nil
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpSub(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o - v, nil
	case Int:
		return o - Char(v), nil
	case Uint:
		return o - Char(v), nil
	case Bool:
		return o.BinOpSub(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpMul(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o * v, nil
	case Bool:
		return o.BinOpMul(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Mul.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpQuo(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return o / v, nil
	case Bool:
		return o.BinOpQuo(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Quo.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpRem(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o % v, nil
	case Bool:
		return o.BinOpRem(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Rem.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpAnd(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o & v, nil
	case Bool:
		return o.BinOpAnd(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.And.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpOr(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o | v, nil
	case Bool:
		return o.BinOpOr(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Or.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpXor(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o ^ v, nil
	case Bool:
		return o.BinOpXor(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Xor.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpAndNot(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o &^ v, nil
	case Bool:
		return o.BinOpAndNot(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.AndNot.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpShl(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o << v, nil
	case Bool:
		return o.BinOpShl(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Shl.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpShr(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return o >> v, nil
	case Bool:
		return o.BinOpShr(vm, Char(boolInt(v)))
	}
	return nil, NewOperandTypeError(token.Shr.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpLess(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return Bool(o < v), nil
	case Int:
		return Bool(Int(o) < v), nil
	case Uint:
		return Bool(Uint(o) < v), nil
	case Bool:
		return o.BinOpLess(vm, Char(boolInt(v)))
	case *NilType:
		return False, nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpLessEq(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return Bool(o <= v), nil
	case Int:
		return Bool(Int(o) <= v), nil
	case Uint:
		return Bool(Uint(o) <= v), nil
	case Bool:
		return o.BinOpLessEq(vm, Char(boolInt(v)))
	case *NilType:
		return False, nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpGreater(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return Bool(o > v), nil
	case Int:
		return Bool(Int(o) > v), nil
	case Uint:
		return Bool(Uint(o) > v), nil
	case Bool:
		return o.BinOpGreater(vm, Char(boolInt(v)))
	case *NilType:
		return True, nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o Char) BinOpGreaterEq(vm *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		return Bool(o >= v), nil
	case Int:
		return Bool(Int(o) >= v), nil
	case Uint:
		return Bool(Uint(o) >= v), nil
	case Bool:
		return o.BinOpGreaterEq(vm, Char(boolInt(v)))
	case *NilType:
		return True, nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// Format implements fmt.Formatter interface.
func (o Char) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, rune(o))
}

func (o Char) ToBytes() (Bytes, error) {
	return []byte(string([]rune{rune(o)})), nil
}
