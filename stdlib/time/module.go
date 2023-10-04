// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Package time provides time module for measuring and displaying time for Gad
// script language. It wraps Go's time package functionalities.
// Note that: Gad's int values are converted to Go's time.Duration values.
package time

import (
	"strconv"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/stdlib"
)

var utcLoc gad.Object = &Location{Value: time.UTC}
var localLoc gad.Object = &Location{Value: time.Local}
var zeroTime gad.Object = &Time{}

// Module represents time module.
var Module = map[string]gad.Object{
	// gad:doc
	// # time Module
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
	"January":   gad.Int(time.January),
	"February":  gad.Int(time.February),
	"March":     gad.Int(time.March),
	"April":     gad.Int(time.April),
	"May":       gad.Int(time.May),
	"June":      gad.Int(time.June),
	"July":      gad.Int(time.July),
	"August":    gad.Int(time.August),
	"September": gad.Int(time.September),
	"October":   gad.Int(time.October),
	"November":  gad.Int(time.November),
	"December":  gad.Int(time.December),

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
	"Sunday":    gad.Int(time.Sunday),
	"Monday":    gad.Int(time.Monday),
	"Tuesday":   gad.Int(time.Tuesday),
	"Wednesday": gad.Int(time.Wednesday),
	"Thursday":  gad.Int(time.Thursday),
	"Friday":    gad.Int(time.Friday),
	"Saturday":  gad.Int(time.Saturday),

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
	"ANSIC":       gad.String(time.ANSIC),
	"UnixDate":    gad.String(time.UnixDate),
	"RubyDate":    gad.String(time.RubyDate),
	"RFC822":      gad.String(time.RFC822),
	"RFC822Z":     gad.String(time.RFC822Z),
	"RFC850":      gad.String(time.RFC850),
	"RFC1123":     gad.String(time.RFC1123),
	"RFC1123Z":    gad.String(time.RFC1123Z),
	"RFC3339":     gad.String(time.RFC3339),
	"RFC3339Nano": gad.String(time.RFC3339Nano),
	"Kitchen":     gad.String(time.Kitchen),
	"Stamp":       gad.String(time.Stamp),
	"StampMilli":  gad.String(time.StampMilli),
	"StampMicro":  gad.String(time.StampMicro),
	"StampNano":   gad.String(time.StampNano),

	// gad:doc
	// ### Durations
	//
	// Nanosecond
	// Microsecond
	// Millisecond
	// Second
	// Minute
	// Hour
	"Nanosecond":  gad.Int(time.Nanosecond),
	"Microsecond": gad.Int(time.Microsecond),
	"Millisecond": gad.Int(time.Millisecond),
	"Second":      gad.Int(time.Second),
	"Minute":      gad.Int(time.Minute),
	"Hour":        gad.Int(time.Hour),

	// gad:doc
	// ## Functions
	// UTC() -> location
	// Returns Universal Coordinated Time (UTC) location.
	"UTC": &gad.Function{
		Name:  "UTC",
		Value: stdlib.FuncPRO(utcFunc),
	},

	// gad:doc
	// Local() -> location
	// Returns the system's local time zone location.
	"Local": &gad.Function{
		Name:  "Local",
		Value: stdlib.FuncPRO(localFunc),
	},

	// gad:doc
	// MonthString(m int) -> month string
	// Returns English name of the month m ("January", "February", ...).
	"MonthString": &gad.Function{
		Name:  "MonthString",
		Value: stdlib.FuncPiRO(monthStringFunc),
	},

	// gad:doc
	// WeekdayString(w int) -> weekday string
	// Returns English name of the int weekday w, note that 0 is Sunday.
	"WeekdayString": &gad.Function{
		Name:  "WeekdayString",
		Value: stdlib.FuncPiRO(weekdayStringFunc),
	},

	// gad:doc
	// DurationString(d int) -> string
	// Returns a string representing the duration d in the form "72h3m0.5s".
	"DurationString": &gad.Function{
		Name:  "DurationString",
		Value: stdlib.FuncPi64RO(durationStringFunc),
	},
	// gad:doc
	// DurationNanoseconds(d int) -> int
	// Returns the duration d as an int nanosecond count.
	"DurationNanoseconds": &gad.Function{
		Name:  "DurationNanoseconds",
		Value: stdlib.FuncPi64RO(durationNanosecondsFunc),
	},
	// gad:doc
	// DurationMicroseconds(d int) -> int
	// Returns the duration d as an int microsecond count.
	"DurationMicroseconds": &gad.Function{
		Name:  "DurationMicroseconds",
		Value: stdlib.FuncPi64RO(durationMicrosecondsFunc),
	},
	// gad:doc
	// DurationMilliseconds(d int) -> int
	// Returns the duration d as an int millisecond count.
	"DurationMilliseconds": &gad.Function{
		Name:  "DurationMilliseconds",
		Value: stdlib.FuncPi64RO(durationMillisecondsFunc),
	},
	// gad:doc
	// DurationSeconds(d int) -> float
	// Returns the duration d as a floating point number of seconds.
	"DurationSeconds": &gad.Function{
		Name:  "DurationSeconds",
		Value: stdlib.FuncPi64RO(durationSecondsFunc),
	},
	// gad:doc
	// DurationMinutes(d int) -> float
	// Returns the duration d as a floating point number of minutes.
	"DurationMinutes": &gad.Function{
		Name:  "DurationMinutes",
		Value: stdlib.FuncPi64RO(durationMinutesFunc),
	},
	// gad:doc
	// DurationHours(d int) -> float
	// Returns the duration d as a floating point number of hours.
	"DurationHours": &gad.Function{
		Name:  "DurationHours",
		Value: stdlib.FuncPi64RO(durationHoursFunc),
	},
	// gad:doc
	// Sleep(duration int) -> nil
	// Pauses the current goroutine for at least the duration.
	"Sleep": &gad.Function{
		Name:  "Sleep",
		Value: sleepFunc,
	},
	// gad:doc
	// ParseDuration(s string) -> duration int
	// Parses duration s and returns duration as int or error.
	"ParseDuration": &gad.Function{
		Name:  "ParseDuration",
		Value: stdlib.FuncPsROe(parseDurationFunc),
	},
	// gad:doc
	// DurationRound(duration int, m int) -> duration int
	// Returns the result of rounding duration to the nearest multiple of m.
	"DurationRound": &gad.Function{
		Name:  "DurationRound",
		Value: stdlib.FuncPi64i64RO(durationRoundFunc),
	},
	// gad:doc
	// DurationTruncate(duration int, m int) -> duration int
	// Returns the result of rounding duration toward zero to a multiple of m.
	"DurationTruncate": &gad.Function{
		Name:  "DurationTruncate",
		Value: stdlib.FuncPi64i64RO(durationTruncateFunc),
	},
	// gad:doc
	// FixedZone(name string, sec int) -> location
	// Returns a Location that always uses the given zone name and offset
	// (seconds east of UTC).
	"FixedZone": &gad.Function{
		Name:  "FixedZone",
		Value: stdlib.FuncPsiRO(fixedZoneFunc),
	},
	// gad:doc
	// LoadLocation(name string) -> location
	// Returns the Location with the given name.
	"LoadLocation": &gad.Function{
		Name:  "LoadLocation",
		Value: stdlib.FuncPsROe(loadLocationFunc),
	},
	// gad:doc
	// IsLocation(any) -> bool
	// Reports whether any value is of location type.
	"IsLocation": &gad.Function{
		Name:  "IsLocation",
		Value: stdlib.FuncPORO(isLocationFunc),
	},
	// gad:doc
	// Time() -> time
	// Returns zero time.
	"Time": &gad.Function{
		Name:  "Time",
		Value: stdlib.FuncPRO(zerotimeFunc),
	},
	// gad:doc
	// Since(t time) -> duration int
	// Returns the time elapsed since t.
	"Since": &gad.Function{
		Name:  "Since",
		Value: funcPTRO(sinceFunc),
	},
	// gad:doc
	// Until(t time) -> duration int
	// Returns the duration until t.
	"Until": &gad.Function{
		Name:  "Until",
		Value: funcPTRO(untilFunc),
	},
	// gad:doc
	// Date(year int, month int, day int[, hour int, min int, sec int, nsec int, loc location]) -> time
	// Returns the Time corresponding to yyyy-mm-dd hh:mm:ss + nsec nanoseconds
	// in the appropriate zone for that time in the given location. Zero values
	// of optional arguments are used if not provided.
	"Date": &gad.Function{
		Name:  "Date",
		Value: dateFunc,
	},
	// gad:doc
	// Now() -> time
	// Returns the current local time.
	"Now": &gad.Function{
		Name:  "Now",
		Value: stdlib.FuncPRO(nowFunc),
	},
	// gad:doc
	// Parse(layout string, value string[, loc location]) -> time
	// Parses a formatted string and returns the time value it represents.
	// If location is not provided, Go's `time.Parse` function is called
	// otherwise `time.ParseInLocation` is called.
	"Parse": &gad.Function{
		Name:  "Parse",
		Value: parseFunc,
	},
	// gad:doc
	// Unix(sec int[, nsec int]) -> time
	// Returns the local time corresponding to the given Unix time,
	// sec seconds and nsec nanoseconds since January 1, 1970 UTC.
	// Zero values of optional arguments are used if not provided.
	"Unix": &gad.Function{
		Name:  "Unix",
		Value: unixFunc,
	},
	// gad:doc
	// Add(t time, duration int) -> time
	// Deprecated: Use .Add method of time object.
	// Returns the time of t+duration.
	"Add": &gad.Function{
		Name:  "Add",
		Value: funcPTi64RO(timeAdd),
	},
	// gad:doc
	// Sub(t1 time, t2 time) -> int
	// Deprecated: Use .Sub method of time object.
	// Returns the duration of t1-t2.
	"Sub": &gad.Function{
		Name:  "Sub",
		Value: funcPTTRO(timeSub),
	},
	// gad:doc
	// AddDate(t time, years int, months int, days int) -> time
	// Deprecated: Use .AddDate method of time object.
	// Returns the time corresponding to adding the given number of
	// years, months, and days to t.
	"AddDate": &gad.Function{
		Name:  "AddDate",
		Value: funcPTiiiRO(timeAddDate),
	},
	// gad:doc
	// After(t1 time, t2 time) -> bool
	// Deprecated: Use .After method of time object.
	// Reports whether the time t1 is after t2.
	"After": &gad.Function{
		Name:  "After",
		Value: funcPTTRO(timeAfter),
	},
	// gad:doc
	// Before(t1 time, t2 time) -> bool
	// Deprecated: Use .Before method of time object.
	// Reports whether the time t1 is before t2.
	"Before": &gad.Function{
		Name:  "Before",
		Value: funcPTTRO(timeBefore),
	},
	// gad:doc
	// Format(t time, layout string) -> string
	// Deprecated: Use .Format method of time object.
	// Returns a textual representation of the time value formatted according
	// to layout.
	"Format": &gad.Function{
		Name:  "Format",
		Value: funcPTsRO(timeFormat),
	},
	// gad:doc
	// AppendFormat(t time, b bytes, layout string) -> bytes
	// Deprecated: Use .AppendFormat method of time object.
	// It is like `Format` but appends the textual representation to b and
	// returns the extended buffer.
	"AppendFormat": &gad.Function{
		Name:  "AppendFormat", // funcPTb2sRO
		Value: funcPTb2sRO(timeAppendFormat),
	},
	// gad:doc
	// In(t time, loc location) -> time
	// Deprecated: Use .In method of time object.
	// Returns a copy of t representing the same time t, but with the copy's
	// location information set to loc for display purposes.
	"In": &gad.Function{
		Name:  "In",
		Value: funcPTLRO(timeIn),
	},
	// gad:doc
	// Round(t time, duration int) -> time
	// Deprecated: Use .Round method of time object.
	// Round returns the result of rounding t to the nearest multiple of
	// duration.
	"Round": &gad.Function{
		Name:  "Round",
		Value: funcPTi64RO(timeRound),
	},
	// gad:doc
	// Truncate(t time, duration int) -> time
	// Deprecated: Use .Truncate method of time object.
	// Truncate returns the result of rounding t down to a multiple of duration.
	"Truncate": &gad.Function{
		Name:  "Truncate",
		Value: funcPTi64RO(timeTruncate),
	},
	// gad:doc
	// IsTime(any) -> bool
	// Reports whether any value is of time type.
	"IsTime": &gad.Function{
		Name:  "IsTime",
		Value: stdlib.FuncPORO(isTimeFunc),
	},
}

