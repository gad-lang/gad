// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gad-lang/gad/parser/node"
)

// dateTimeLitObject folds a digit-suffix date/time literal body into its
// constant Object — a Date (`D`), or a *Time for the `T` (calendar) and `U`
// (unix seconds) forms.
func dateTimeLitObject(kind node.DateTimeLitKind, body string) (Object, error) {
	switch kind {
	case node.DateLitKind:
		d, err := strToDate(body)
		if err != nil {
			return nil, err
		}
		return d, nil
	case node.UnixTimeLitKind:
		t, err := strToUnixTime(body)
		if err != nil {
			return nil, err
		}
		return &Time{Value: t}, nil
	default:
		t, err := strToTime(body)
		if err != nil {
			return nil, err
		}
		return &Time{Value: t}, nil
	}
}

// timeStrToFunc adapts a string parser into a builtin function that takes one
// str argument and wraps a parse failure as ErrType.
func timeStrToFunc(parse func(string) (Object, error)) CallableFunc {
	return func(c Call) (Object, error) {
		arg := &Arg{Name: "s", TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr)}
		if err := c.Args.Destructure(arg); err != nil {
			return Nil, err
		}
		o, err := parse(arg.Value.ToString())
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return o, nil
	}
}

// locationNew is the Location(...) constructor: a Location pass-through, a
// string (offset/name, see strToLocation) or an int offset in seconds.
func locationNew(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case *Location:
		return v, nil
	case Str:
		loc, err := strToLocation(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return &Location{Value: loc}, nil
	case RawStr:
		loc, err := strToLocation(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return &Location{Value: loc}, nil
	case Int:
		return &Location{Value: time.FixedZone(fmt.Sprintf("%+05d", int(v)/36), int(v))}, nil
	}
	return Nil, NewArgumentTypeError("1st", "str|int", c.Args.Get(0).Type().Name())
}

// timeNew is the time(...) constructor: a time pass-through, a string (see
// strToTime), a Date (midnight UTC) or an int (unix seconds).
func timeNew(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}
	switch v := c.Args.Get(0).(type) {
	case *Time:
		return v, nil
	case Str:
		t, err := strToTime(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return &Time{Value: t}, nil
	case RawStr:
		t, err := strToTime(string(v))
		if err != nil {
			return Nil, ErrType.NewError(err.Error())
		}
		return &Time{Value: t}, nil
	case Date:
		return &Time{Value: v.Time(time.UTC)}, nil
	case Int:
		return &Time{Value: time.Unix(int64(v), 0).UTC()}, nil
	case Uint:
		return &Time{Value: time.Unix(int64(v), 0).UTC()}, nil
	}
	return Nil, NewArgumentTypeError("1st", "str|date|int", c.Args.Get(0).Type().Name())
}

// strToLocation parses a time-zone location from either an offset
// (`-0300`, `-03:00`, `+0530`, `Z`/`UTC`) or an IANA name (`America/Sao_Paulo`).
// A short upper-case token that is not a known name becomes a fixed zero-offset
// zone carrying that label (e.g. `GRU`).
func strToLocation(s string) (*time.Location, error) {
	switch s {
	case "", "Z", "UTC", "utc":
		return time.UTC, nil
	}

	// numeric offset: [+-]HHMM or [+-]HH:MM
	if c := s[0]; c == '+' || c == '-' {
		sign := 1
		if c == '-' {
			sign = -1
		}
		body := strings.Replace(s[1:], ":", "", 1)
		if len(body) == 4 {
			h, herr := strconv.Atoi(body[:2])
			m, merr := strconv.Atoi(body[2:])
			if herr == nil && merr == nil {
				return time.FixedZone(s, sign*(h*3600+m*60)), nil
			}
		}
		return nil, fmt.Errorf("invalid location offset %q", s)
	}

	if loc, err := time.LoadLocation(s); err == nil {
		return loc, nil
	}
	// an unknown short name (e.g. an airport code): keep it as a labelled zone.
	return time.FixedZone(s, 0), nil
}

// strToUnixTime parses a unix-timestamp literal (without the trailing `U`):
// whole seconds (`1781609136`) or a fractional part interpreted by its digit
// count — 3 → milli, 6 → micro, 9 → nano.
func strToUnixTime(s string) (time.Time, error) {
	sec, frac, hasFrac := strings.Cut(s, ".")
	secs, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid unix time %q", s)
	}
	var nsec int64
	if hasFrac {
		switch len(frac) {
		case 3, 6, 9:
		default:
			return time.Time{}, fmt.Errorf("unix time fraction %q must have 3, 6 or 9 digits", frac)
		}
		f, ferr := strconv.ParseInt(frac, 10, 64)
		if ferr != nil {
			return time.Time{}, fmt.Errorf("invalid unix time fraction %q", frac)
		}
		nsec = f * pow10(9-len(frac))
	}
	return time.Unix(secs, nsec).UTC(), nil
}

func pow10(n int) int64 {
	r := int64(1)
	for ; n > 0; n-- {
		r *= 10
	}
	return r
}

// strToTime parses a date/time literal (with an optional trailing `T`) of the
// form `[YYYYMMDD[_]]HHMMSS[.fraction][Zlocation]`. The date defaults to
// 0001-01-01 when only a time is given, and the location to UTC.
func strToTime(s string) (time.Time, error) {
	body := strings.TrimSuffix(s, "T")

	loc := time.UTC
	if i := strings.IndexByte(body, 'Z'); i >= 0 {
		var err error
		if loc, err = strToLocation(body[i+1:]); err != nil {
			return time.Time{}, err
		}
		body = body[:i]
	}

	// optional fractional seconds
	var nsec int
	if dot := strings.IndexByte(body, '.'); dot >= 0 {
		frac := body[dot+1:]
		switch len(frac) {
		case 3, 6, 9:
		default:
			return time.Time{}, fmt.Errorf("time fraction %q must have 3, 6 or 9 digits", frac)
		}
		f, err := strconv.Atoi(frac)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid time fraction %q", frac)
		}
		nsec = f * int(pow10(9-len(frac)))
		body = body[:dot]
	}

	// split optional date from the time
	datePart, timePart := "", body
	if u := strings.IndexByte(body, '_'); u >= 0 {
		datePart, timePart = body[:u], body[u+1:]
	} else if len(body) == 14 {
		datePart, timePart = body[:8], body[8:]
	} else if len(body) == 8 {
		datePart, timePart = body, ""
	}

	year, month, day := 1, 1, 1
	if datePart != "" {
		if len(datePart) != 8 {
			return time.Time{}, fmt.Errorf("invalid date %q (want YYYYMMDD)", datePart)
		}
		var err error
		if year, err = strconv.Atoi(datePart[:4]); err != nil {
			return time.Time{}, err
		}
		if month, err = strconv.Atoi(datePart[4:6]); err != nil {
			return time.Time{}, err
		}
		if day, err = strconv.Atoi(datePart[6:8]); err != nil {
			return time.Time{}, err
		}
	}

	hour, min, sec := 0, 0, 0
	if timePart != "" {
		if len(timePart) != 6 {
			return time.Time{}, fmt.Errorf("invalid time %q (want HHMMSS)", timePart)
		}
		var err error
		if hour, err = strconv.Atoi(timePart[:2]); err != nil {
			return time.Time{}, err
		}
		if min, err = strconv.Atoi(timePart[2:4]); err != nil {
			return time.Time{}, err
		}
		if sec, err = strconv.Atoi(timePart[4:6]); err != nil {
			return time.Time{}, err
		}
	}

	return time.Date(year, time.Month(month), day, hour, min, sec, nsec, loc), nil
}
