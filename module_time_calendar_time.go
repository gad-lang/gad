// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gad-lang/gad/token"
)

// CalendarTimeType is the object type of CalendarTime values (the gad
// `calendarTime` type). It is callable as a constructor:
// CalendarTime(uint|int) from a nanosecond count, or CalendarTime(str) parsing
// a zone-less timestamp. See also strToCalendarTime.
var CalendarTimeType = NewBuiltinObjType("calendarTime").WithNew(calendarTimeNew)

func calendarTimeNew(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case CalendarTime:
		return v, nil
	case *Time:
		return CalendarTimeFromTime(v.Value), nil
	case CalendarDate:
		return CalendarTimeFromTime(v.Time(time.UTC)), nil
	case Uint:
		return CalendarTime(v), nil
	case Int:
		return CalendarTime(v), nil
	case Str:
		t, err := strToCalendarTime(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return t, nil
	}
	return Nil, NewArgumentTypeError("1st", "uint|int|str", c.Args.Get(0).Type().Name())
}

// CalendarTime is a zone-less wall-clock timestamp stored as the number of
// nanoseconds since the Unix epoch (interpreted as UTC wall clock). Unlike
// *Time it carries no location; it mirrors a Go uint64 and is one of the time
// module's value types. Instants before 1970-01-01 are not representable.
type CalendarTime uint64

var _ NameCallerObject = CalendarTime(0)

// NewCalendarTime builds a CalendarTime from its components (UTC wall clock).
func NewCalendarTime(year, month, day, hour, min, sec, nsec int) CalendarTime {
	t := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.UTC)
	return CalendarTimeFromTime(t)
}

// CalendarTimeFromTime returns the zone-less CalendarTime of t (its wall-clock
// fields are kept; the zone is dropped).
func CalendarTimeFromTime(t time.Time) CalendarTime {
	w := time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return CalendarTime(w.UnixNano())
}

func (CalendarTime) Type() ObjectType { return CalendarTimeType }

// wall returns the value as a UTC time.Time.
func (o CalendarTime) wall() time.Time { return time.Unix(0, int64(o)).UTC() }

func (o CalendarTime) Year() int       { return o.wall().Year() }
func (o CalendarTime) Month() int      { return int(o.wall().Month()) }
func (o CalendarTime) Day() int        { return o.wall().Day() }
func (o CalendarTime) Hour() int       { return o.wall().Hour() }
func (o CalendarTime) Minute() int     { return o.wall().Minute() }
func (o CalendarTime) Second() int     { return o.wall().Second() }
func (o CalendarTime) Nanosecond() int { return o.wall().Nanosecond() }

// Time returns this wall time in the given location (UTC when nil).
func (o CalendarTime) Time(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	w := o.wall()
	return time.Date(w.Year(), w.Month(), w.Day(),
		w.Hour(), w.Minute(), w.Second(), w.Nanosecond(), loc)
}

// Date returns the calendar date (YYYYMMDD) part of this timestamp.
func (o CalendarTime) Date() CalendarDate {
	w := o.wall()
	return NewCalendarDate(w.Year(), int(w.Month()), w.Day())
}

// ToString renders the timestamp as "YYYY-MM-DD HH:MM:SS[.fraction]".
func (o CalendarTime) ToString() string {
	return o.wall().Format("2006-01-02 15:04:05.999999999")
}

// Print writes the timestamp (Printabler); without it the printer's reflection
// fallback would recurse on this named-uint64 Object.
func (o CalendarTime) Print(s *PrinterState) error {
	if s.IsRepr {
		defer s.WrapRepr(o)()
	}
	return s.WriteString(o.ToString())
}

// MarshalJSON encodes the timestamp as a JSON string "YYYY-MM-DD HH:MM:SS".
func (o CalendarTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(o.ToString())), nil
}

// IsFalsy reports whether the timestamp is the zero value.
func (o CalendarTime) IsFalsy() bool { return o == 0 }

