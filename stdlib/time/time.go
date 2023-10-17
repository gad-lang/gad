// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package time

import (
	"reflect"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/registry"
	"github.com/gad-lang/gad/token"
)

func init() {
	registry.RegisterObjectConverter(reflect.TypeOf(time.Duration(0)),
		func(in any) (any, bool) {
			return gad.Int(in.(time.Duration)), true
		},
	)

	registry.RegisterObjectConverter(reflect.TypeOf(time.Time{}),
		func(in any) (any, bool) {
			return &Time{Value: in.(time.Time)}, true
		},
	)
	registry.RegisterObjectConverter(reflect.TypeOf((*time.Time)(nil)),
		func(in any) (any, bool) {
			v := in.(*time.Time)
			if v == nil {
				return gad.Nil, true
			}
			return &Time{Value: *v}, true
		},
	)
	registry.RegisterAnyConverter(reflect.TypeOf((*Time)(nil)),
		func(in any) (any, bool) {
			return in.(*Time).Value, true
		},
	)

	registry.RegisterObjectConverter(reflect.TypeOf((*time.Location)(nil)),
		func(in any) (any, bool) {
			v := in.(*time.Location)
			if v == nil {
				return gad.Nil, true
			}
			return &Location{Value: v}, true
		},
	)
	registry.RegisterAnyConverter(reflect.TypeOf((*Location)(nil)),
		func(in any) (any, bool) {
			return in.(*Location).Value, true
		},
	)
}

// gad:doc
// ## Types
// ### time
//
// ToInterface Type
//
// ```go
// // Time represents time values and implements gad.Object interface.
// type Time struct {
//   Value time.Time
// }
// ```

var TimeType = &gad.BuiltinObjType{
	NameValue: "time",
}

// Time represents time values and implements gad.Object interface.
type Time struct {
	Value time.Time
}

var _ gad.NameCallerObject = (*Time)(nil)

func (*Time) Type() gad.ObjectType {
	return TimeType
}

// ToString implements gad.Object interface.
func (o *Time) ToString() string {
	return o.Value.String()
}

// IsFalsy implements gad.Object interface.
func (o *Time) IsFalsy() bool {
	return o.Value.IsZero()
}

