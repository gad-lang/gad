// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"strconv"
	"time"

	"github.com/gad-lang/gad/token"
)

// DurationType is the object type of Duration values (the gad `duration` type).
// It is callable as a constructor: Duration(int) from a nanosecond count, or
// Duration(str) parsing a Go duration string ("1h30m"). See also strToDuration.
var DurationType = NewBuiltinObjType("duration").WithNew(NewDurationFunc)

// NewDurationFunc is the duration(...) constructor: a duration pass-through, an
// int/uint (nanoseconds) or a string (see strToDuration, e.g. "1h30m").
func NewDurationFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case Duration:
		return v, nil
	case Int:
		return Duration(v), nil
	case Uint:
		return Duration(v), nil
	case Str:
		d, err := strToDuration(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return d, nil
	}
	return Nil, NewArgumentTypeError("1st", "int|str", c.Args.Get(0).Type().Name())
}

// Duration is a span of time with nanosecond precision; it mirrors Go's
// time.Duration and is one of the time module's value types.
type Duration time.Duration

var _ NameCallerObject = Duration(0)

func (Duration) Type() ObjectType { return DurationType }

// ToString renders the duration like Go does, e.g. "1h30m0s".
func (o Duration) ToString() string { return time.Duration(o).String() }

// Print writes the duration (Printabler); without it the printer's reflection
// fallback would recurse on this named-int Object.
func (o Duration) Print(s *PrinterState) error {
	if s.IsRepr {
		defer s.WrapRepr(o)()
	}
	return s.WriteString(o.ToString())
}

// MarshalJSON encodes the duration as a JSON string, e.g. "1h30m0s".
func (o Duration) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(o.ToString())), nil
}

// IsFalsy reports whether the duration is zero.
func (o Duration) IsFalsy() bool { return o == 0 }

// Equal implements Object. A Duration equals another Duration or an int with the
// same nanosecond count.
func (o Duration) Equal(right Object) bool {
	switch v := right.(type) {
	case Duration:
		return o == v
	case Int:
		return int64(o) == int64(v)
	}
	return false
}

// BinaryOp supports duration arithmetic and comparison:
//   - `duration ± duration`      -> duration
//   - `duration * int`           -> duration (scale)
//   - `duration / int`           -> duration (scale)
//   - `duration / duration`      -> float (ratio)
//   - `duration % int|duration`  -> duration (remainder)
//   - ordered comparisons        -> bool
//
// The operators accept a Duration or an Int (taken as a nanosecond count).
func (o Duration) BinOpAdd(_ *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		return o + v, nil
	case Int:
		return o + Duration(v), nil
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpSub(_ *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		return o - v, nil
	case Int:
		return o - Duration(v), nil
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpMul(_ *VM, right Object) (Object, error) {
	if v, ok := right.(Int); ok {
		return Duration(int64(o) * int64(v)), nil
	}
	return nil, NewOperandTypeError(token.Mul.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpQuo(_ *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return Float(float64(o) / float64(v)), nil
	case Int:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return Duration(int64(o) / int64(v)), nil
	}
	return nil, NewOperandTypeError(token.Quo.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpRem(_ *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return o % v, nil
	case Int:
		if v == 0 {
			return nil, ErrZeroDivision
		}
		return Duration(int64(o) % int64(v)), nil
	}
	return nil, NewOperandTypeError(token.Rem.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpLess(_ *VM, right Object) (Object, error) {
	if v, ok := durationRHS(right); ok {
		return Bool(o < v), nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpLessEq(_ *VM, right Object) (Object, error) {
	if v, ok := durationRHS(right); ok {
		return Bool(o <= v), nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpGreater(_ *VM, right Object) (Object, error) {
	if v, ok := durationRHS(right); ok {
		return Bool(o > v), nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o Duration) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	if v, ok := durationRHS(right); ok {
		return Bool(o >= v), nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// durationRHS converts a comparable right operand (Duration or Int) to a
// Duration.
func durationRHS(right Object) (Duration, bool) {
	switch v := right.(type) {
	case Duration:
		return v, true
	case Int:
		return Duration(v), true
	}
	return 0, false
}

// CallName dispatches the duration methods (Go time.Duration accessors).
func (o Duration) CallName(name string, c Call) (Object, error) {
	d := time.Duration(o)
	switch name {
	case "nanoseconds":
		return Int(d.Nanoseconds()), nil
	case "microseconds":
		return Int(d.Microseconds()), nil
	case "milliseconds":
		return Int(d.Milliseconds()), nil
	case "seconds":
		return Float(d.Seconds()), nil
	case "minutes":
		return Float(d.Minutes()), nil
	case "hours":
		return Float(d.Hours()), nil
	case "round":
		unit, err := truncateUnitArg(c)
		if err != nil {
			return Nil, err
		}
		rd, err := roundDurationUnit(d, unit)
		if err != nil {
			return Nil, err
		}
		return Duration(rd), nil
	case "trunc":
		unit, err := truncateUnitArg(c)
		if err != nil {
			return Nil, err
		}
		td, err := truncateDurationUnit(d, unit)
		if err != nil {
			return Nil, err
		}
		return Duration(td), nil
	}
	return Nil, ErrInvalidIndex.NewError(name)
}

// strToDuration parses a Go duration string (e.g. "1h30m") into a Duration.
func strToDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return Duration(d), nil
}