// Equal implements Object. A CalendarTime equals another CalendarTime or a
// uint/int holding the same nanosecond count.
func (o CalendarTime) Equal(right Object) bool {
	switch v := right.(type) {
	case CalendarTime:
		return o == v
	case Uint:
		return uint64(o) == uint64(v)
	case Int:
		return int64(o) == int64(v)
	}
	return false
}

// BinaryOp supports duration arithmetic and ordered comparisons:
//   - `calendarTime ± int|duration` -> calendarTime (the int is nanoseconds)
//   - `calendarTime - calendarTime`  -> duration
//   - `calendarTime <|<=|>|>= calendarTime` -> bool
func (o CalendarTime) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Int:
		return o.shift(tok, int64(v))
	case Duration:
		return o.shift(tok, int64(v))
	case CalendarTime:
		switch tok {
		case token.Sub:
			return Duration(int64(o) - int64(v)), nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

// shift applies a nanosecond offset for the Add/Sub operators.
func (o CalendarTime) shift(tok token.Token, ns int64) (Object, error) {
	switch tok {
	case token.Add:
		return CalendarTime(int64(o) + ns), nil
	case token.Sub:
		return CalendarTime(int64(o) - ns), nil
	}
	return nil, NewOperandTypeError(tok.String(), CalendarTimeType.Name(), "int|duration")
}

// CallName dispatches the calendar-time accessor methods.
func (o CalendarTime) CallName(name string, c Call) (Object, error) {
	switch name {
	case "year":
		return Int(o.Year()), nil
	case "month":
		return Int(o.Month()), nil
	case "day":
		return Int(o.Day()), nil
	case "hour":
		return Int(o.Hour()), nil
	case "minute":
		return Int(o.Minute()), nil
	case "second":
		return Int(o.Second()), nil
	case "ns":
		return Int(o.Nanosecond()), nil
	case "weekday":
		return Int(o.wall().Weekday()), nil
	case "unix":
		return Int(o.wall().Unix()), nil
	case "add":
		d, err := calendarTimeShiftArg(c)
		if err != nil {
			return Nil, err
		}
		return CalendarTime(int64(o) + d), nil
	case "sub":
		d, err := calendarTimeShiftArg(c)
		if err != nil {
			return Nil, err
		}
		return CalendarTime(int64(o) - d), nil
	case "trunc":
		unit, err := truncateUnitArg(c)
		if err != nil {
			return Nil, err
		}
		t, err := truncateTimeUnit(o.wall(), unit)
		if err != nil {
			return Nil, err
		}
		return CalendarTimeFromTime(t), nil
	case "addDate":
		var years, months, days int
		if err := c.Args.Destructure(
			&Arg{Name: "years", TypeAssertion: TypeAssertionFromTypes(TInt)},
			&Arg{Name: "months", TypeAssertion: TypeAssertionFromTypes(TInt)},
			&Arg{Name: "days", TypeAssertion: TypeAssertionFromTypes(TInt)},
		); err != nil {
			return Nil, err
		}
		return CalendarTimeFromTime(o.wall().AddDate(years, months, days)), nil
	case "date":
		return o.Date(), nil
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

// calendarTimeShiftArg reads a single int|duration argument as a nanosecond
// offset for the .add/.sub methods.
func calendarTimeShiftArg(c Call) (int64, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return 0, err
	}
	switch v := c.Args.Get(0).(type) {
	case Int:
		return int64(v), nil
	case Duration:
		return int64(v), nil
	}
	return 0, NewArgumentTypeError("1st", "int|duration", c.Args.Get(0).Type().Name())
}

// calendarTimeLayouts are the zone-less layouts strToCalendarTime accepts.
var calendarTimeLayouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// strToCalendarTime parses a zone-less timestamp ("2026-01-31T23:59:55.001",
// "2026-01-31 23:59:55" or "2026-01-31") into a CalendarTime. A zone, if
// present, is rejected — CalendarTime is wall-clock only.
func strToCalendarTime(s string) (CalendarTime, error) {
	for _, layout := range calendarTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return CalendarTimeFromTime(t), nil
		}
	}
	return 0, fmt.Errorf("invalid calendar time %q (want YYYY-MM-DD[ HH:MM:SS], no zone)", s)
}
