package gad

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gad-lang/gad/token"
)

// day is the length of a TimeOfDay's range: a time of day is below it.
const day = 24 * time.Hour

// TimeOfDayType is the object type of TimeOfDay values (the gad `ctime`
// type). It is callable as a constructor: ctime(h, m[, s]; ns=…),
// ctime(duration), ctime(str), ctime(cdtime|time) (its clock).
var TimeOfDayType = NewBuiltinObjType("ctime").WithNew(NewTimeOfDayFunc)

// TimeOfDay is a zone-less time of day, from 00:00:00 to 23:59:59.999999999,
// stored as the nanoseconds since midnight. Its arithmetic is a clock's: a
// duration that passes midnight wraps around (`23:00T + dur 2h` is `01:00T`).
type TimeOfDay uint64

var _ NameCallerObject = TimeOfDay(0)

// NewTimeOfDay is the time of day h:m:s.nsec; ok is false when a field is out
// of its range.
func NewTimeOfDay(h, m, s, nsec int) (_ TimeOfDay, ok bool) {
	if h < 0 || h > 23 || m < 0 || m > 59 || s < 0 || s > 59 || nsec < 0 || nsec > 999999999 {
		return 0, false
	}
	return TimeOfDay(time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(s)*time.Second + time.Duration(nsec)), true
}

// TimeOfDayFromDuration is d, wrapped into a day (a negative d counts back
// from midnight).
func TimeOfDayFromDuration(d time.Duration) TimeOfDay {
	d %= day
	if d < 0 {
		d += day
	}
	return TimeOfDay(d)
}

// TimeOfDayFromTime is the clock of t (its wall clock in its zone).
func TimeOfDayFromTime(t time.Time) TimeOfDay {
	h, m, s := t.Clock()
	v, _ := NewTimeOfDay(h, m, s, t.Nanosecond())
	return v
}

// NewTimeOfDayFunc is the ctime(...) constructor.
func NewTimeOfDayFunc(c Call) (Object, error) {
	if c.Args.Length() == 1 {
		switch v := c.Args.Get(0).(type) {
		case TimeOfDay:
			return v, nil
		case Duration:
			if v < 0 || time.Duration(v) >= day {
				return Nil, ErrType.NewError(fmt.Sprintf("ctime: %s is not within a day", time.Duration(v)))
			}
			return TimeOfDay(v), nil
		case CalendarDateTime:
			return v.Clock(), nil
		case *Time:
			return TimeOfDayFromTime(v.Value), nil
		case Str, RawStr:
			t, err := strToTimeOfDay(v.ToString())
			if err != nil {
				return Nil, ErrType.NewError(err.Error())
			}
			return t, nil
		}
	}
	var (
		h  = &Arg{Name: "hour", TypeAssertion: TypeAssertionFromTypes(TInt)}
		m  = &Arg{Name: "minute", TypeAssertion: TypeAssertionFromTypes(TInt)}
		s  = &Arg{Name: "second", TypeAssertion: TypeAssertionFromTypes(TInt)}
		ns = &NamedArgVar{Name: "ns", Value: Int(0), TypeAssertion: TypeAssertionFromTypes(TInt)}
	)
	args := []*Arg{h, m}
	if c.Args.Length() == 3 {
		args = append(args, s)
	} else {
		s.Value = Int(0)
	}
	if err := c.Args.Destructure(args...); err != nil {
		return Nil, err
	}
	if err := c.NamedArgs.Get(ns); err != nil {
		return Nil, err
	}
	t, ok := NewTimeOfDay(int(h.Value.(Int)), int(m.Value.(Int)), int(s.Value.(Int)), int(ns.Value.(Int)))
	if !ok {
		return Nil, ErrType.NewError(fmt.Sprintf("ctime: %v:%v:%v.%v is not a time of day",
			h.Value, m.Value, s.Value, ns.Value))
	}
	return t, nil
}

func (TimeOfDay) Type() ObjectType { return TimeOfDayType }

func (o TimeOfDay) Hour() int       { return int(time.Duration(o) / time.Hour) }
func (o TimeOfDay) Minute() int     { return int(time.Duration(o) % time.Hour / time.Minute) }
func (o TimeOfDay) Second() int     { return int(time.Duration(o) % time.Minute / time.Second) }
func (o TimeOfDay) Nanosecond() int { return int(time.Duration(o) % time.Second) }

// Duration is the time since midnight.
func (o TimeOfDay) Duration() time.Duration { return time.Duration(o) }

// fraction is the nanoseconds as a decimal fraction without its trailing
// zeros (`.25`), "" when there are none.
func (o TimeOfDay) fraction() string {
	ns := o.Nanosecond()
	if ns == 0 {
		return ""
	}
	return "." + strings.TrimRight(fmt.Sprintf("%09d", ns), "0")
}

// ToString renders the time of day as "HH:MM:SS[.frac]".
func (o TimeOfDay) ToString() string {
	return fmt.Sprintf("%02d:%02d:%02d", o.Hour(), o.Minute(), o.Second()) + o.fraction()
}

// literal is its gad literal: `10:30T`, `10:30:15T`, `10:30:15.25T`.
func (o TimeOfDay) literal() string {
	s := fmt.Sprintf("%02d:%02d", o.Hour(), o.Minute())
	if o.Second() != 0 || o.Nanosecond() != 0 {
		s += fmt.Sprintf(":%02d", o.Second()) + o.fraction()
	}
	return s + "T"
}

// Print writes the time of day: its text `10:30:15.25`, its literal
// `10:30:15.25T`.
func (o TimeOfDay) Print(s *PrinterState) error {
	return s.literalOr(o, o.literal, o.ToString())
}

// MarshalJSON encodes the time of day as the JSON string "HH:MM:SS[.frac]".
func (o TimeOfDay) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(o.ToString())), nil
}

