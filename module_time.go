// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Package time provides time module for measuring and displaying time for Gad
// script language. It wraps ToInterface's time package functionalities.
// Note that: Gad's int values are converted to ToInterface's time.Duration values.
package gad

import (
	"reflect"
	"strconv"
	"time"

	"github.com/gad-lang/gad/registry"
)

// TimeModuleSpec is the module spec shared by the builtin `time` namespace
// members and the importable time module.
var TimeModuleSpec = NewModuleSpecFromName("time")

var TimeUtcLoc Object = &Location{Value: time.UTC}
var TimeLocalLoc Object = &Location{Value: time.Local}
var TimeZeroTime Object = &Time{}

// TimeModule returns the `time` builtin namespace. It is also used by the stdlib
// `time` importable module.
func TimeModule() Dict { return newTimeModule() }

// newTimeModule builds the `time` builtin namespace (Go's time).
func newTimeModule() Dict {
	return Dict{
		// gad:doc
		// # time module
		// ## Types
		// Type is a type of Time Value
		"Type": TimeType,
		// CalendarDate is the `calendarDate` value type (YYYYMMDD); a constructor.
		"CalendarDate": CalendarDateType,
		// Duration is the `duration` value type; callable as a constructor.
		"Duration": DurationType,
		// Location is the `Location` value type.
		"Location": TimeLocationType,

		//
		// ## Constants
		// ### Months
		//
		// January
		// February
		// March
		// April
		// May
		// June
		// July
		// August
		// September
		// October
		// November
		// December
		"January":   Int(time.January),
		"February":  Int(time.February),
		"March":     Int(time.March),
		"April":     Int(time.April),
		"May":       Int(time.May),
		"June":      Int(time.June),
		"July":      Int(time.July),
		"August":    Int(time.August),
		"September": Int(time.September),
		"October":   Int(time.October),
		"November":  Int(time.November),
		"December":  Int(time.December),

		// gad:doc
		// ### Weekdays
		//
		// Sunday
		// Monday
		// Tuesday
		// Wednesday
		// Thursday
		// Friday
		// Saturday
		"Sunday":    Int(time.Sunday),
		"Monday":    Int(time.Monday),
		"Tuesday":   Int(time.Tuesday),
		"Wednesday": Int(time.Wednesday),
		"Thursday":  Int(time.Thursday),
		"Friday":    Int(time.Friday),
		"Saturday":  Int(time.Saturday),

		// gad:doc
		// ### Layouts
		//
		// ANSIC
		// UnixDate
		// RubyDate
		// RFC822
		// RFC822Z
		// RFC850
		// RFC1123
		// RFC1123Z
		// RFC3339
		// RFC3339Nano
		// Kitchen
		// Stamp
		// StampMilli
		// StampMicro
		// StampNano
		"ANSIC":       Str(time.ANSIC),
		"UnixDate":    Str(time.UnixDate),
		"RubyDate":    Str(time.RubyDate),
		"RFC822":      Str(time.RFC822),
		"RFC822Z":     Str(time.RFC822Z),
		"RFC850":      Str(time.RFC850),
		"RFC1123":     Str(time.RFC1123),
		"RFC1123Z":    Str(time.RFC1123Z),
		"RFC3339":     Str(time.RFC3339),
		"RFC3339Nano": Str(time.RFC3339Nano),
		"Kitchen":     Str(time.Kitchen),
		"Stamp":       Str(time.Stamp),
		"StampMilli":  Str(time.StampMilli),
		"StampMicro":  Str(time.StampMicro),
		"StampNano":   Str(time.StampNano),

		// gad:doc
		// ### Durations
		//
		// Nanosecond
		// Microsecond
		// Millisecond
		// Second
		// Minute
		// Hour
		"Nanosecond":  Int(time.Nanosecond),
		"Microsecond": Int(time.Microsecond),
		"Millisecond": Int(time.Millisecond),
		"Second":      Int(time.Second),
		"Minute":      Int(time.Minute),
		"Hour":        Int(time.Hour),

		// gad:doc
		// ## Functions
		// utc() <Location>
		// Returns Universal Coordinated Time (UTC) location.
		"utc": &BuiltinFunction{
			FuncName: "utc",
			Value:    funcPRO(TimeUtcFunc),
		},

		// gad:doc
		// local() <Location>
		// Returns the system's local time zone location.
		"local": &BuiltinFunction{
			FuncName: "local",
			Value:    funcPRO(TimeLocalFunc),
		},

		// gad:doc
		// monthString(m int) <month str>
		// Returns English name of the month m ("January", "February", ...).
		"monthString": &BuiltinFunction{
			FuncName: "monthString",
			Value:    funcPiRO(TimeMonthStringFunc),
		},

		// gad:doc
		// weekdayString(w int) <weekday str>
		// Returns English name of the int weekday w, note that 0 is Sunday.
		"weekdayString": &BuiltinFunction{
			FuncName: "weekdayString",
			Value:    funcPiRO(TimeWeekdayStringFunc),
		},

		// gad:doc
		// durationString(d int) <str>
		// Returns a string representing the duration d in the form "72h3m0.5s".
		"durationString": &BuiltinFunction{
			FuncName: "durationString",
			Value:    funcPi64RO(TimeDurationStringFunc),
		},
		// gad:doc
		// durationNanoseconds(d int) <int>
		// Returns the duration d as an int nanosecond count.
		"durationNanoseconds": &BuiltinFunction{
			FuncName: "durationNanoseconds",
			Value:    funcPi64RO(TimeDurationNanosecondsFunc),
		},
		// gad:doc
		// durationMicroseconds(d int) <int>
		// Returns the duration d as an int microsecond count.
		"durationMicroseconds": &BuiltinFunction{
			FuncName: "durationMicroseconds",
			Value:    funcPi64RO(TimeDurationMicrosecondsFunc),
		},
		// gad:doc
		// durationMilliseconds(d int) <int>
		// Returns the duration d as an int millisecond count.
		"durationMilliseconds": &BuiltinFunction{
			FuncName: "durationMilliseconds",
			Value:    funcPi64RO(TimeDurationMillisecondsFunc),
		},
		// gad:doc
		// durationSeconds(d int) <float>
		// Returns the duration d as a floating point number of seconds.
		"durationSeconds": &BuiltinFunction{
			FuncName: "durationSeconds",
			Value:    funcPi64RO(TimeDurationSecondsFunc),
		},
		// gad:doc
		// durationMinutes(d int) <float>
		// Returns the duration d as a floating point number of minutes.
		"durationMinutes": &BuiltinFunction{
			FuncName: "durationMinutes",
			Value:    funcPi64RO(TimeDurationMinutesFunc),
		},
		// gad:doc
		// durationHours(d int) <float>
		// Returns the duration d as a floating point number of hours.
		"durationHours": &BuiltinFunction{
			FuncName: "durationHours",
			Value:    funcPi64RO(TimeDurationHoursFunc),
		},
		// gad:doc
		// sleep(duration int) <nil>
		// Pauses the current goroutine for at least the duration.
		"sleep": &BuiltinFunction{
			FuncName: "sleep",
			Value:    TimeSleepFunc,
		},
		// gad:doc
		// parseDuration(s str) <duration int>
		// Parses duration s and returns duration as int or error.
		"parseDuration": &BuiltinFunction{
			FuncName: "parseDuration",
			Value:    funcPsROe(TimeParseDurationFunc),
		},
		// gad:doc
		// durationRound(duration int, m int) <duration int>
		// Returns the result of rounding duration to the nearest multiple of m.
		"durationRound": &BuiltinFunction{
			FuncName: "durationRound",
			Value:    funcPi64i64RO(TimeDurationRoundFunc),
		},
		// gad:doc
		// durationTruncate(duration int, m int) <duration int>
		// Returns the result of rounding duration toward zero to a multiple of m.
		"durationTruncate": &BuiltinFunction{
			FuncName: "durationTruncate",
			Value:    funcPi64i64RO(TimeDurationTruncateFunc),
		},
		// gad:doc
		// fixedZone(name str, sec int) <Location>
		// Returns a Location that always uses the given zone name and offset
		// (seconds east of UTC).
		"fixedZone": &BuiltinFunction{
			FuncName: "fixedZone",
			Value:    funcPsiRO(TimeFixedZoneFunc),
		},
		// gad:doc
		// loadLocation(name str) <Location>
		// Returns the Location with the given name.
		"loadLocation": &BuiltinFunction{
			FuncName: "loadLocation",
			Value:    funcPsROe(TimeLoadLocationFunc),
		},
		// gad:doc
		// isLocation(any) <bool>
		// Reports whether any value is of location type.
		"isLocation": &BuiltinFunction{
			FuncName: "isLocation",
			Value:    funcPORO(TimeIsLocationFunc),
		},
		// gad:doc
		// time() <time>
		// Returns zero time.
		"time": &BuiltinFunction{
			FuncName: "time",
			Value:    funcPRO(TimeZerotimeFunc),
		},
		// gad:doc
		// since(t time) <duration int>
		// Returns the time elapsed since t.
		"since": &BuiltinFunction{
			FuncName: "since",
			Value:    FuncPTRO(TimeSinceFunc),
		},
		// gad:doc
		// until(t time) <duration int>
		// Returns the duration until t.
		"until": &BuiltinFunction{
			FuncName: "until",
			Value:    FuncPTRO(TimeUntilFunc),
		},
		// gad:doc
		// date(year int, month int, day int[, hour int, min int, sec int, nsec int, loc Location]) <time>
		// Returns the Time corresponding to yyyy-mm-dd hh:mm:ss + nsec nanoseconds
		// in the appropriate zone for that time in the given location. Zero values
		// of optional arguments are used if not provided.
		"date": &BuiltinFunction{
			FuncName: "date",
			Value:    TimeDateFunc,
		},
		// gad:doc
		// now() <time>
		// Returns the current local time.
		"now": &BuiltinFunction{
			FuncName: "now",
			Value:    funcPRO(TimeNowFunc),
		},
		// gad:doc
		// parse(layout str, value str[, loc Location]) <time>
		// Parses a formatted string and returns the time value it represents.
		// If location is not provided, ToInterface's `time.Parse` function is called
		// otherwise `time.ParseInLocation` is called.
		"parse": &BuiltinFunction{
			FuncName: "parse",
			Value:    TimeParseFunc,
		},
		// gad:doc
		// strToDate(s str) <date>
		// Parses a date from "YYYYMMDD" or "YYYY-MM-DD".
		"strToDate": &BuiltinFunction{
			FuncName: "strToDate",
			Value:    timeStrToFunc(func(s string) (Object, error) { return strToDate(s) }),
		},
		// gad:doc
		// strToTime(s str) <time>
		// Parses a time literal `[YYYYMMDD[_]]HHMMSS[.frac][Zloc][T]`.
		"strToTime": &BuiltinFunction{
			FuncName: "strToTime",
			Value: timeStrToFunc(func(s string) (Object, error) {
				t, err := strToTime(s)
				return &Time{Value: t}, err
			}),
		},
		// gad:doc
		// strToDuration(s str) <duration>
		// Parses a Go duration string (e.g. "1h30m").
		"strToDuration": &BuiltinFunction{
			FuncName: "strToDuration",
			Value:    timeStrToFunc(func(s string) (Object, error) { return strToDuration(s) }),
		},
		// gad:doc
		// strToLocation(s str) <Location>
		// Parses a location from an offset ("-0300"/"-03:00") or an IANA name.
		"strToLocation": &BuiltinFunction{
			FuncName: "strToLocation",
			Value: timeStrToFunc(func(s string) (Object, error) {
				loc, err := strToLocation(s)
				return &Location{Value: loc}, err
			}),
		},
		// gad:doc
		// unix(sec int[, nsec int]) <time>
		// Returns the local time corresponding to the given Unix time,
		// sec seconds and nsec nanoseconds since January 1, 1970 UTC.
		// Zero values of optional arguments are used if not provided.
		"unix": &BuiltinFunction{
			FuncName: "unix",
			Value:    TimeUnixFunc,
		},
		// gad:doc
		// add(t time, duration int) <time>
		// Deprecated: Use .Add method of time object.
		// Returns the time of t+duration.
		"add": &BuiltinFunction{
			FuncName: "add",
			Value:    FuncPTi64RO(TimeAdd),
		},
		// gad:doc
		// sub(t1 time, t2 time) <int>
		// Deprecated: Use .Sub method of time object.
		// Returns the duration of t1-t2.
		"sub": &BuiltinFunction{
			FuncName: "sub",
			Value:    FuncPTTRO(TimeSub),
		},
		// gad:doc
		// addDate(t time, years int, months int, days int) <time>
		// Deprecated: Use .AddDate method of time object.
		// Returns the time corresponding to adding the given number of
		// years, months, and days to t.
		"addDate": &BuiltinFunction{
			FuncName: "addDate",
			Value:    FuncPTiiiRO(TimeAddDate),
		},
		// gad:doc
		// after(t1 time, t2 time) <bool>
		// Deprecated: Use .After method of time object.
		// Reports whether the time t1 is after t2.
		"after": &BuiltinFunction{
			FuncName: "after",
			Value:    FuncPTTRO(TimeAfter),
		},
		// gad:doc
		// before(t1 time, t2 time) <bool>
		// Deprecated: Use .Before method of time object.
		// Reports whether the time t1 is before t2.
		"before": &BuiltinFunction{
			FuncName: "before",
			Value:    FuncPTTRO(TimeBefore),
		},
		// gad:doc
		// format(t time, layout str) <str>
		// Deprecated: Use .Format method of time object.
		// Returns a textual representation of the time value formatted according
		// to layout.
		"format": &BuiltinFunction{
			FuncName: "format",
			Value:    FuncPTsRO(TimeFormat),
		},
		// gad:doc
		// appendFormat(t time, b bytes, layout str) <bytes>
		// Deprecated: Use .AppendFormat method of time object.
		// It is like `Format` but appends the textual representation to b and
		// returns the extended buffer.
		"appendFormat": &BuiltinFunction{
			FuncName: "appendFormat", // FuncPTb2sRO
			Value:    FuncPTb2sRO(TimeAppendFormat),
		},
		// gad:doc
		// in(t time, loc Location) <time>
		// Deprecated: Use .In method of time object.
		// Returns a copy of t representing the same time t, but with the copy's
		// location information set to loc for display purposes.
		"in": &BuiltinFunction{
			FuncName: "in",
			Value:    FuncPTLRO(TimeIn),
		},
		// gad:doc
		// round(t time, duration int) <time>
		// Deprecated: Use .Round method of time object.
		// Round returns the result of rounding t to the nearest multiple of
		// duration.
		"round": &BuiltinFunction{
			FuncName: "round",
			Value:    FuncPTi64RO(TimeRound),
		},
		// gad:doc
		// truncate(t time, duration int) <time>
		// Deprecated: Use .Truncate method of time object.
		// Truncate returns the result of rounding t down to a multiple of duration.
		"truncate": &BuiltinFunction{
			FuncName: "truncate",
			Value:    FuncPTi64RO(TimeTruncate),
		},
		// gad:doc
		// isTime(any) <bool>
		// Reports whether any value is of time type.
		"isTime": &BuiltinFunction{
			FuncName: "isTime",
			Value:    funcPORO(TimeIsTimeFunc),
		},
	}
}

