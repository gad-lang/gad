// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"time"

	"github.com/gad-lang/gad/token"
)

// gad:doc
// ## Types
// ### time
//
// ToInterface Type
//
// ```go
// // Time represents time values and implements Object interface.
// type Time struct {
//   Value time.Time
// }
// ```

var TimeType = NewBuiltinObjType("time").WithNew(timeNew)

// Time represents time values and implements Object interface.
type Time struct {
	Value time.Time
}

var _ NameCallerObject = (*Time)(nil)

func (*Time) Type() ObjectType {
	return TimeType
}

// ToString implements Object interface.
func (o *Time) ToString() string {
	return o.Value.String()
}

// Print writes the time as a leaf value. *Time is a primitive (see
// IsPrimitive), so it must print as its ToString rather than letting the
// generic printer recurse into the wrapped time.Time internals.
func (o *Time) Print(s *PrinterState) error {
	if s.IsRepr {
		defer s.WrapRepr(o)()
	}
	return s.WriteString(o.ToString())
}

// IsFalsy implements Object interface.
func (o *Time) IsFalsy() bool {
	return o.Value.IsZero()
}

// Equal implements Object interface.
func (o *Time) Equal(right Object) bool {
	if v, ok := right.(*Time); ok {
		return o.Value.Equal(v.Value)
	}
	return false
}

// gad:doc
// #### Overloaded time Operators
//
// - `time + int` -> time
// - `time - int` -> time
// - `time - time` -> int
// - `time < time` -> bool
// - `time > time` -> bool
// - `time <= time` -> bool
// - `time >= time` -> bool
//
// Note that, `int` values as duration must be the right hand side operand.

// BinaryOp implements Object interface.
func (o *Time) BinaryOp(_ *VM, tok token.Token,
	right Object) (Object, error) {

	switch v := right.(type) {
	case Int:
		switch tok {
		case token.Add:
			return &Time{Value: o.Value.Add(time.Duration(v))}, nil
		case token.Sub:
			return &Time{Value: o.Value.Add(time.Duration(-v))}, nil
		}
	case Duration:
		switch tok {
		case token.Add:
			return &Time{Value: o.Value.Add(time.Duration(v))}, nil
		case token.Sub:
			return &Time{Value: o.Value.Add(-time.Duration(v))}, nil
		}
	case *Time:
		switch tok {
		case token.Sub:
			return Int(o.Value.Sub(v.Value)), nil
		case token.Less:
			return Bool(o.Value.Before(v.Value)), nil
		case token.LessEq:
			return Bool(o.Value.Before(v.Value) || o.Value.Equal(v.Value)), nil
		case token.Greater:
			return Bool(o.Value.After(v.Value)), nil
		case token.GreaterEq:
			return Bool(o.Value.After(v.Value) || o.Value.Equal(v.Value)),
				nil
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
		right.Type().Name())
}

// gad:doc
// #### time Getters
//
// Deprecated: Use method call. These selectors will return a callable object in
// the future. See methods.
//
// Dynamically calculated getters for a time value are as follows:
//
// | Selector  | Return Type                                     |
// |:----------|:------------------------------------------------|
// |.Date      | {"year": int, "month": int, "day": int}         |
// |.Clock     | {"hour": int, "minute": int, "second": int}     |
// |.UTC       | time                                            |
// |.Unix      | int                                             |
// |.UnixNano  | int                                             |
// |.Year      | int                                             |
// |.Month     | int                                             |
// |.Day       | int                                             |
// |.Hour      | int                                             |
// |.Minute    | int                                             |
// |.Second    | int                                             |
// |.NanoSecond| int                                             |
// |.IsZero    | bool                                            |
// |.Local     | time                                            |
// |.Location  | location                                        |
// |.YearDay   | int                                             |
// |.Weekday   | int                                             |
// |.ISOWeek   | {"year": int, "week": int}                      |
// |.Zone      | {"name": string, "offset": int}                 |

// IndexGet implements Object interface.
func (o *Time) IndexGet(_ *VM, index Object) (Object, error) {
	v, ok := index.(Str)
	if !ok {
		return Nil, NewIndexTypeError("str", index.Type().Name())
	}

	// For simplicity, we use method call for now. As getters are deprecated, we
	// will return callable object in the future here.

	switch v {
	case "Date", "Clock", "UTC", "Unix", "UnixNano", "Year", "Month", "Day",
		"Hour", "Minute", "Second", "Nanosecond", "IsZero", "Local", "Location",
		"YearDay", "Weekday", "ISOWeek", "Zone":
		return o.CallName(string(v), Call{})
	}
	return Nil, nil
}

// gad:doc
// #### time Methods
//
// | Method                               | Return Type                                 |
// |:-------------------------------------|:--------------------------------------------|
// |.Add(duration int)                    | time                                        |
// |.Sub(t2 time)                         | int                                         |
// |.AddDate(year int, month int, day int)| int                                         |
// |.After(t2 time)                       | bool                                        |
// |.Before(t2 time)                      | bool                                        |
// |.Format(layout string)                | string                                      |
// |.AppendFormat(b bytes, layout string) | bytes                                       |
// |.In(loc location)                     | time                                        |
// |.Round(duration int)                  | time                                        |
// |.Truncate(duration int)               | time                                        |
// |.Equal(t2 time)                       | bool                                        |
// |.Date()                               | {"year": int, "month": int, "day": int}     |
// |.Clock()                              | {"hour": int, "minute": int, "second": int} |
// |.UTC()                                | time                                        |
// |.Unix()                               | int                                         |
// |.UnixNano()                           | int                                         |
// |.Year()                               | int                                         |
// |.Month()                              | int                                         |
// |.Day()                                | int                                         |
// |.Hour()                               | int                                         |
// |.Minute()                             | int                                         |
// |.Second()                             | int                                         |
// |.NanoSecond()                         | int                                         |
// |.IsZero()                             | bool                                        |
// |.Local()                              | time                                        |
// |.Location()                           | location                                    |
// |.YearDay()                            | int                                         |
// |.Weekday()                            | int                                         |
// |.ISOWeek()                            | {"year": int, "week": int}                  |
// |.Zone()                               | {"name": string, "offset": int}             |

func (o *Time) CallName(name string, c Call) (Object, error) {
	fn, ok := MethodTable[name]
	if !ok {
		// gad methods are camelCase; the table is keyed by the Go (PascalCase)
		// names, so capitalise the first letter and retry.
		fn, ok = MethodTable[capitalizeFirst(name)]
	}
	if !ok {
		return Nil, ErrInvalidIndex.NewError(name)
	}
	return fn(o, &c)
}

// capitalizeFirst upper-cases the first ASCII letter of s.
func capitalizeFirst(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-'a'+'A') + s[1:]
}

