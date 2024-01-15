// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/internal/compat"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// Int represents signed integer values and implements Object interface.
type Int int64

func (o Int) Type() ObjectType {
	return DetectTypeOf(o)
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

// BinaryOp implements Object interface.
func (o Int) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		switch tok {
		case token.Add:
			return o + v, nil
		case token.Sub:
			return o - v, nil
		case token.Mul:
			return o * v, nil
		case token.Quo:
			if v == 0 {
				return nil, ErrZeroDivision
			}
			return o / v, nil
		case token.Rem:
			return o % v, nil
		case token.And:
			return o & v, nil
		case token.Or:
			return o | v, nil
		case token.Xor:
			return o ^ v, nil
		case token.AndNot:
			return o &^ v, nil
		case token.Shl:
			return o << v, nil
		case token.Shr:
			return o >> v, nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Uint:
		return Uint(o).BinaryOp(vm, tok, right)
	case Float:
		return Float(o).BinaryOp(vm, tok, right)
	case Decimal:
		return DecimalFromInt(o).BinaryOp(vm, tok, right)
	case Char:
		switch tok {
		case token.Add:
			return Char(o) + v, nil
		case token.Sub:
			return Char(o) - v, nil
		case token.Less:
			return Bool(o < Int(v)), nil
		case token.LessEq:
			return Bool(o <= Int(v)), nil
		case token.Greater:
			return Bool(o > Int(v)), nil
		case token.GreaterEq:
			return Bool(o >= Int(v)), nil
		}
	case Bool:
		if v {
			right = Int(1)
		} else {
			right = Int(0)
		}
		return o.BinaryOp(vm, tok, right)
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name(),
	)
}

// Format implements fmt.Formatter interface.
func (o Int) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, int64(o))
}

// Uint represents unsigned integer values and implements Object interface.
type Uint uint64

func (o Uint) Type() ObjectType {
	return DetectTypeOf(o)
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

// BinaryOp implements Object interface.
func (o Uint) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Uint:
		switch tok {
		case token.Add:
			return o + v, nil
		case token.Sub:
			return o - v, nil
		case token.Mul:
			return o * v, nil
		case token.Quo:
			if v == 0 {
				return nil, ErrZeroDivision
			}
			return o / v, nil
		case token.Rem:
			return o % v, nil
		case token.And:
			return o & v, nil
		case token.Or:
			return o | v, nil
		case token.Xor:
			return o ^ v, nil
		case token.AndNot:
			return o &^ v, nil
		case token.Shl:
			return o << v, nil
		case token.Shr:
			return o >> v, nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Int:
		return o.BinaryOp(vm, tok, Uint(v))
	case Float:
		return Float(o).BinaryOp(vm, tok, right)
	case Decimal:
		return DecimalFromUint(o).BinaryOp(vm, tok, right)
	case Char:
		switch tok {
		case token.Add:
			return Char(o) + v, nil
		case token.Sub:
			return Char(o) - v, nil
		case token.Less:
			return Bool(o < Uint(v)), nil
		case token.LessEq:
			return Bool(o <= Uint(v)), nil
		case token.Greater:
			return Bool(o > Uint(v)), nil
		case token.GreaterEq:
			return Bool(o >= Uint(v)), nil
		}
	case Bool:
		if v {
			right = Uint(1)
		} else {
			right = Uint(0)
		}
		return o.BinaryOp(vm, tok, right)
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name(),
	)
}

// Format implements fmt.Formatter interface.
func (o Uint) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, uint64(o))
}

// Float represents float values and implements Object interface.
type Float float64

func (o Float) Type() ObjectType {
	return DetectTypeOf(o)
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

// BinaryOp implements Object interface.
func (o Float) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Float:
		switch tok {
		case token.Add:
			return o + v, nil
		case token.Sub:
			return o - v, nil
		case token.Mul:
			return o * v, nil
		case token.Quo:
			if v == 0 {
				return nil, ErrZeroDivision
			}
			return o / v, nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Int:
		return o.BinaryOp(vm, tok, Float(v))
	case Uint:
		return o.BinaryOp(vm, tok, Float(v))
	case Decimal:
		return DecimalFromFloat(o).BinaryOp(vm, tok, right)
	case Bool:
		if v {
			right = Float(1)
		} else {
			right = Float(0)
		}
		return o.BinaryOp(vm, tok, right)
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name(),
	)
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
	return o.Go().MarshalBinary()
}

func (o Decimal) Go() decimal.Decimal {
	return decimal.Decimal(o)
}

func (o Decimal) Type() ObjectType {
	return DetectTypeOf(o)
}

func (o Decimal) ToString() string {
	return o.Go().String()
}