func TimeUtcFunc() Object { return TimeUtcLoc }

func TimeLocalFunc() Object { return TimeLocalLoc }

func TimeMonthStringFunc(m int) Object {
	return Str(time.Month(m).String())
}

func TimeWeekdayStringFunc(w int) Object {
	return Str(time.Weekday(w).String())
}

func TimeDurationStringFunc(d int64) Object {
	return Str(time.Duration(d).String())
}

func TimeDurationNanosecondsFunc(d int64) Object {
	return Int(time.Duration(d).Nanoseconds())
}

func TimeDurationMicrosecondsFunc(d int64) Object {
	return Int(time.Duration(d).Microseconds())
}

func TimeDurationMillisecondsFunc(d int64) Object {
	return Int(time.Duration(d).Milliseconds())
}

func TimeDurationSecondsFunc(d int64) Object {
	return Float(time.Duration(d).Seconds())
}

func TimeDurationMinutesFunc(d int64) Object {
	return Float(time.Duration(d).Minutes())
}

func TimeDurationHoursFunc(d int64) Object {
	return Float(time.Duration(d).Hours())
}

func TimeSleepFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	arg0 := c.Args.Get(0)

	var dur time.Duration
	if v, ok := ToGoInt64(arg0); !ok {
		return TimeNewArgTypeErr("1st", "int", arg0.Type().Name())
	} else {
		dur = time.Duration(v)
	}

	for {
		if dur <= 10*time.Millisecond {
			time.Sleep(dur)
			break
		}
		dur -= 10 * time.Millisecond
		time.Sleep(10 * time.Millisecond)
		if c.VM.Aborted() {
			return Nil, ErrVMAborted
		}
	}
	return Nil, nil
}