func utcFunc() gad.Object { return utcLoc }

func localFunc() gad.Object { return localLoc }

func monthStringFunc(m int) gad.Object {
	return gad.String(time.Month(m).String())
}

func weekdayStringFunc(w int) gad.Object {
	return gad.String(time.Weekday(w).String())
}

func durationStringFunc(d int64) gad.Object {
	return gad.String(time.Duration(d).String())
}

func durationNanosecondsFunc(d int64) gad.Object {
	return gad.Int(time.Duration(d).Nanoseconds())
}

func durationMicrosecondsFunc(d int64) gad.Object {
	return gad.Int(time.Duration(d).Microseconds())
}

func durationMillisecondsFunc(d int64) gad.Object {
	return gad.Int(time.Duration(d).Milliseconds())
}

func durationSecondsFunc(d int64) gad.Object {
	return gad.Float(time.Duration(d).Seconds())
}

func durationMinutesFunc(d int64) gad.Object {
	return gad.Float(time.Duration(d).Minutes())
}

func durationHoursFunc(d int64) gad.Object {
	return gad.Float(time.Duration(d).Hours())
}

func sleepFunc(c gad.Call) (gad.Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return gad.Nil, err
	}
	arg0 := c.Args.Get(0)

	var dur time.Duration
	if v, ok := gad.ToGoInt64(arg0); !ok {
		return newArgTypeErr("1st", "int", arg0.Type().Name())
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
			return gad.Nil, gad.ErrVMAborted
		}
	}
	return gad.Nil, nil
}