// Equal implements gad.Object interface.
func (o *Time) Equal(right gad.Object) bool {
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

// BinaryOp implements gad.Object interface.
func (o *Time) BinaryOp(tok token.Token,
	right gad.Object) (gad.Object, error) {

	switch v := right.(type) {
	case gad.Int:
		switch tok {
		case token.Add:
			return &Time{Value: o.Value.Add(time.Duration(v))}, nil
		case token.Sub:
			return &Time{Value: o.Value.Add(time.Duration(-v))}, nil
		}
	case *Time:
		switch tok {
		case token.Sub:
			return gad.Int(o.Value.Sub(v.Value)), nil
		case token.Less:
			return gad.Bool(o.Value.Before(v.Value)), nil
		case token.LessEq:
			return gad.Bool(o.Value.Before(v.Value) || o.Value.Equal(v.Value)), nil
		case token.Greater:
			return gad.Bool(o.Value.After(v.Value)), nil
		case token.GreaterEq:
			return gad.Bool(o.Value.After(v.Value) || o.Value.Equal(v.Value)),
				nil
		}
	case *gad.NilType:
		switch tok {
		case token.Less, token.LessEq:
			return gad.False, nil
		case token.Greater, token.GreaterEq:
			return gad.True, nil
		}
	}
	return nil, gad.NewOperandTypeError(
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

// IndexGet implements gad.Object interface.
func (o *Time) IndexGet(_ *gad.VM, index gad.Object) (gad.Object, error) {
	v, ok := index.(gad.String)
	if !ok {
		return gad.Nil, gad.NewIndexTypeError("string", index.Type().Name())
	}

	// For simplicity, we use method call for now. As getters are deprecated, we
	// will return callable object in the future here.

	switch v {
	case "Date", "Clock", "UTC", "Unix", "UnixNano", "Year", "Month", "Day",
		"Hour", "Minute", "Second", "Nanosecond", "IsZero", "Local", "Location",
		"YearDay", "Weekday", "ISOWeek", "Zone":
		return o.CallName(string(v), gad.Call{})
	}
	return gad.Nil, nil
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

func (o *Time) CallName(name string, c gad.Call) (gad.Object, error) {
	fn, ok := methodTable[name]
	if !ok {
		return gad.Nil, gad.ErrInvalidIndex.NewError(name)
	}
	return fn(o, &c)
}

var methodTable = map[string]func(*Time, *gad.Call) (gad.Object, error){
	"Add": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		d, ok := gad.ToGoInt64(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		return timeAdd(o, d), nil
	},
	"Sub": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return timeSub(o, t2), nil
	},
	"AddDate": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(3); err != nil {
			return gad.Nil, err
		}
		year, ok := gad.ToGoInt(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		month, ok := gad.ToGoInt(c.Args.Get(1))
		if !ok {
			return newArgTypeErr("2nd", "int", c.Args.Get(1).Type().Name())
		}
		day, ok := gad.ToGoInt(c.Args.Get(2))
		if !ok {
			return newArgTypeErr("3rd", "int", c.Args.Get(2).Type().Name())
		}
		return timeAddDate(o, year, month, day), nil
	},
	"After": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return timeAfter(o, t2), nil
	},
	"Before": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return timeBefore(o, t2), nil
	},
	"Format": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		format, ok := gad.ToGoString(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "string", c.Args.Get(0).Type().Name())
		}
		return timeFormat(o, format), nil
	},
	"AppendFormat": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(2); err != nil {
			return gad.Nil, err
		}
		b, ok := gad.ToGoByteSlice(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "bytes", c.Args.Get(0).Type().Name())
		}
		format, ok := gad.ToGoString(c.Args.Get(1))
		if !ok {
			return newArgTypeErr("2nd", "string", c.Args.Get(1).Type().Name())
		}
		return timeAppendFormat(o, b, format), nil
	},
	"In": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		loc, ok := ToLocation(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "location", c.Args.Get(0).Type().Name())
		}
		return timeIn(o, loc), nil
	},
	"Round": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		d, ok := gad.ToGoInt64(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		return timeRound(o, d), nil
	},
	"Truncate": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		d, ok := gad.ToGoInt64(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
		}
		return timeTruncate(o, d), nil
	},
	"Equal": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return gad.Nil, err
		}
		t2, ok := ToTime(c.Args.Get(0))
		if !ok {
			return newArgTypeErr("1st", "time", c.Args.Get(0).Type().Name())
		}
		return timeEqual(o, t2), nil
	},
	"Date": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		y, m, d := o.Value.Date()
		return gad.Map{"year": gad.Int(y), "month": gad.Int(m),
			"day": gad.Int(d)}, nil
	},
	"Clock": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		h, m, s := o.Value.Clock()
		return gad.Map{"hour": gad.Int(h), "minute": gad.Int(m),
			"second": gad.Int(s)}, nil
	},
	"UTC": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return &Time{Value: o.Value.UTC()}, nil
	},
	"Unix": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Unix()), nil
	},
	"UnixNano": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.UnixNano()), nil
	},
	"Year": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Year()), nil
	},
	"Month": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Month()), nil
	},
	"Day": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Day()), nil
	},
	"Hour": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Hour()), nil
	},
	"Minute": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Minute()), nil
	},
	"Second": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Second()), nil
	},
	"Nanosecond": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Nanosecond()), nil
	},
	"IsZero": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Bool(o.Value.IsZero()), nil
	},
	"Local": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return &Time{Value: o.Value.Local()}, nil
	},
	"Location": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return &Location{Value: o.Value.Location()}, nil
	},
	"YearDay": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.YearDay()), nil
	},
	"Weekday": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		return gad.Int(o.Value.Weekday()), nil
	},
	"ISOWeek": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		y, w := o.Value.ISOWeek()
		return gad.Map{"year": gad.Int(y), "week": gad.Int(w)}, nil
	},
	"Zone": func(o *Time, c *gad.Call) (gad.Object, error) {
		if err := c.Args.CheckLen(0); err != nil {
			return gad.Nil, err
		}
		name, offset := o.Value.Zone()
		return gad.Map{"name": gad.String(name), "offset": gad.Int(offset)}, nil
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

func timeAdd(t *Time, duration int64) gad.Object {
	return &Time{Value: t.Value.Add(time.Duration(duration))}
}

func timeSub(t1, t2 *Time) gad.Object {
	return gad.Int(t1.Value.Sub(t2.Value))
}

func timeAddDate(t *Time, years, months, days int) gad.Object {
	return &Time{Value: t.Value.AddDate(years, months, days)}
}

func timeAfter(t1, t2 *Time) gad.Object {
	return gad.Bool(t1.Value.After(t2.Value))
}

func timeBefore(t1, t2 *Time) gad.Object {
	return gad.Bool(t1.Value.Before(t2.Value))
}

func timeFormat(t *Time, layout string) gad.Object {
	return gad.String(t.Value.Format(layout))
}

func timeAppendFormat(t *Time, b []byte, layout string) gad.Object {
	return gad.Bytes(t.Value.AppendFormat(b, layout))
}

func timeIn(t *Time, loc *Location) gad.Object {
	return &Time{Value: t.Value.In(loc.Value)}
}

func timeRound(t *Time, duration int64) gad.Object {
	return &Time{Value: t.Value.Round(time.Duration(duration))}
}

func timeTruncate(t *Time, duration int64) gad.Object {
	return &Time{Value: t.Value.Truncate(time.Duration(duration))}
}

func timeEqual(t1, t2 *Time) gad.Object {
	return gad.Bool(t1.Value.Equal(t2.Value))
}

func newArgTypeErr(pos, want, got string) (gad.Object, error) {
	return gad.Nil, gad.NewArgumentTypeError(pos, want, got)
}