func TimeParseDurationFunc(s string) (Object, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	return Int(d), nil
}

func TimeDurationRoundFunc(d, m int64) Object {
	return Int(time.Duration(d).Round(time.Duration(m)))
}

func TimeDurationTruncateFunc(d, m int64) Object {
	return Int(time.Duration(d).Truncate(time.Duration(m)))
}

func TimeFixedZoneFunc(name string, sec int) Object {
	return &Location{Value: time.FixedZone(name, sec)}
}

func TimeLoadLocationFunc(name string) (Object, error) {
	l, err := time.LoadLocation(name)
	if err != nil {
		return Nil, err
	}
	return &Location{Value: l}, nil
}

func TimeIsLocationFunc(o Object) Object {
	_, ok := o.(*Location)
	return Bool(ok)
}

func TimeZerotimeFunc() Object { return TimeZeroTime }

func TimeSinceFunc(t *Time) Object { return Int(time.Since(t.Value)) }

func TimeUntilFunc(t *Time) Object { return Int(time.Until(t.Value)) }

func TimeDateFunc(c Call) (Object, error) {
	size := c.Args.Length()
	if size < 3 || size > 8 {
		return Nil, ErrWrongNumArguments.NewError(
			"want=3..8 got=" + strconv.Itoa(size))
	}
	ymdHmsn := [7]int{}
	loc := &Location{Value: time.Local}
	var ok bool
	for i := 0; i < size; i++ {
		arg := c.Args.Get(i)
		if i < 7 {
			ymdHmsn[i], ok = ToGoInt(arg)
			if !ok {
				return TimeNewArgTypeErr(strconv.Itoa(i+1), "int", arg.Type().Name())
			}
			continue
		}
		loc, ok = arg.(*Location)
		if !ok {
			return TimeNewArgTypeErr(strconv.Itoa(i+1), "Location", arg.Type().Name())
		}
	}

	return &Time{
		Value: time.Date(ymdHmsn[0], time.Month(ymdHmsn[1]), ymdHmsn[2],
			ymdHmsn[3], ymdHmsn[4], ymdHmsn[5], ymdHmsn[6], loc.Value),
	}, nil
}

