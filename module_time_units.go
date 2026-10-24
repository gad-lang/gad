package gad

import (
	"fmt"
	"strings"
	"time"
)

// The units of `trunc` and `round` (issue #26): named parameters, the largest
// first; the largest one that is not falsy is the unit, its value how many of
// it (a flag is one): `x.trunc(; m=3)` (the quarter), `x.trunc(; d=byDay, h)`.
const (
	unitYear        = "y"
	unitMonth       = "m"
	unitWeek        = "w"
	unitDay         = "d"
	unitHour        = "h"
	unitMinute      = "min"
	unitSecond      = "s"
	unitMillisecond = "ms"
	unitMicrosecond = "us"
	unitNanosecond  = "ns"
)

// timeUnitsOrder are the units, the largest first.
var timeUnitsOrder = []string{unitYear, unitMonth, unitWeek, unitDay, unitHour,
	unitMinute, unitSecond, unitMillisecond, unitMicrosecond, unitNanosecond}

// The units each type takes.
var (
	calendarDateUnits = []string{unitYear, unitMonth, unitWeek, unitDay}
	timeOfDayUnits    = []string{unitHour, unitMinute, unitSecond, unitMillisecond, unitMicrosecond, unitNanosecond}
	dateTimeUnits     = timeUnitsOrder
	durationUnits     = []string{unitWeek, unitDay, unitHour, unitMinute, unitSecond, unitMillisecond, unitMicrosecond, unitNanosecond}
)

// clockUnitDuration is the length of the clock units.
var clockUnitDuration = map[string]time.Duration{
	unitDay:         24 * time.Hour,
	unitHour:        time.Hour,
	unitMinute:      time.Minute,
	unitSecond:      time.Second,
	unitMillisecond: time.Millisecond,
	unitMicrosecond: time.Microsecond,
	unitNanosecond:  time.Nanosecond,
}

// The enums of the counts that have a name (the `time` module's):
// MonthSpans for m, HourSpans for h, MinuteSpans for min; w takes a member of
// Weekdays, the day the week starts on.
var (
	TimeMonthSpansEnum  = newTimeSpansEnum("MonthSpans", "Bimester", 2, "Quarter", 3, "Semester", 6)
	TimeHourSpansEnum   = newTimeSpansEnum("HourSpans", "QuarterDay", 6, "HalfDay", 12)
	TimeMinuteSpansEnum = newTimeSpansEnum("MinuteSpans", "QuarterHour", 15, "HalfHour", 30)
	TimeWeekdaysEnum    = newTimeWeekdaysEnum()

	timeUnitEnums = map[string]*Enum{
		unitMonth:  TimeMonthSpansEnum,
		unitHour:   TimeHourSpansEnum,
		unitMinute: TimeMinuteSpansEnum,
	}
)

func newTimeSpansEnum(name string, members ...any) *Enum {
	e := NewEnum(name, TimeModuleSpec)
	for i := 0; i < len(members); i += 2 {
		e.AddValue(members[i].(string), Int(members[i+1].(int)))
	}
	return e
}

// timeUnits is the unit a trunc/round call names: n of it; for a week, the
// day it starts on.
type timeUnits struct {
	unit      string
	n         int
	weekStart time.Weekday
}

// timeUnitsArg reads the units of a trunc/round call: no positional argument,
// named ones among allowed; none is the second (a day for a cdate).
func timeUnitsArg(name string, c Call, allowed []string) (u timeUnits, err error) {
	if c.Args.Length() > 0 {
		return u, ErrType.NewError(name + ": the unit is a named argument: (; y), (; m=3), (; min=15), …")
	}
	named := c.NamedArgs.Dict()
	for k := range named {
		if !containsStr(timeUnitsOrder, k) {
			return u, ErrUnexpectedNamedArg.NewError(fmt.Sprintf("%s: %s (the units are %s)", name, k,
				strings.Join(timeUnitsOrder, ", ")))
		}
		if !containsStr(allowed, k) {
			return u, ErrUnexpectedNamedArg.NewError(fmt.Sprintf("%s: %s is not a unit of this type (%s)", name, k,
				strings.Join(allowed, ", ")))
		}
	}
	u.weekStart = time.Monday
	for _, k := range timeUnitsOrder {
		v, ok := named[k]
		if _, member := v.(*EnumValue); !ok || !member && v.IsFalsy() {
			// a member of an enum is never off: time.Weekdays.Sunday is 0
			continue
		}
		u.unit, u.n = k, 1
		switch t := v.(type) {
		case Flag, Bool:
		case Int:
			// a count of weeks only on a duration (a date's week starts on a
			// weekday: w=time.Weekdays.Sunday)
			if t < 1 || k == unitWeek && t != 1 && allowed[0] != unitWeek {
				return u, ErrType.NewError(fmt.Sprintf("%s: %s=%d (a positive count)", name, k, t))
			}
			u.n = int(t)
		case *EnumValue:
			if k == unitWeek && t.Enum == TimeWeekdaysEnum {
				u.weekStart = time.Weekday(t.Value.(Int))
				break
			}
			if e := timeUnitEnums[k]; e == nil || t.Enum != e {
				return u, ErrType.NewError(fmt.Sprintf("%s: %s=%s is not a member of %s", name, k, t.Name, unitEnumName(k)))
			}
			u.n = int(t.Value.(Int))
		default:
			return u, ErrType.NewError(fmt.Sprintf("%s: %s=%s (a positive count, yes or %s)", name, k,
				v.Type().Name(), unitEnumName(k)))
		}
		return u, nil
	}
	// none: the second (a cdate, the day)
	if containsStr(allowed, unitSecond) {
		u.unit, u.n = unitSecond, 1
	} else {
		u.unit, u.n = unitDay, 1
	}
	return u, nil
}

