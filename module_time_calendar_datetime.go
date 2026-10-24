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

// CalendarDateTimeType is the object type of CalendarDateTime values (the gad
// `cdtime` type). It is callable as a constructor:
// CalendarDateTime(uint|int) from a nanosecond count, or CalendarDateTime(str) parsing
// a zone-less timestamp. See also strToCalendarDateTime.
var CalendarDateTimeType = NewBuiltinObjType("cdtime").WithNew(NewCalendarDateTimeFunc)

// NewCalendarDateTimeFunc is the cdtime(...) constructor: a cdtime
// pass-through, a time/cdate (its wall-clock value), an int/uint
// (UnixNano) or a string (see strToCalendarDateTime).
func NewCalendarDateTimeFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case CalendarDateTime:
		return v, nil
	case *Time:
		return CalendarDateTimeFromTime(v.Value), nil
	case CalendarDate:
		return CalendarDateTimeFromTime(v.Time(time.UTC)), nil
	case Uint:
		return CalendarDateTime(v), nil
	case Int:
		return CalendarDateTime(v), nil
	case Str:
		t, err := strToCalendarDateTime(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return t, nil
	}
	return Nil, NewArgumentTypeError("1st", "uint|int|str", c.Args.Get(0).Type().Name())
}

// CalendarDateTime is a zone-less wall-clock timestamp stored as the number of
// nanoseconds since the Unix epoch (interpreted as UTC wall clock). Unlike
// *Time it carries no location; it mirrors a Go uint64 and is one of the time
// module's value types. Instants before 1970-01-01 are not representable.
type CalendarDateTime uint64

var _ NameCallerObject = CalendarDateTime(0)

// NewCalendarDateTime builds a CalendarDateTime from its components (UTC wall clock).
func NewCalendarDateTime(year, month, day, hour, min, sec, nsec int) CalendarDateTime {
	t := time.Date(year, time.Month(month), day, hour, min, sec, nsec, time.UTC)
	return CalendarDateTimeFromTime(t)
}

// CalendarDateTimeFromTime returns the zone-less CalendarDateTime of t (its wall-clock
// fields are kept; the zone is dropped).
func CalendarDateTimeFromTime(t time.Time) CalendarDateTime {
	w := time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
	return CalendarDateTime(w.UnixNano())
}

func (CalendarDateTime) Type() ObjectType { return CalendarDateTimeType }

// wall returns the value as a UTC time.Time.
func (o CalendarDateTime) wall() time.Time { return time.Unix(0, int64(o)).UTC() }

func (o CalendarDateTime) Year() int       { return o.wall().Year() }
func (o CalendarDateTime) Month() int      { return int(o.wall().Month()) }
func (o CalendarDateTime) Day() int        { return o.wall().Day() }
func (o CalendarDateTime) Hour() int       { return o.wall().Hour() }
func (o CalendarDateTime) Minute() int     { return o.wall().Minute() }
func (o CalendarDateTime) Second() int     { return o.wall().Second() }
func (o CalendarDateTime) Nanosecond() int { return o.wall().Nanosecond() }

// Time returns this wall time in the given location (UTC when nil).
func (o CalendarDateTime) Time(loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	w := o.wall()
	return time.Date(w.Year(), w.Month(), w.Day(),
		w.Hour(), w.Minute(), w.Second(), w.Nanosecond(), loc)
}

// Clock is the time of day of this timestamp.
func (o CalendarDateTime) Clock() TimeOfDay {
	return TimeOfDayFromTime(o.wall())
}

// Date returns the calendar date (YYYYMMDD) part of this timestamp.
func (o CalendarDateTime) Date() CalendarDate {
	w := o.wall()
	return NewCalendarDate(w.Year(), int(w.Month()), w.Day())
}

// ToString renders the timestamp as "YYYY-MM-DD HH:MM:SS[.fraction]".
func (o CalendarDateTime) ToString() string {
	return o.wall().Format("2006-01-02 15:04:05.999999999")
}

// Print writes the timestamp (Printabler); without it the printer's reflection
// fallback would recurse on this named-uint64 Object.
func (o CalendarDateTime) Print(s *PrinterState) error {
	return s.literalOr(o, func() string { return calendarDateTimeLiteral(o) }, o.ToString())
}

// MarshalJSON encodes the timestamp as a JSON string "YYYY-MM-DD HH:MM:SS".
func (o CalendarDateTime) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(o.ToString())), nil
}

// IsFalsy reports whether the timestamp is the zero value.
func (o CalendarDateTime) IsFalsy() bool { return o == 0 }

// Equal implements Object. A CalendarDateTime equals another CalendarDateTime or a
// uint/int holding the same nanosecond count.
func (o CalendarDateTime) Equal(right Object) bool {
	switch v := right.(type) {
	case CalendarDateTime:
		return o == v
	case Uint:
		return uint64(o) == uint64(v)
	case Int:
		return int64(o) == int64(v)
	}
	return false
}