func parseDurationFunc(s string) (gad.Object, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}
	return gad.Int(d), nil
}

func durationRoundFunc(d, m int64) gad.Object {
	return gad.Int(time.Duration(d).Round(time.Duration(m)))
}

func durationTruncateFunc(d, m int64) gad.Object {
	return gad.Int(time.Duration(d).Truncate(time.Duration(m)))
}

func fixedZoneFunc(name string, sec int) gad.Object {
	return &Location{Value: time.FixedZone(name, sec)}
}

func loadLocationFunc(name string) (gad.Object, error) {
	l, err := time.LoadLocation(name)
	if err != nil {
		return gad.Nil, err
	}
	return &Location{Value: l}, nil
}

func isLocationFunc(o gad.Object) gad.Object {
	_, ok := o.(*Location)
	return gad.Bool(ok)
}

func zerotimeFunc() gad.Object { return zeroTime }

func sinceFunc(t *Time) gad.Object { return gad.Int(time.Since(t.Value)) }

func untilFunc(t *Time) gad.Object { return gad.Int(time.Until(t.Value)) }

func dateFunc(c gad.Call) (gad.Object, error) {
	size := c.Args.Len()
	if size < 3 || size > 8 {
		return gad.Nil, gad.ErrWrongNumArguments.NewError(
			"want=3..8 got=" + strconv.Itoa(size))
	}
	ymdHmsn := [7]int{}
	loc := &Location{Value: time.Local}
	var ok bool
	for i := 0; i < size; i++ {
		arg := c.Args.Get(i)
		if i < 7 {
			ymdHmsn[i], ok = gad.ToGoInt(arg)
			if !ok {
				return newArgTypeErr(strconv.Itoa(i+1), "int", arg.Type().Name())
			}
			continue
		}
		loc, ok = arg.(*Location)
		if !ok {
			return newArgTypeErr(strconv.Itoa(i+1), "location", arg.Type().Name())
		}
	}

	return &Time{
		Value: time.Date(ymdHmsn[0], time.Month(ymdHmsn[1]), ymdHmsn[2],
			ymdHmsn[3], ymdHmsn[4], ymdHmsn[5], ymdHmsn[6], loc.Value),
	}, nil
}