var MethodTable = map[string]func(*Time, *Call) (Object, error){
	"Add": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		d, ok := ToGoInt64(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		return TimeAdd(o, d), nil
	},
	"Sub": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return TimeSub(o, t2), nil
	},
	"AddDate": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(3); err != nil {
			return Nil, err
		}
		year, ok := ToGoInt(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		month, ok := ToGoInt(c.Args.Get(1))
		if !ok {
			return TimeNewArgTypeErr("2nd", "int", c.Args.Get(1).Type().Name())
		}
		day, ok := ToGoInt(c.Args.Get(2))
		if !ok {
			return TimeNewArgTypeErr("3rd", "int", c.Args.Get(2).Type().Name())
		}
		return TimeAddDate(o, year, month, day), nil
	},
	"After": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return TimeAfter(o, t2), nil
	},
	"Before": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return TimeBefore(o, t2), nil
	},
	"Format": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		format, ok := ToGoString(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "str", c.Args.Get(0).Type().Name())
		}
		return TimeFormat(o, format), nil
	},
	"AppendFormat": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(2); err != nil {
			return Nil, err
		}
		b, ok := ToGoByteSlice(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "bytes", c.Args.Get(0).Type().Name())
		}
		format, ok := ToGoString(c.Args.Get(1))
		if !ok {
			return TimeNewArgTypeErr("2nd", "str", c.Args.Get(1).Type().Name())
		}
		return TimeAppendFormat(o, b, format), nil
	},
	"In": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		loc, ok := ToLocation(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "Location", c.Args.Get(0).Type().Name())
		}
		return TimeIn(o, loc), nil
	},
	"Round": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		d, ok := ToGoInt64(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		return TimeRound(o, d), nil
	},
	"Truncate": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		d, ok := ToGoInt64(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		return TimeTruncate(o, d), nil
	},
	// trunc(unit char) lower-truncates to the start of a calendar unit
	// (y, M, w, d, h, m, s, ms, us, ns).
	"trunc": func(o *Time, c *Call) (Object, error) {
		unit, err := truncateUnitArg(*c)
		if err != nil {
			return Nil, err
		}
		t, err := truncateTimeUnit(o.Value, unit)
		if err != nil {
			return Nil, err
		}
		return &Time{Value: t}, nil
	},
	// round(unit char) rounds to the nearest unit boundary (a tie rounds up).
	"round": func(o *Time, c *Call) (Object, error) {
		unit, err := truncateUnitArg(*c)
		if err != nil {
			return Nil, err
		}
		t, err := roundTimeUnit(o.Value, unit)
		if err != nil {
			return Nil, err
		}
		return &Time{Value: t}, nil
	},
	"Equal": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return TimeNewArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return TimeEqual(o, t2), nil
	},
	"Date": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		y, m, d := o.Value.Date()
		return Dict{"year": Int(y), "month": Int(m),
			"day": Int(d)}, nil
	},
	"Clock": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		h, m, s := o.Value.Clock()
		return Dict{"hour": Int(h), "minute": Int(m),
			"second": Int(s)}, nil
	},
	"UTC": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return &Time{Value: o.Value.UTC()}, nil
	},
	"Unix": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Unix()), nil
	},
	"UnixNano": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.UnixNano()), nil
	},
	"Year": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Year()), nil
	},
	"Month": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Month()), nil
	},
	"Day": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Day()), nil
	},
	"Hour": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Hour()), nil
	},
	"Minute": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Minute()), nil
	},
	"Second": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Second()), nil
	},
	"Nanosecond": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Nanosecond()), nil
	},
	// ns is the camelCase short alias of Nanosecond.
	"ns": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Nanosecond()), nil
	},
	"IsZero": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Bool(o.Value.IsZero()), nil
	},
	"Local": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return &Time{Value: o.Value.Local()}, nil
	},
	"Location": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return &Location{Value: o.Value.Location()}, nil
	},
	"YearDay": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.YearDay()), nil
	},
	"Weekday": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		return Int(o.Value.Weekday()), nil
	},
	"ISOWeek": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		y, w := o.Value.ISOWeek()
		return Dict{"year": Int(y), "week": Int(w)}, nil
	},
	"Zone": func(o *Time, c *Call) (Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return Nil, err
		}
		name, offset := o.Value.Zone()
		return Dict{"name": Str(name), "offset": Int(offset)}, nil
	},
}