func unitEnumName(unit string) string {
	if unit == unitWeek {
		return "time.Weekdays"
	}
	if e := timeUnitEnums[unit]; e != nil {
		return "time." + e.Name()
	}
	return "an int"
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// trunc is t at the start of its unit, n of them counted from the start of
// the next larger unit (`m=3` from January, `min=15` from the hour); in t's
// location.
func (u timeUnits) trunc(t time.Time) time.Time {
	y, mo, d := t.Date()
	loc := t.Location()
	h, mi, s := t.Clock()
	ns := t.Nanosecond()
	n := u.n
	switch u.unit {
	case unitYear:
		return time.Date(y-y%n, 1, 1, 0, 0, 0, 0, loc)
	case unitMonth:
		m := int(mo) - 1
		return time.Date(y, time.Month(m-m%n+1), 1, 0, 0, 0, 0, loc)
	case unitWeek:
		off := (int(t.Weekday()) - int(u.weekStart) + 7) % 7
		return time.Date(y, mo, d-off, 0, 0, 0, 0, loc)
	case unitDay:
		return time.Date(y, mo, d-(d-1)%n, 0, 0, 0, 0, loc)
	case unitHour:
		return time.Date(y, mo, d, h-h%n, 0, 0, 0, loc)
	case unitMinute:
		return time.Date(y, mo, d, h, mi-mi%n, 0, 0, loc)
	case unitSecond:
		return time.Date(y, mo, d, h, mi, s-s%n, 0, loc)
	}
	step := int(clockUnitDuration[u.unit]) * n
	return time.Date(y, mo, d, h, mi, s, ns-ns%step, loc)
}

// next is the start of the unit after floor (the output of trunc).
func (u timeUnits) next(floor time.Time) time.Time {
	switch u.unit {
	case unitYear:
		return floor.AddDate(u.n, 0, 0)
	case unitMonth:
		return floor.AddDate(0, u.n, 0)
	case unitWeek:
		return floor.AddDate(0, 0, 7)
	case unitDay:
		return floor.AddDate(0, 0, u.n)
	}
	return floor.Add(clockUnitDuration[u.unit] * time.Duration(u.n))
}

// round is t at the nearest start of its unit (a tie rounds up).
func (u timeUnits) round(t time.Time) time.Time {
	floor := u.trunc(t)
	next := u.next(floor)
	if t.Sub(floor) < next.Sub(t) {
		return floor
	}
	return next
}

// duration is the length of the unit, for a duration (no year nor month).
func (u timeUnits) duration() time.Duration {
	if u.unit == unitWeek {
		return 7 * 24 * time.Hour * time.Duration(u.n)
	}
	return clockUnitDuration[u.unit] * time.Duration(u.n)
}

// truncRoundCall is the trunc or round method (name) of a type whose values
// are a time.Time in some form.
func truncRoundCall(name string, c Call, allowed []string, t time.Time) (time.Time, error) {
	u, err := timeUnitsArg(name, c, allowed)
	if err != nil {
		return t, err
	}
	if name == "round" {
		return u.round(t), nil
	}
	return u.trunc(t), nil
}

// dateShiftArgs reads the (years, months, days) int arguments shared by the
// .addDate methods of the calendar value types.
func dateShiftArgs(c Call) (years, months, days int, err error) {
	y := &Arg{Name: "years", TypeAssertion: TypeAssertionFromTypes(TInt)}
	m := &Arg{Name: "months", TypeAssertion: TypeAssertionFromTypes(TInt)}
	d := &Arg{Name: "days", TypeAssertion: TypeAssertionFromTypes(TInt)}
	if err = c.Args.Destructure(y, m, d); err != nil {
		return
	}
	return int(y.Value.(Int)), int(m.Value.(Int)), int(d.Value.(Int)), nil
}

// timeLayoutArg reads a single Go layout string argument for the .format
// methods of the calendar value types.
func timeLayoutArg(c Call) (string, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return "", err
	}
	switch v := c.Args.Get(0).(type) {
	case Str:
		return string(v), nil
	case RawStr:
		return string(v), nil
	}
	return "", NewArgumentTypeError("1st", "str", c.Args.Get(0).Type().Name())
}
