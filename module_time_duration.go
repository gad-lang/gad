// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"time"

	"github.com/gad-lang/gad/token"
)

// DurationType is the object type of Duration values (the gad `duration` type).
// It is callable as a constructor: Duration(int) from a nanosecond count, or
// Duration(str) parsing a Go duration string ("1h30m"). See also strToDuration.
var DurationType = NewBuiltinObjType("duration").WithNew(durationNew)

func durationNew(c Call) (Object, error) {
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

// BinaryOp supports duration arithmetic and comparison: duration ± duration,
// duration */ int (scale), and the ordered comparisons. An int operand is taken
// as a nanosecond count.
func (o Duration) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	d, ok := durationOperand(right)
	if !ok {
		return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
	}
	switch tok {
	case token.Add:
		return o + d, nil
	case token.Sub:
		return o - d, nil
	case token.Less:
		return Bool(o < d), nil
	case token.LessEq:
		return Bool(o <= d), nil
	case token.Greater:
		return Bool(o > d), nil
	case token.GreaterEq:
		return Bool(o >= d), nil
	}
	// scaling by a plain int
	if i, isInt := right.(Int); isInt {
		switch tok {
		case token.Mul:
			return Duration(int64(o) * int64(i)), nil
		case token.Quo:
			if i == 0 {
				return nil, ErrZeroDivision
			}
			return Duration(int64(o) / int64(i)), nil
		}
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

// durationOperand interprets right as a Duration: a Duration directly, or an int
// nanosecond count.
func durationOperand(right Object) (Duration, bool) {
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
		return durationUnaryDur(c, func(m time.Duration) Object {
			return Duration(d.Round(m))
		})
	case "truncate":
		return durationUnaryDur(c, func(m time.Duration) Object {
			return Duration(d.Truncate(m))
		})
	}
	return Nil, ErrInvalidIndex.NewError(name)
}

func durationUnaryDur(c Call, fn func(m time.Duration) Object) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	m, ok := durationOperand(c.Args.Get(0))
	if !ok {
		return Nil, NewArgumentTypeError("1st", "duration|int", c.Args.Get(0).Type().Name())
	}
	return fn(time.Duration(m)), nil
}

// strToDuration parses a Go duration string (e.g. "1h30m") into a Duration.
func strToDuration(s string) (Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return Duration(d), nil
}
