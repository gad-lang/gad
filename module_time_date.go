// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"time"

	"github.com/gad-lang/gad/token"
)

// CalendarDateType is the object type of CalendarDate values (the gad
// `calendarDate` type). It is callable as a constructor:
// CalendarDate(uint|int) from a YYYYMMDD integer, or CalendarDate(str) parsing
// "YYYYMMDD"/"YYYY-MM-DD". See also strToDate.
var CalendarDateType = NewBuiltinObjType("calendarDate").WithNew(calendarDateNew)

func calendarDateNew(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case CalendarDate:
		return v, nil
	case CalendarTime:
		return v.Date(), nil
	case *Time:
		return CalendarDateFromTime(v.Value), nil
	case Uint:
		return CalendarDate(v), nil
	case Int:
		return CalendarDate(v), nil
	case Str:
		d, err := strToDate(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return d, nil
	}
	return Nil, NewArgumentTypeError("1st", "uint|int|str", c.Args.Get(0).Type().Name())
}

// CalendarDate is a calendar date encoded as the unsigned integer YYYYMMDD
// (e.g. 20260131 for 2026-01-31); it mirrors a Go uint and is one of the time
// module's value types.
type CalendarDate uint

var _ NameCallerObject = CalendarDate(0)

// NewCalendarDate builds a CalendarDate from its year, month and day components.
func NewCalendarDate(year, month, day int) CalendarDate {
	return CalendarDate(year*10000 + month*100 + day)
}

// CalendarDateFromTime returns the CalendarDate (YYYYMMDD) part of t.
func CalendarDateFromTime(t time.Time) CalendarDate {
	return NewCalendarDate(t.Year(), int(t.Month()), t.Day())
}

func (CalendarDate) Type() ObjectType { return CalendarDateType }

func (o CalendarDate) Year() int  { return int(o) / 10000 }
func (o CalendarDate) Month() int { return (int(o) / 100) % 100 }
func (o CalendarDate) Day() int   { return int(o) % 100 }

// Time returns midnight of this date in the given location (UTC when nil).
func (o CalendarDate) Time(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	return time.Date(o.Year(), time.Month(o.Month()), o.Day(), 0, 0, 0, 0, loc)
}

// ToString renders the date as YYYY-MM-DD.
func (o CalendarDate) ToString() string {
	return fmt.Sprintf("%04d-%02d-%02d", o.Year(), o.Month(), o.Day())
}

// Print writes the date (Printabler); without it the printer's reflection
// fallback would recurse on this named-uint Object.
func (o CalendarDate) Print(s *PrinterState) error {
	if s.IsRepr {
		defer s.WrapRepr(o)()
	}
	return s.WriteString(o.ToString())
}

// IsFalsy reports whether the date is the zero value.
func (o CalendarDate) IsFalsy() bool { return o == 0 }

// Equal implements Object. A CalendarDate equals another CalendarDate or a
// uint/int with the same YYYYMMDD value.
func (o CalendarDate) Equal(right Object) bool {
	switch v := right.(type) {
	case CalendarDate:
		return o == v
	case Uint:
		return uint64(o) == uint64(v)
	case Int:
		return int64(o) == int64(v)
	}
	return false
}

// BinaryOp supports duration arithmetic, date difference and ordered
// comparisons:
//   - `calendarDate ± duration`  -> calendarDate when the result lands on
//     midnight (a whole number of days), otherwise calendarTime
//   - `calendarDate - calendarDate` -> duration
//   - ordered comparisons        -> bool (the YYYYMMDD encoding is monotonic)
func (o CalendarDate) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		switch tok {
		case token.Add:
			return o.addDuration(time.Duration(v)), nil
		case token.Sub:
			return o.addDuration(-time.Duration(v)), nil
		}
	case CalendarDate:
		switch tok {
		case token.Sub:
			return Duration(o.Time(time.UTC).Sub(v.Time(time.UTC))), nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Uint, Int:
		r := CalendarDate(toUint64(v))
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
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

// addDuration adds d to midnight of this date, returning a CalendarDate when the
// result lands exactly on midnight (a whole number of days) and a CalendarTime
// otherwise (the duration carries a time-of-day part).
func (o CalendarDate) addDuration(d time.Duration) Object {
	t := o.Time(time.UTC).Add(d)
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0 {
		return CalendarDateFromTime(t)
	}
	return CalendarTimeFromTime(t)
}

// toUint64 reads an Int or Uint as a uint64.
func toUint64(v Object) uint64 {
	switch n := v.(type) {
	case Uint:
		return uint64(n)
	case Int:
		return uint64(n)
	}
	return 0
}

// CallName dispatches the date accessor methods.
func (o CalendarDate) CallName(name string, c Call) (Object, error) {
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
func strToDate(s string) (CalendarDate, error) {
	var y, m, d int
	if _, err := fmt.Sscanf(s, "%04d-%02d-%02d", &y, &m, &d); err == nil && len(s) == 10 {
		return NewCalendarDate(y, m, d), nil
	}
	if _, err := fmt.Sscanf(s, "%04d%02d%02d", &y, &m, &d); err == nil && len(s) == 8 {
		return NewCalendarDate(y, m, d), nil
	}
	return 0, fmt.Errorf("invalid date %q (want YYYYMMDD or YYYY-MM-DD)", s)
}