func TimeNowFunc() Object { return &Time{Value: time.Now()} }

func TimeParseFunc(c Call) (Object, error) {
	size := c.Args.Length()
	if size != 2 && size != 3 {
		return Nil, ErrWrongNumArguments.NewError(
			"want=2..3 got=" + strconv.Itoa(size))
	}
	layout, ok := ToGoString(c.Args.Get(0))
	if !ok {
		return TimeNewArgTypeErr("1st", "str", c.Args.Get(0).Type().Name())
	}
	value, ok := ToGoString(c.Args.Get(1))
	if !ok {
		return TimeNewArgTypeErr("2nd", "str", c.Args.Get(1).Type().Name())
	}
	if size == 2 {
		tm, err := time.Parse(layout, value)
		if err != nil {
			return Nil, err
		}
		return &Time{Value: tm}, nil
	}
	loc, ok := ToLocation(c.Args.Get(2))
	if !ok {
		return TimeNewArgTypeErr("3rd", "Location", c.Args.Get(2).Type().Name())
	}
	tm, err := time.ParseInLocation(layout, value, loc.Value)
	if err != nil {
		return Nil, err
	}
	return &Time{Value: tm}, nil
}

func TimeUnixFunc(c Call) (Object, error) {
	size := c.Args.Length()
	if size != 1 && size != 2 {
		return Nil, ErrWrongNumArguments.NewError(
			"want=1..2 got=" + strconv.Itoa(size))
	}

	sec, ok := ToGoInt64(c.Args.Get(0))
	if !ok {
		return TimeNewArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
	}

	var nsec int64
	if size > 1 {
		nsec, ok = ToGoInt64(c.Args.Get(1))
		if !ok {
			return TimeNewArgTypeErr("2nd", "int", c.Args.Get(1).Type().Name())
		}
	}
	return &Time{Value: time.Unix(sec, nsec)}, nil
}