// MarshalBinary implements encoding.BinaryMarshaler interface.
func (o *Time) MarshalBinary() ([]byte, error) {
	return o.Value.MarshalBinary()
}

// MarshalJSON implements json.JSONMarshaler interface.
func (o *Time) MarshalJSON() ([]byte, error) {
	return o.Value.MarshalJSON()
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler interface.
func (o *Time) UnmarshalBinary(data []byte) error {
	var t time.Time
	if err := t.UnmarshalBinary(data); err != nil {
		return err
	}
	o.Value = t
	return nil
}

// UnmarshalJSON implements json.JSONUnmarshaler interface.
func (o *Time) UnmarshalJSON(data []byte) error {
	var t time.Time
	if err := t.UnmarshalJSON(data); err != nil {
		return err
	}
	o.Value = t
	return nil
}

func TimeAdd(t *Time, duration int64) Object {
	return &Time{Value: t.Value.Add(time.Duration(duration))}
}

func TimeSub(t1, t2 *Time) Object {
	return Int(t1.Value.Sub(t2.Value))
}

func TimeAddDate(t *Time, years, months, days int) Object {
	return &Time{Value: t.Value.AddDate(years, months, days)}
}

func TimeAfter(t1, t2 *Time) Object {
	return Bool(t1.Value.After(t2.Value))
}

func TimeBefore(t1, t2 *Time) Object {
	return Bool(t1.Value.Before(t2.Value))
}

func TimeFormat(t *Time, layout string) Object {
	return Str(t.Value.Format(layout))
}

func TimeAppendFormat(t *Time, b []byte, layout string) Object {
	return Bytes(t.Value.AppendFormat(b, layout))
}

func TimeIn(t *Time, loc *Location) Object {
	return &Time{Value: t.Value.In(loc.Value)}
}

func TimeRound(t *Time, duration int64) Object {
	return &Time{Value: t.Value.Round(time.Duration(duration))}
}

func TimeTruncate(t *Time, duration int64) Object {
	return &Time{Value: t.Value.Truncate(time.Duration(duration))}
}

func TimeEqual(t1, t2 *Time) Object {
	return Bool(t1.Value.Equal(t2.Value))
}

func TimeNewArgTypeErr(pos, want, got string) (Object, error) {
	return Nil, NewArgumentTypeError(pos, want, got)
}