// IsFalsy reports whether the time of day is midnight.
func (o TimeOfDay) IsFalsy() bool { return o == 0 }

func (o TimeOfDay) Equal(right Object) bool {
	v, ok := right.(TimeOfDay)
	return ok && o == v
}

// add is o shifted by d, around the clock.
func (o TimeOfDay) add(d time.Duration) TimeOfDay {
	return TimeOfDayFromDuration(time.Duration(o) + d)
}

// BinaryOp supports clock arithmetic and ordered comparisons:
//   - `ctime ± duration` -> ctime (wraps around midnight)
//   - `ctime - ctime` -> duration
//   - `ctime + cdate` -> cdtime (as `cdate + ctime`)
func (o TimeOfDay) BinOpAdd(_ *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		return o.add(time.Duration(v)), nil
	case CalendarDate:
		return v.At(o), nil
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o TimeOfDay) BinOpSub(_ *VM, right Object) (Object, error) {
	switch v := right.(type) {
	case Duration:
		return o.add(-time.Duration(v)), nil
	case TimeOfDay:
		return Duration(time.Duration(o) - time.Duration(v)), nil
	}
	return nil, NewOperandTypeError(token.Sub.String(), o.Type().Name(), right.Type().Name())
}

func (o TimeOfDay) cmp(tok token.Token, right Object) (Object, error) {
	v, ok := right.(TimeOfDay)
	if !ok {
		return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
	}
	switch tok {
	case token.Less:
		return Bool(o < v), nil
	case token.LessEq:
		return Bool(o <= v), nil
	case token.Greater:
		return Bool(o > v), nil
	}
	return Bool(o >= v), nil
}

func (o TimeOfDay) BinOpLess(_ *VM, right Object) (Object, error) { return o.cmp(token.Less, right) }
func (o TimeOfDay) BinOpLessEq(_ *VM, right Object) (Object, error) {
	return o.cmp(token.LessEq, right)
}
func (o TimeOfDay) BinOpGreater(_ *VM, right Object) (Object, error) {
	return o.cmp(token.Greater, right)
}
func (o TimeOfDay) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	return o.cmp(token.GreaterEq, right)
}

// UnOpInc steps the clock by its least significant non-zero unit (at midnight,
// an hour).
func (o TimeOfDay) UnOpInc(*VM) (Object, error) { return o.add(o.step()), nil }

// UnOpDec steps the clock back by its least significant non-zero unit.
func (o TimeOfDay) UnOpDec(*VM) (Object, error) { return o.add(-o.step()), nil }

func (o TimeOfDay) step() time.Duration {
	s := leastTimeStep(o.Hour(), o.Minute(), o.Second(), o.Nanosecond())
	if s == day {
		return time.Hour
	}
	return s
}

// CallName dispatches the time-of-day methods.
func (o TimeOfDay) CallName(name string, c Call) (Object, error) {
	switch name {
	case "hour":
		return Int(o.Hour()), nil
	case "minute":
		return Int(o.Minute()), nil
	case "second":
		return Int(o.Second()), nil
	case "ms":
		return Int(o.Nanosecond() / 1e6), nil
	case "us":
		return Int(o.Nanosecond() / 1e3), nil
	case "ns":
		return Int(o.Nanosecond()), nil
	case "seconds":
		return Float(o.Duration().Seconds()), nil
	case "duration":
		return Duration(o), nil
	case "format":
		layout, err := timeLayoutArg(c)
		if err != nil {
			return Nil, err
		}
		return Str(time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC).Add(o.Duration()).Format(layout)), nil
	case "add":
		d, err := calendarDateTimeShiftArg(c)
		if err != nil {
			return Nil, err
		}
		return o.add(time.Duration(d)), nil
	case "sub":
		d, err := calendarDateTimeShiftArg(c)
		if err != nil {
			return Nil, err
		}
		return o.add(-time.Duration(d)), nil
	case "trunc", "round":
		units, err := timeUnitsArg(name, c, timeOfDayUnits)
		if err != nil {
			return Nil, err
		}
		t := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Add(o.Duration())
		if name == "trunc" {
			t = units.trunc(t)
		} else {
			t = units.round(t)
		}
		if t.Day() != 1 {
			// rounded up to the next midnight
			return TimeOfDay(0), nil
		}
		return TimeOfDayFromTime(t), nil
	}
	return Nil, ErrInvalidIndex.NewError(name)
}

// strToTimeOfDay parses "HH:MM[:SS[.frac]]".
func strToTimeOfDay(s string) (TimeOfDay, error) {
	bad := func() (TimeOfDay, error) {
		return 0, fmt.Errorf("invalid time of day %q (want HH:MM[:SS[.frac]], 00:00 to 23:59:59.999999999)", s)
	}
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return bad()
	}
	num := func(p string) (int, bool) {
		if len(p) != 2 {
			return 0, false
		}
		n, err := strconv.Atoi(p)
		return n, err == nil
	}
	h, ok1 := num(parts[0])
	m, ok2 := num(parts[1])
	if !ok1 || !ok2 {
		return bad()
	}
	var sec, nsec int
	if len(parts) == 3 {
		secs, frac, hasFrac := strings.Cut(parts[2], ".")
		var ok bool
		if sec, ok = num(secs); !ok {
			return bad()
		}
		if hasFrac {
			if len(frac) == 0 || len(frac) > 9 {
				return bad()
			}
			n, err := strconv.Atoi(frac + strings.Repeat("0", 9-len(frac)))
			if err != nil {
				return bad()
			}
			nsec = n
		}
	}
	t, ok := NewTimeOfDay(h, m, sec, nsec)
	if !ok {
		return bad()
	}
	return t, nil
}