func nowFunc() gad.Object { return &Time{Value: time.Now()} }

func parseFunc(c gad.Call) (gad.Object, error) {
	size := c.Args.Len()
	if size != 2 && size != 3 {
		return gad.Nil, gad.ErrWrongNumArguments.NewError(
			"want=2..3 got=" + strconv.Itoa(size))
	}
	layout, ok := gad.ToGoString(c.Args.Get(0))
	if !ok {
		return newArgTypeErr("1st", "string", c.Args.Get(0).Type().Name())
	}
	value, ok := gad.ToGoString(c.Args.Get(1))
	if !ok {
		return newArgTypeErr("2nd", "string", c.Args.Get(1).Type().Name())
	}
	if size == 2 {
		tm, err := time.Parse(layout, value)
		if err != nil {
			return gad.Nil, err
		}
		return &Time{Value: tm}, nil
	}
	loc, ok := ToLocation(c.Args.Get(2))
	if !ok {
		return newArgTypeErr("3rd", "location", c.Args.Get(2).Type().Name())
	}
	tm, err := time.ParseInLocation(layout, value, loc.Value)
	if err != nil {
		return gad.Nil, err
	}
	return &Time{Value: tm}, nil
}

func unixFunc(c gad.Call) (gad.Object, error) {
	size := c.Args.Len()
	if size != 1 && size != 2 {
		return gad.Nil, gad.ErrWrongNumArguments.NewError(
			"want=1..2 got=" + strconv.Itoa(size))
	}

	sec, ok := gad.ToGoInt64(c.Args.Get(0))
	if !ok {
		return newArgTypeErr("1st", "int", c.Args.Get(0).Type().Name())
	}

	var nsec int64
	if size > 1 {
		nsec, ok = gad.ToGoInt64(c.Args.Get(1))
		if !ok {
			return newArgTypeErr("2nd", "int", c.Args.Get(1).Type().Name())
		}
	}
	return &Time{Value: time.Unix(sec, nsec)}, nil
}

func isTimeFunc(o gad.Object) gad.Object {
	_, ok := o.(*Time)
	return gad.Bool(ok)
}