// BinaryOp supports duration arithmetic and ordered comparisons:
//   - `cdtime ± int|duration` -> cdtime (the int is nanoseconds)
//   - `cdtime - cdtime`  -> duration
//   - `cdtime <|<=|>|>= cdtime` -> bool
func (o CalendarDateTime) BinOpAdd(_ *VM, right Object) (Object, error) {
	if ns, ok := calendarDateTimeShiftRHS(right); ok {
		return o.shift(token.Add, ns)
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o CalendarDateTime) BinOpSub(_ *VM, right Object) (Object, error) {
	if ns, ok := calendarDateTimeShiftRHS(right); ok {
		return o.shift(token.Sub, ns)
	}
	if v, ok := right.(CalendarDateTime); ok {
		return Duration(int64(o) - int64(v)), nil
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o CalendarDateTime) BinOpLess(_ *VM, right Object) (Object, error) {
	if v, ok := right.(CalendarDateTime); ok {
		return Bool(o < v), nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o CalendarDateTime) BinOpLessEq(_ *VM, right Object) (Object, error) {
	if v, ok := right.(CalendarDateTime); ok {
		return Bool(o <= v), nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

func (o CalendarDateTime) BinOpGreater(_ *VM, right Object) (Object, error) {
	if v, ok := right.(CalendarDateTime); ok {
		return Bool(o > v), nil
	}
	return nil, NewOperandTypeError(token.Greater.String(), o.Type().Name(), right.Type().Name())
}

func (o CalendarDateTime) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	if v, ok := right.(CalendarDateTime); ok {
		return Bool(o >= v), nil
	}
	return nil, NewOperandTypeError(token.GreaterEq.String(), o.Type().Name(), right.Type().Name())
}

// calendarDateTimeShiftRHS returns the nanosecond shift for an Int or Duration
// right operand.
func calendarDateTimeShiftRHS(right Object) (int64, bool) {
	switch v := right.(type) {
	case Int:
		return int64(v), true
	case Duration:
		return int64(v), true
	}
	return 0, false
}

// shift applies a nanosecond offset for the Add/Sub operators.
func (o CalendarDateTime) shift(tok token.Token, ns int64) (Object, error) {
	switch tok {
	case token.Add:
		return CalendarDateTime(int64(o) + ns), nil
	case token.Sub:
		return CalendarDateTime(int64(o) - ns), nil
	}
	return nil, NewOperandTypeError(tok.String(), CalendarDateTimeType.Name(), "int|duration")
}

// CallName dispatches the calendar-time accessor methods.
func (o CalendarDateTime) CallName(name string, c Call) (Object, error) {
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
	case "format":
		layout, err := timeLayoutArg(c)
		if err != nil {
			return Nil, err
		}
		return Str(o.wall().Format(layout)), nil
	case "add":
		d, err := calendarDateTimeShiftArg(c)
		if err != nil {
			return Nil, err
		}
		return CalendarDateTime(int64(o) + d), nil
	case "sub":
		d, err := calendarDateTimeShiftArg(c)
		if err != nil {
			return Nil, err
		}
		return CalendarDateTime(int64(o) - d), nil
	case "trunc", "round":
		t, err := truncRoundCall(name, c, dateTimeUnits, o.wall())
		if err != nil {
			return Nil, err
		}
		return CalendarDateTimeFromTime(t), nil
	case "addDate":
		years, months, days, err := dateShiftArgs(c)
		if err != nil {
			return Nil, err
		}
		return CalendarDateTimeFromTime(o.wall().AddDate(years, months, days)), nil
	case "date":
		return o.Date(), nil
	case "clock":
		return o.Clock(), nil
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

// calendarDateTimeShiftArg reads a single int|duration argument as a nanosecond
// offset for the .add/.sub methods.
func calendarDateTimeShiftArg(c Call) (int64, error) {
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

// calendarDateTimeLayouts are the zone-less layouts strToCalendarDateTime accepts.
var calendarDateTimeLayouts = []string{
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04",
	"2006-01-02",
	"20060102T15:04:05.999999999",
	"20060102T15:04",
	"20060102",
}

// strToCalendarDateTime parses a zone-less timestamp ("2026-01-31T23:59:55.001",
// "2026-01-31 23:59:55" or "2026-01-31") into a CalendarDateTime. A zone, if
// present, is rejected — CalendarDateTime is wall-clock only.
func strToCalendarDateTime(s string) (CalendarDateTime, error) {
	for _, layout := range calendarDateTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return CalendarDateTimeFromTime(t), nil
		}
	}
	return 0, fmt.Errorf("invalid calendar time %q (want YYYY-MM-DD[ HH:MM:SS], no zone)", s)
}
