// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"time"
)

// truncateUnitArg reads the unit argument shared by the .truncate methods of
// the time value types. The unit is a char or string naming a calendar unit
// ('y', 'M', 'w', 'd') or a Go duration unit ("h", "m", "s", "ms", "us"/"µs",
// "ns").
func truncateUnitArg(c Call) (string, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return "", err
	}
	switch v := c.Args.Get(0).(type) {
	case Char:
		return string(rune(v)), nil
	case Str:
		return string(v), nil
	case RawStr:
		return string(v), nil
	}
	return "", NewArgumentTypeError("1st", "char|str", c.Args.Get(0).Type().Name())
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

// truncateTimeUnit lower-truncates t to the start of the unit named by unit:
// 'y' year, 'M' month, 'w' week (Monday), 'd' day, 'h' hour, 'm' minute,
// 's' second, "ms" millisecond, "us"/"µs" microsecond, "ns" nanosecond.
func truncateTimeUnit(t time.Time, unit string) (time.Time, error) {
	y, mo, d := t.Date()
	loc := t.Location()
	hh, mm, ss := t.Hour(), t.Minute(), t.Second()
	switch unit {
	case "y":
		return time.Date(y, 1, 1, 0, 0, 0, 0, loc), nil
	case "M":
		return time.Date(y, mo, 1, 0, 0, 0, 0, loc), nil
	case "w":
		off := (int(t.Weekday()) + 6) % 7 // days since Monday
		return time.Date(y, mo, d, 0, 0, 0, 0, loc).AddDate(0, 0, -off), nil
	case "d":
		return time.Date(y, mo, d, 0, 0, 0, 0, loc), nil
	case "h":
		return time.Date(y, mo, d, hh, 0, 0, 0, loc), nil
	case "m":
		return time.Date(y, mo, d, hh, mm, 0, 0, loc), nil
	case "s":
		return time.Date(y, mo, d, hh, mm, ss, 0, loc), nil
	case "ms":
		return time.Date(y, mo, d, hh, mm, ss, t.Nanosecond()/1e6*1e6, loc), nil
	case "us", "µs", "μs":
		return time.Date(y, mo, d, hh, mm, ss, t.Nanosecond()/1e3*1e3, loc), nil
	case "ns":
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid truncate unit %q (want y, M, w, d, h, m, s, ms, us or ns)", unit)
}

// advanceTimeUnit returns the start of the unit immediately after floor, which
// must already be unit-aligned (the output of truncateTimeUnit).
func advanceTimeUnit(floor time.Time, unit string) time.Time {
	switch unit {
	case "y":
		return floor.AddDate(1, 0, 0)
	case "M":
		return floor.AddDate(0, 1, 0)
	case "w":
		return floor.AddDate(0, 0, 7)
	case "d":
		return floor.AddDate(0, 0, 1)
	case "h":
		return floor.Add(time.Hour)
	case "m":
		return floor.Add(time.Minute)
	case "s":
		return floor.Add(time.Second)
	case "ms":
		return floor.Add(time.Millisecond)
	case "us", "µs", "μs":
		return floor.Add(time.Microsecond)
	}
	return floor // "ns": already exact
}

// roundTimeUnit rounds t to the nearest unit boundary (a tie rounds up). It
// honours variable-length units (year, month, week) by measuring the real gap
// to the next boundary.
func roundTimeUnit(t time.Time, unit string) (time.Time, error) {
	floor, err := truncateTimeUnit(t, unit)
	if err != nil {
		return time.Time{}, err
	}
	next := advanceTimeUnit(floor, unit)
	if next.Equal(floor) {
		return floor, nil
	}
	mid := floor.Add(next.Sub(floor) / 2)
	if t.Before(mid) {
		return floor, nil
	}
	return next, nil
}

// roundDurationUnit rounds a duration to the nearest whole multiple of the
// fixed-length unit ("w", "d", "h", "m", "s", "ms", "us"/"µs", "ns"). The
// calendar units 'y' and 'M' have no fixed length and are rejected.
func roundDurationUnit(d time.Duration, unit string) (time.Duration, error) {
	switch unit {
	case "w":
		return d.Round(7 * 24 * time.Hour), nil
	case "d":
		return d.Round(24 * time.Hour), nil
	case "h":
		return d.Round(time.Hour), nil
	case "m":
		return d.Round(time.Minute), nil
	case "s":
		return d.Round(time.Second), nil
	case "ms":
		return d.Round(time.Millisecond), nil
	case "us", "µs", "μs":
		return d.Round(time.Microsecond), nil
	case "ns":
		return d, nil
	}
	return 0, fmt.Errorf("invalid round unit %q (want w, d, h, m, s, ms, us or ns)", unit)
}

// truncateDurationUnit truncates a duration toward zero to a whole multiple of
// the fixed-length unit named by unit ("w", "d", "h", "m", "s", "ms",
// "us"/"µs", "ns"). The calendar units 'y' and 'M' have no fixed length and are
// rejected.
func truncateDurationUnit(d time.Duration, unit string) (time.Duration, error) {
	switch unit {
	case "w":
		return d.Truncate(7 * 24 * time.Hour), nil
	case "d":
		return d.Truncate(24 * time.Hour), nil
	case "h":
		return d.Truncate(time.Hour), nil
	case "m":
		return d.Truncate(time.Minute), nil
	case "s":
		return d.Truncate(time.Second), nil
	case "ms":
		return d.Truncate(time.Millisecond), nil
	case "us", "µs", "μs":
		return d.Truncate(time.Microsecond), nil
	case "ns":
		return d, nil
	}
	return 0, fmt.Errorf("invalid truncate unit %q (want w, d, h, m, s, ms, us or ns)", unit)
}