func TimeIsTimeFunc(o Object) Object {
	_, ok := o.(*Time)
	return Bool(ok)
}

// init registers the time module's Go<->object converters globally (via the
// registry package, no VM) so the builtin `time` namespace interoperates with
// Go time values exactly like the imported module, and installs the
// `int(v time; unit=char)` override on the builtin int type.
//
// Converters:
//   - `time.Time`, `*time.Time` -> `time`
//   - `time.Duration` -> `int` (nanoseconds)
//   - `time.Location` -> `location`
func init() {
	registry.RegisterAnyConverter(reflect.TypeOf((*Time)(nil)), func(in any) (any, bool) {
		return in.(*Time).Value, true
	})
	registry.RegisterAnyConverter(reflect.TypeOf((*Location)(nil)), func(in any) (any, bool) {
		return in.(*Location).Value, true
	})
	registry.RegisterAnyConverter(reflect.TypeOf(Duration(0)), func(in any) (any, bool) {
		return time.Duration(in.(Duration)), true
	})
	registry.RegisterAnyConverter(reflect.TypeOf(CalendarDate(0)), func(in any) (any, bool) {
		return uint(in.(CalendarDate)), true
	})
	registry.RegisterObjectConverter(reflect.TypeFor[time.Time](), func(in any) (any, bool) {
		return &Time{Value: in.(time.Time)}, true
	})
	registry.RegisterObjectConverter(reflect.TypeFor[*time.Time](), func(in any) (any, bool) {
		t := in.(*time.Time)
		if t == nil {
			return Nil, true
		}
		return &Time{Value: *t}, true
	})
	registry.RegisterObjectConverter(reflect.TypeFor[time.Duration](), func(in any) (any, bool) {
		return Int(in.(time.Duration)), true
	})
	TimeLocationConv := func(in any) (any, bool) {
		switch t := in.(type) {
		case *time.Location:
			if t == nil {
				return Nil, true
			}
			return &Location{Value: t}, true
		case time.Location:
			return &Location{Value: &t}, true
		}
		return nil, false
	}
	registry.RegisterObjectConverter(reflect.TypeFor[time.Location](), TimeLocationConv)
	registry.RegisterObjectConverter(reflect.TypeFor[*time.Location](), TimeLocationConv)

	// int(v time; unit=char) <int>: converts a `time` to a Unix timestamp
	// (elapsed since 1970-01-01 UTC). The `unit` named argument selects the
	// resolution: `'n'` nanoseconds, `'m'` microseconds, `'l'` milliseconds, or
	// seconds (default). Registered via BuiltinObjects.AddMethod (not
	// AddMethodOverride) so the method survives StaticBuiltins.build().
	BuiltinObjects.AddMethod(BuiltinInt, NewFunction("TimeToInt",
		func(c Call) (o Object, err error) {
			var (
				arg = &Arg{Name: "v"}
				get = GoTimeArg(arg)
			)

			if err = c.Args.Destructure(arg); err != nil {
				return
			}

			var (
				t       = get().Value
				unit, _ = c.NamedArgs.MustGetValueOrNil("unit").(Char)
			)

			switch unit {
			case 'n':
				return Int(t.UnixNano()), nil
			case 'm':
				return Int(t.UnixMicro()), nil
			case 'l':
				return Int(t.UnixMilli()), nil
			default:
				return Int(t.Unix()), nil
			}
		},
		FunctionWithUsage(`converts time to Unix time value elapsed since January 1, 1970 UTC`),
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("v").Type(TimeType).Usage("time object")
		}),
		FunctionWithNamedParams(func(newParam func(name string) *NamedParamBuilder) {
			newParam("unit").Type(TChar).Usage(`
Available values:

'n'
	the number of nano seconds.
'm'
	the number of micro seconds.
'l'
	the number of milli seconds.
default
	the number of seconds.
`)
		})))
}

func GoTimeArg(arg *Arg) (get func() *Time) {
	arg.TypeAssertion = TypeAssertionFromTypes(TimeType)
	return func() *Time {
		return arg.Value.(*Time)
	}
}
