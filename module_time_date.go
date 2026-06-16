// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"time"

	"github.com/gad-lang/gad/token"
)

// DateType is the object type of Date values (the gad `date` type). It is
// callable as a constructor: Date(uint|int) from a YYYYMMDD integer, or
// Date(str) parsing "YYYYMMDD"/"YYYY-MM-DD". See also strToDate.
var DateType = NewBuiltinObjType("date").WithNew(dateNew)

func dateNew(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case Date:
		return v, nil
	case Uint:
		return Date(v), nil
	case Int:
		return Date(v), nil
	case Str:
		d, err := strToDate(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return d, nil
	}
	return Nil, NewArgumentTypeError("1st", "uint|int|str", c.Args.Get(0).Type().Name())
}

// Date is a calendar date encoded as the unsigned integer YYYYMMDD (e.g.
// 20260131 for 2026-01-31); it mirrors a Go uint and is one of the time
// module's value types.
type Date uint

var _ NameCallerObject = Date(0)

// NewDate builds a Date from its year, month and day components.
func NewDate(year, month, day int) Date {
	return Date(year*10000 + month*100 + day)
}

// DateFromTime returns the Date (YYYYMMDD) part of t.
func DateFromTime(t time.Time) Date {
	return NewDate(t.Year(), int(t.Month()), t.Day())
}

func (Date) Type() ObjectType { return DateType }

func (o Date) Year() int  { return int(o) / 10000 }
func (o Date) Month() int { return (int(o) / 100) % 100 }
func (o Date) Day() int   { return int(o) % 100 }

// Time returns midnight of this date in the given location (UTC when nil).
func (o Date) Time(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(o.Year(), time.Month(o.Month()), o.Day(), 0, 0, 0, 0, loc)
}

// ToString renders the date as YYYY-MM-DD.
func (o Date) ToString() string {
	return fmt.Sprintf("%04d-%02d-%02d", o.Year(), o.Month(), o.Day())
}

// Print writes the date (Printabler); without it the printer's reflection
// fallback would recurse on this named-uint Object.
func (o Date) Print(s *PrinterState) error {
	if s.IsRepr {
		defer s.WrapRepr(o)()
	}
	return s.WriteString(o.ToString())
}

// IsFalsy reports whether the date is the zero value.
func (o Date) IsFalsy() bool { return o == 0 }

// Equal implements Object. A Date equals another Date or a uint/int with the
// same YYYYMMDD value.
func (o Date) Equal(right Object) bool {
	switch v := right.(type) {
	case Date:
		return o == v
	case Uint:
		return uint64(o) == uint64(v)
	case Int:
		return int64(o) == int64(v)
	}
	return false
}

// BinaryOp supports the ordered comparisons between dates (their YYYYMMDD
// encoding compares chronologically).
func (o Date) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	var r Date
	switch v := right.(type) {
	case Date:
		r = v
	case Uint:
		r = Date(v)
	case Int:
		r = Date(v)
	default:
		return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
	}
	switch tok {
	case token.Less:
		return Bool(o < r), nil
	case token.LessEq:
		return Bool(o <= r), nil
	case token.Greater:
		return Bool(o > r), nil
	case token.GreaterEq:
		return Bool(o >= r), nil
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

// CallName dispatches the date accessor methods.
func (o Date) CallName(name string, c Call) (Object, error) {
	switch name {
	case "year":
		return Int(o.Year()), nil
	case "month":
		return Int(o.Month()), nil
	case "day":
		return Int(o.Day()), nil
	case "time":
		loc := time.UTC
		if c.Args.Length() > 0 {
			if l, ok := c.Args.Get(0).(*Location); ok {
				loc = l.Value
			}
		}
		return &Time{Value: o.Time(loc)}, nil
	}
	return Nil, ErrInvalidIndex.NewError(name)
}

// strToDate parses a date from "YYYYMMDD" or "YYYY-MM-DD".
func strToDate(s string) (Date, error) {
	var y, m, d int
	if _, err := fmt.Sscanf(s, "%04d-%02d-%02d", &y, &m, &d); err == nil && len(s) == 10 {
		return NewDate(y, m, d), nil
	}
	if _, err := fmt.Sscanf(s, "%04d%02d%02d", &y, &m, &d); err == nil && len(s) == 8 {
		return NewDate(y, m, d), nil
	}
	return 0, fmt.Errorf("invalid date %q (want YYYYMMDD or YYYY-MM-DD)", s)
}