// Equal implements Object interface.
func (o Decimal) Equal(right Object) bool {
	switch v := right.(type) {
	case Decimal:
		return o.Go().Equal(v.Go())
	case Int:
		return o.Go().Equal(decimal.Decimal(DecimalFromInt(v)))
	case Uint:
		return o.Go().Equal(decimal.Decimal(DecimalFromUint(v)))
	case Float:
		return o.Go().Equal(decimal.Decimal(DecimalFromFloat(v)))
	case Bool:
		return o.Go().IsZero() != bool(v)
	}
	return false
}

// IsFalsy implements Object interface.
func (o Decimal) IsFalsy() bool {
	// IEEE 754 says that only NaNs satisfy f != f.
	// See math.IsNan
	return o.Go().IsZero()
}

// BinaryOp implements Object interface.
func (o Decimal) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Decimal:
		switch tok {
		case token.Add:
			return Decimal(o.Go().Add(v.Go())), nil
		case token.Sub:
			return Decimal(o.Go().Sub(v.Go())), nil
		case token.Mul:
			return Decimal(o.Go().Mul(v.Go())), nil
		case token.Quo:
			return Decimal(o.Go().Div(v.Go())), nil
		case token.Less:
			return Bool(o.Go().LessThan(v.Go())), nil
		case token.LessEq:
			return Bool(o.Go().LessThanOrEqual(v.Go())), nil
		case token.Greater:
			return Bool(o.Go().GreaterThan(v.Go())), nil
		case token.GreaterEq:
			return Bool(o.Go().GreaterThanOrEqual(v.Go())), nil
		}
	case Int:
		return o.BinaryOp(vm, tok, DecimalFromInt(v))
	case Uint:
		return o.BinaryOp(vm, tok, DecimalFromUint(v))
	case Float:
		return o.BinaryOp(vm, tok, DecimalFromFloat(v))
	case Char:
		return o.BinaryOp(vm, tok, DecimalFromUint(Uint(v)))
	case Str:
		d, err := DecimalFromString(v)
		if err != nil {
			return nil, ErrType.NewError(err.Error())
		}
		return o.BinaryOp(vm, tok, d)
	case Bytes:
		var d decimal.Decimal
		if err := d.UnmarshalBinary(v); err != nil {
			return nil, err
		}
		return o.BinaryOp(vm, tok, Decimal(d))
	case Bool:
		if v {
			right = DecimalFromUint(1)
		} else {
			right = DecimalFromUint(0)
		}
		return o.BinaryOp(vm, tok, right)
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name(),
	)
}

// Format implements fmt.Formatter interface.
func (o Decimal) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, o.Go())
}

func (o Decimal) ToBytes() (b Bytes, err error) {
	return o.Go().MarshalBinary()
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
	return DetectTypeOf(o)
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

// BinaryOp implements Object interface.
func (o Char) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Char:
		switch tok {
		case token.Add:
			return o + v, nil
		case token.Sub:
			return o - v, nil
		case token.Mul:
			return o * v, nil
		case token.Quo:
			if v == 0 {
				return nil, ErrZeroDivision
			}
			return o / v, nil
		case token.Rem:
			return o % v, nil
		case token.And:
			return o & v, nil
		case token.Or:
			return o | v, nil
		case token.Xor:
			return o ^ v, nil
		case token.AndNot:
			return o &^ v, nil
		case token.Shl:
			return o << v, nil
		case token.Shr:
			return o >> v, nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Int:
		switch tok {
		case token.Add:
			return o + Char(v), nil
		case token.Sub:
			return o - Char(v), nil
		case token.Less:
			return Bool(Int(o) < v), nil
		case token.LessEq:
			return Bool(Int(o) <= v), nil
		case token.Greater:
			return Bool(Int(o) > v), nil
		case token.GreaterEq:
			return Bool(Int(o) >= v), nil
		}
	case Uint:
		switch tok {
		case token.Add:
			return o + Char(v), nil
		case token.Sub:
			return o - Char(v), nil
		case token.Less:
			return Bool(Uint(o) < v), nil
		case token.LessEq:
			return Bool(Uint(o) <= v), nil
		case token.Greater:
			return Bool(Uint(o) > v), nil
		case token.GreaterEq:
			return Bool(Uint(o) >= v), nil
		}
	case Bool:
		if v {
			right = Char(1)
		} else {
			right = Char(0)
		}
		return o.BinaryOp(vm, tok, right)
	case Str:
		if tok == token.Add {
			var sb strings.Builder
			sb.Grow(len(v) + 4)
			sb.WriteRune(rune(o))
			sb.WriteString(string(v))
			return Str(sb.String()), nil
		}
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name(),
	)
}

// Format implements fmt.Formatter interface.
func (o Char) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, rune(o))
}

func (o Char) ToBytes() (Bytes, error) {
	return []byte(string([]rune{rune(o)})), nil
}
