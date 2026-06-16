package time

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad"
)

func TestModuleTypes(t *testing.T) {
	l := &Location{Value: time.UTC}
	require.Equal(t, "Location", l.Type().Name())
	require.False(t, l.IsFalsy())
	require.Equal(t, "UTC", l.ToString())
	require.True(t, (&Location{}).Equal(&Location{}))
	require.True(t, (&Location{}).Equal(gad.Str("UTC")))
	require.False(t, (&Location{}).Equal(gad.Int(0)))

	tm := &Time{}
	require.Equal(t, "time", tm.Type().Name())
	require.True(t, tm.IsFalsy())
	require.NotEmpty(t, tm.ToString())
	require.True(t, tm.Equal(&Time{}))
	require.False(t, tm.Equal(gad.Int(0)))
	r, err := tm.IndexGet(nil, gad.Str(""))
	require.NoError(t, err)
	require.Equal(t, gad.Nil, r)

	now := time.Now()
	tm2 := &Time{Value: now}
	require.False(t, tm2.IsFalsy())
	require.Equal(t, now.String(), tm2.ToString())

	var b bytes.Buffer
	err = gob.NewEncoder(&b).Encode(tm2)
	require.NoError(t, err)
	var tm3 Time
	err = gob.NewDecoder(&b).Decode(&tm3)
	require.NoError(t, err)
	require.Equal(t, tm2.Value.Format(time.RFC3339Nano),
		tm3.Value.Format(time.RFC3339Nano))
}

// lcFirst lowercases the first byte (ASCII) of s, matching the camelCase keys
// of the time module (e.g. Go's "January" -> the module key "january").
func lcFirst(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]|0x20) + s[1:]
}

func TestModuleMonthWeekday(t *testing.T) {
	module := getModule()
	f := module["monthString"].(gad.CallerObject)
	_, err := gad.MustCall(f)
	require.Error(t, err)
	_, err = gad.MustCall(f, gad.Str(""))
	require.Error(t, err)

	for i := 1; i <= 12; i++ {
		require.Contains(t, module, lcFirst(time.Month(i).String()))
		require.Equal(t, gad.Int(i), module[lcFirst(time.Month(i).String())])

		r, err := gad.MustCall(f, gad.Int(i))
		require.NoError(t, err)
		require.EqualValues(t, time.Month(i).String(), r)
	}

	f = module["weekdayString"].(gad.CallerObject)
	_, err = gad.MustCall(f)
	require.Error(t, err)
	_, err = gad.MustCall(f, gad.Str(""))
	require.Error(t, err)
	for i := 0; i <= 6; i++ {
		require.Contains(t, module, lcFirst(time.Weekday(i).String()))
		require.Equal(t, gad.Int(i), module[lcFirst(time.Weekday(i).String())])

		r, err := gad.MustCall(f, gad.Int(i))
		require.NoError(t, err)
		require.EqualValues(t, time.Weekday(i).String(), r)
	}
}

func TestModuleFormats(t *testing.T) {
	var module = getModule()
	require.Equal(t, module["ansic"], gad.Str(time.ANSIC))
	require.Equal(t, module["unixDate"], gad.Str(time.UnixDate))
	require.Equal(t, module["rubyDate"], gad.Str(time.RubyDate))
	require.Equal(t, module["rfc822"], gad.Str(time.RFC822))
	require.Equal(t, module["rfc822Z"], gad.Str(time.RFC822Z))
	require.Equal(t, module["rfc850"], gad.Str(time.RFC850))
	require.Equal(t, module["rfc1123"], gad.Str(time.RFC1123))
	require.Equal(t, module["rfc1123Z"], gad.Str(time.RFC1123Z))
	require.Equal(t, module["rfc3339"], gad.Str(time.RFC3339))
	require.Equal(t, module["rfc3339Nano"], gad.Str(time.RFC3339Nano))
	require.Equal(t, module["kitchen"], gad.Str(time.Kitchen))
	require.Equal(t, module["stamp"], gad.Str(time.Stamp))
	require.Equal(t, module["stampMilli"], gad.Str(time.StampMilli))
	require.Equal(t, module["stampMicro"], gad.Str(time.StampMicro))
	require.Equal(t, module["stampNano"], gad.Str(time.StampNano))
}

func TestModuleDuration(t *testing.T) {
	var module = getModule()
	require.Equal(t, module["nanosecond"], gad.Int(time.Nanosecond))
	require.Equal(t, module["microsecond"], gad.Int(time.Microsecond))
	require.Equal(t, module["millisecond"], gad.Int(time.Millisecond))
	require.Equal(t, module["second"], gad.Int(time.Second))
	require.Equal(t, module["minute"], gad.Int(time.Minute))
	require.Equal(t, module["hour"], gad.Int(time.Hour))

	goFnMap := map[string]func(time.Duration) any{
		"Nanoseconds": func(d time.Duration) any {
			return d.Nanoseconds()
		},
		"Microseconds": func(d time.Duration) any {
			return d.Microseconds()
		},
		"Milliseconds": func(d time.Duration) any {
			return d.Milliseconds()
		},
		"Seconds": func(d time.Duration) any {
			return d.Seconds()
		},
		"Minutes": func(d time.Duration) any {
			return d.Minutes()
		},
		"Hours": func(d time.Duration) any {
			return d.Hours()
		},
	}
	durToString := module["durationString"].(gad.CallerObject)
	_, err := gad.MustCall(durToString)
	require.Error(t, err)

	durParse := module["parseDuration"].(gad.CallerObject)
	_, err = gad.MustCall(durParse)
	require.Error(t, err)
	_, err = gad.MustCall(durParse, gad.Str(""))
	require.Error(t, err)
	_, err = gad.MustCall(durParse, gad.Int(0))
	require.NoError(t, err)

	testCases := []struct {
		dur time.Duration
	}{
		{time.Nanosecond}, {time.Microsecond}, {time.Millisecond}, {time.Second},
		{time.Minute}, {time.Hour},
		{time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond + time.Nanosecond},
		{2*time.Hour + 3*time.Minute + 4*time.Second + 5*time.Millisecond + 6*time.Microsecond + 7*time.Nanosecond},
		{-2*time.Hour + 3*time.Minute + 4*time.Second + 5*time.Millisecond + 6*time.Microsecond + 7*time.Nanosecond},
	}

	for _, tC := range testCases {
		for fn := range goFnMap {
			t.Run(fmt.Sprintf("%s:%s", tC.dur, fn), func(t *testing.T) {
				f := module["duration"+fn].(gad.CallerObject)
				ret, err := gad.MustCall(f, gad.Int(tC.dur))
				require.NoError(t, err)
				expect := goFnMap[fn](tC.dur)
				require.EqualValues(t, expect, ret)

				// test illegal type
				_, err = gad.MustCall(f, &illegalDur{Value: tC.dur})
				require.Error(t, err)
				// test no arg
				_, err = gad.MustCall(f)
				require.Error(t, err)

				// test to string
				s, err := gad.MustCall(durToString, gad.Int(tC.dur))
				require.NoError(t, err)
				require.EqualValues(t, tC.dur.String(), s)

				// test parse
				d, err := gad.MustCall(durParse, s)
				require.NoError(t, err)
				ed, err := time.ParseDuration(tC.dur.String())
				require.NoError(t, err)
				require.EqualValues(t, ed, d)
			})
		}
	}

	durRound := module["durationRound"].(gad.CallerObject)
	r, err := gad.MustCall(durRound, gad.Int(time.Second+time.Millisecond),
		gad.Int(time.Second))
	require.NoError(t, err)
	require.EqualValues(t, time.Second, r)
	_, err = gad.MustCall(durRound, gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(durRound, gad.Str(""), gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(durRound, gad.Int(0), gad.Str(""))
	require.Error(t, err)

	durTruncate := module["durationTruncate"].(gad.CallerObject)
	r, err = gad.MustCall(durTruncate, gad.Int(time.Second+5*time.Millisecond),
		gad.Int(2*time.Millisecond))
	require.NoError(t, err)
	require.EqualValues(t, time.Second+4*time.Millisecond, r)
	_, err = gad.MustCall(durTruncate, gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(durTruncate, gad.Str(""), gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(durTruncate, gad.Int(0), gad.Str(""))
	require.Error(t, err)
}

func TestModuleLocation(t *testing.T) {
	var module = getModule()
	fixedZone := module["fixedZone"].(gad.CallerObject)
	r, err := gad.MustCall(fixedZone, gad.Str("Ankara"), gad.Int(3*60*60))
	require.NoError(t, err)
	require.Equal(t, "Ankara", r.ToString())

	_, err = gad.MustCall(fixedZone, gad.Str("Ankara"))
	require.Error(t, err)
	_, err = gad.MustCall(fixedZone, gad.Str("Ankara"), gad.Uint(0))
	require.NoError(t, err)
	_, err = gad.MustCall(fixedZone, gad.Int(0), gad.Array{})
	require.Error(t, err)
	_, err = gad.MustCall(fixedZone)
	require.Error(t, err)

	loadLocation := module["loadLocation"].(gad.CallerObject)
	r, err = gad.MustCall(loadLocation, gad.Str("Europe/Istanbul"))
	require.NoError(t, err)
	require.Equal(t, "Europe/Istanbul", r.ToString())
	r, err = gad.MustCall(loadLocation, gad.Str(""))
	require.NoError(t, err)
	require.Equal(t, "UTC", r.ToString())
	_, err = gad.MustCall(loadLocation)
	require.Error(t, err)
	_, err = gad.MustCall(loadLocation, gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(loadLocation, gad.Str("invalid"))
	require.Error(t, err)

	isLocation := module["isLocation"].(gad.CallerObject)
	r, err = gad.MustCall(isLocation, &Location{Value: time.Local})
	require.NoError(t, err)
	require.EqualValues(t, true, r)
	r, err = gad.MustCall(isLocation, gad.Int(0))
	require.NoError(t, err)
	require.EqualValues(t, false, r)
	_, err = gad.MustCall(isLocation, gad.Int(0), gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(isLocation)
	require.Error(t, err)
}

func TestModuleTime(t *testing.T) {
	var module = getModule()
	now := time.Now()

	require.Equal(t, now.String(), (&Time{Value: now}).ToString())

	zTime := module["time"].(gad.CallerObject)
	r, err := gad.MustCall(zTime)
	require.NoError(t, err)
	require.True(t, r.(*Time).Value.IsZero())
	_, err = gad.MustCall(zTime, gad.Str(""))
	require.Error(t, err)

	since := module["since"].(gad.CallerObject)
	r, err = gad.MustCall(since, &Time{Value: now})
	require.NoError(t, err)
	require.GreaterOrEqual(t, int64(r.(gad.Int)), int64(0))
	_, err = gad.MustCall(since)
	require.Error(t, err)
	_, err = gad.MustCall(since, gad.Str(""))
	require.Error(t, err)

	until := module["until"].(gad.CallerObject)
	r, err = gad.MustCall(until, &Time{Value: now})
	require.NoError(t, err)
	require.LessOrEqual(t, int64(r.(gad.Int)), int64(0))
	_, err = gad.MustCall(until)
	require.Error(t, err)
	_, err = gad.MustCall(until, gad.Str(""))
	require.Error(t, err)

	date := module["date"].(gad.CallerObject)
	r, err = gad.MustCall(date, gad.Int(2020), gad.Int(11), gad.Int(8),
		gad.Int(1), gad.Int(2), gad.Int(3), gad.Int(4),
		&Location{Value: time.Local})
	require.NoError(t, err)
	require.Equal(t,
		time.Date(2020, 11, 8, 1, 2, 3, 4, time.Local), r.(*Time).Value)
	r, err = gad.MustCall(date, gad.Int(2020), gad.Int(11), gad.Int(8))
	require.NoError(t, err)
	require.Equal(t,
		time.Date(2020, 11, 8, 0, 0, 0, 0, time.Local), r.(*Time).Value)

	nowf := module["now"].(gad.CallerObject)
	r, err = gad.MustCall(nowf)
	require.NoError(t, err)
	require.False(t, r.(*Time).Value.IsZero())
	_, err = gad.MustCall(nowf, gad.Int(0))
	require.Error(t, err)

	RFC3339Nano := module["rfc3339Nano"]
	parse := module["parse"].(gad.CallerObject)
	r, err = gad.MustCall(parse, RFC3339Nano, gad.Str(now.Format(time.RFC3339Nano)))
	require.NoError(t, err)
	require.Equal(t, now.Format(time.RFC3339Nano),
		r.(*Time).Value.Format(time.RFC3339Nano))

	r, err = gad.MustCall(parse, RFC3339Nano, gad.Str(now.Format(time.RFC3339Nano)),
		&Location{Value: time.Local})
	require.NoError(t, err)
	require.Equal(t, now.Format(time.RFC3339Nano),
		r.(*Time).Value.Format(time.RFC3339Nano))

	_, err = gad.MustCall(parse)
	require.Error(t, err)

	unix := module["unix"].(gad.CallerObject)
	r, err = gad.MustCall(unix, gad.Int(now.Unix()))
	require.NoError(t, err)
	require.Equal(t, time.Unix(now.Unix(), 0), r.(*Time).Value)
	r, err = gad.MustCall(unix, gad.Int(now.Unix()), gad.Int(1))
	require.NoError(t, err)
	require.Equal(t, time.Unix(now.Unix(), 1), r.(*Time).Value)
	_, err = gad.MustCall(unix)
	require.Error(t, err)

	add := module["add"].(gad.CallerObject)
	r, err = gad.MustCall(add, &Time{Value: now}, gad.Int(time.Second))
	require.NoError(t, err)
	require.Equal(t, now.Add(time.Second), r.(*Time).Value)
	_, err = gad.MustCall(add, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(add, &Time{Value: now}, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(add)
	require.Error(t, err)

	sub := module["sub"].(gad.CallerObject)
	r, err = gad.MustCall(sub, &Time{Value: now}, &Time{Value: now.Add(-time.Hour)})
	require.NoError(t, err)
	require.EqualValues(t, time.Hour, r.(gad.Int))
	_, err = gad.MustCall(sub, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(sub, &Time{Value: now}, gad.Int(0))
	require.NoError(t, err)
	_, err = gad.MustCall(sub)
	require.Error(t, err)

	addDate := module["addDate"].(gad.CallerObject)
	r, err = gad.MustCall(addDate, &Time{Value: now},
		gad.Int(1), gad.Int(2), gad.Int(3))
	require.NoError(t, err)
	require.EqualValues(t, now.AddDate(1, 2, 3), r.(*Time).Value)
	_, err = gad.MustCall(addDate, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(addDate, &Time{Value: now}, gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(addDate)
	require.Error(t, err)

	after := module["after"].(gad.CallerObject)
	r, err = gad.MustCall(after, &Time{Value: now}, &Time{Value: now.Add(time.Hour)})
	require.NoError(t, err)
	require.EqualValues(t, false, r)
	r, err = gad.MustCall(after, &Time{Value: now}, &Time{Value: now.Add(-time.Hour)})
	require.NoError(t, err)
	require.EqualValues(t, true, r)
	_, err = gad.MustCall(after, &Time{Value: now}, gad.Int(0))
	require.NoError(t, err)
	_, err = gad.MustCall(after, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(after)
	require.Error(t, err)

	before := module["before"].(gad.CallerObject)
	r, err = gad.MustCall(before, &Time{Value: now}, &Time{Value: now.Add(time.Hour)})
	require.NoError(t, err)
	require.EqualValues(t, true, r)
	r, err = gad.MustCall(before, &Time{Value: now}, &Time{Value: now.Add(-time.Hour)})
	require.NoError(t, err)
	require.EqualValues(t, false, r)
	_, err = gad.MustCall(before, &Time{Value: now}, gad.Int(0))
	require.NoError(t, err)
	_, err = gad.MustCall(before, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(before)
	require.Error(t, err)

	appendFormat := module["appendFormat"].(gad.CallerObject)
	b := make(gad.Bytes, 100)
	r, err = gad.MustCall(appendFormat, &Time{Value: now}, b, RFC3339Nano)
	require.NoError(t, err)
	require.EqualValues(t,
		now.AppendFormat(make([]byte, 100), time.RFC3339Nano), r)
	_, err = gad.MustCall(appendFormat, &Time{Value: now}, b)
	require.Error(t, err)
	_, err = gad.MustCall(appendFormat, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(appendFormat)
	require.Error(t, err)

	format := module["format"].(gad.CallerObject)
	r, err = gad.MustCall(format, &Time{Value: now}, RFC3339Nano)
	require.NoError(t, err)
	require.EqualValues(t, now.Format(time.RFC3339Nano), r)
	_, err = gad.MustCall(format, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(format)
	require.Error(t, err)

	timeIn := module["in"].(gad.CallerObject)
	r, err = gad.MustCall(timeIn, &Time{Value: now}, &Location{Value: time.Local})
	require.NoError(t, err)
	require.False(t, r.(*Time).Value.IsZero())
	_, err = gad.MustCall(timeIn, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(timeIn)
	require.Error(t, err)

	round := module["round"].(gad.CallerObject)
	r, err = gad.MustCall(round, &Time{Value: now}, gad.Int(time.Second))
	require.NoError(t, err)
	require.Equal(t, now.Round(time.Second), r.(*Time).Value)
	_, err = gad.MustCall(round, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(round)
	require.Error(t, err)

	truncate := module["truncate"].(gad.CallerObject)
	r, err = gad.MustCall(truncate, &Time{Value: now}, gad.Int(time.Hour))
	require.NoError(t, err)
	require.Equal(t, now.Truncate(time.Hour), r.(*Time).Value)
	_, err = gad.MustCall(truncate, &Time{Value: now})
	require.Error(t, err)
	_, err = gad.MustCall(truncate)
	require.Error(t, err)

	isTime := module["isTime"].(gad.CallerObject)
	r, err = gad.MustCall(isTime, &Time{Value: now})
	require.NoError(t, err)
	require.EqualValues(t, true, r)
	r, err = gad.MustCall(isTime, gad.Int(0))
	require.NoError(t, err)
	require.EqualValues(t, false, r)
	_, err = gad.MustCall(isTime, gad.Int(0), gad.Int(0))
	require.Error(t, err)
	_, err = gad.MustCall(isTime)
	require.Error(t, err)

	y, m, d := now.Date()
	testTimeSelector(t, &Time{Value: now}, "Date",
		gad.Dict{"year": gad.Int(y), "month": gad.Int(m), "day": gad.Int(d)})
	h, min, s := now.Clock()
	testTimeSelector(t, &Time{Value: now}, "Clock",
		gad.Dict{"hour": gad.Int(h), "minute": gad.Int(min), "second": gad.Int(s)})
	testTimeSelector(t, &Time{Value: now}, "UTC", &Time{Value: now.UTC()})
	testTimeSelector(t, &Time{Value: now}, "Unix", gad.Int(now.Unix()))
	testTimeSelector(t, &Time{Value: now}, "UnixNano", gad.Int(now.UnixNano()))
	testTimeSelector(t, &Time{Value: now}, "Year", gad.Int(now.Year()))
	testTimeSelector(t, &Time{Value: now}, "Month", gad.Int(now.Month()))
	testTimeSelector(t, &Time{Value: now}, "Day", gad.Int(now.Day()))
	testTimeSelector(t, &Time{Value: now}, "Hour", gad.Int(now.Hour()))
	testTimeSelector(t, &Time{Value: now}, "Minute", gad.Int(now.Minute()))
	testTimeSelector(t, &Time{Value: now}, "Second", gad.Int(now.Second()))
	testTimeSelector(t, &Time{Value: now}, "Nanosecond", gad.Int(now.Nanosecond()))
	testTimeSelector(t, &Time{Value: now}, "IsZero", gad.Bool(false))
	testTimeSelector(t, &Time{Value: now}, "Local", &Time{Value: now.Local()})
	testTimeSelector(t, &Time{Value: now}, "Location",
		&Location{Value: now.Location()})
	testTimeSelector(t, &Time{Value: now}, "YearDay", gad.Int(now.YearDay()))
	testTimeSelector(t, &Time{Value: now}, "Weekday", gad.Int(now.Weekday()))
	y, w := now.ISOWeek()
	testTimeSelector(t, &Time{Value: now}, "ISOWeek",
		gad.Dict{"year": gad.Int(y), "week": gad.Int(w)})
	name, offset := now.Zone()
	testTimeSelector(t, &Time{Value: now}, "Zone",
		gad.Dict{"name": gad.Str(name), "offset": gad.Int(offset)})
	testTimeSelector(t, &Time{Value: now}, "XYZ", gad.Nil)
}

func testTimeSelector(t *testing.T, tm gad.Object,
	selector string, expected gad.Object) {
	t.Helper()
	v, err := gad.Val(tm.(gad.IndexGetter).IndexGet(nil, gad.Str(selector)))
	require.NoError(t, err)
	require.Equal(t, expected, v)
}

func TestScript(t *testing.T) {
	catch := func(s string) string {
		return fmt.Sprintf(`
		time := import("time")
		try {
			return %s
		} catch err {
			return str(err.cause)
		}
		`, s)
	}
	idxTypeErr := func(expected, got string) gad.Str {
		return gad.Str(gad.NewIndexTypeError(expected, got).ToString())
	}
	opTypeErr := func(tok, lhs, rhs string) gad.Str {
		return gad.Str(gad.NewOperandTypeError(
			tok, lhs, rhs).ToString())
	}
	typeErr := func(pos, expected, got string) gad.Str {
		return gad.Str(gad.NewArgumentTypeError(pos, expected, got).ToString())
	}
	nwrongArgs := func(want1, want2, got int) gad.Str {
		var msg string
		if want2 <= 0 {
			msg = fmt.Sprintf("want=%d got=%d", want1, got)
		} else {
			msg = fmt.Sprintf("want=%d..%d got=%d", want1, want2, got)
		}
		return gad.Str(gad.ErrWrongNumArguments.NewError(msg).ToString())
	}

	expectRun(t, `import("time")`, nil, gad.Nil)

	// test registers
	// time
	now := time.Now()

	var s = struct {
		T  time.Time
		Tp *time.Time
		L  *time.Location
		D  time.Duration
	}{
		T:  now,
		Tp: &now,
		L:  time.UTC,
		D:  time.Hour,
	}

	typeOf := func(t *testing.T, field string, typ gad.ObjectType, str string, ptr bool) {
		t.Helper()
		vm, ret, err := runV(fmt.Sprintf(`import("time");param v; return v.%s`, field), &gad.RunOpts{Args: gad.Args{{gad.MustToObject(&s)}}})
		require.NoError(t, err)
		require.Equal(t, typ, ret.Type())
		v := reflect.ValueOf(s).FieldByName(field)
		if ptr {
			v = v.Elem()
		}
		vi := v.Interface()
		require.Equal(t, str, ret.ToString())
		require.Equal(t, vi, vm.ToInterface(ret))
	}

	typeOf(t, "T", TimeType, now.String(), false)
	typeOf(t, "Tp", TimeType, now.String(), true)
	typeOf(t, "L", LocationType, "UTC", false)

	expectRun(t, catch(`time.now()[1]`),
		nil, idxTypeErr("str", "int"))
	expectRun(t, catch(`time.now() + 'c'`),
		nil, opTypeErr("+", "time", "char"))
	expectRun(t, catch(`time.now()()`), nil, gad.Str("NotCallableError: time"))
	expectRun(t, catch(`time.date()`), nil, nwrongArgs(3, 8, 0))
	expectRun(t, catch(`time.date(1)`), nil, nwrongArgs(3, 8, 1))
	expectRun(t, catch(`time.date(1, 2)`), nil, nwrongArgs(3, 8, 2))
	expectRun(t, catch(`time.date(1, 2, "")`),
		nil, typeErr("3", "int", "str"))
	expectRun(t, catch(`time.date(1, 2, 3, 4, 5, 6, 7, "")`),
		nil, typeErr("8", "Location", "str"))
	expectRun(t, catch(`time.parse("", 1)`),
		nil, gad.Str("ErrCall: parsing time \"1\": extra text: \"1\""))
	expectRun(t, catch(`time.parse("", "", 1)`),
		nil, typeErr("3rd", "Location", "int"))
	expectRun(t, catch(`time.unix("")`),
		nil, typeErr("1st", "int", "str"))
	expectRun(t, catch(`time.unix(1, "")`),
		nil, typeErr("2nd", "int", "str"))
	expectRun(t, catch(`time.addDate(time.now(), "", 1, 2)`),
		nil, typeErr("2nd", "int", "str"))
	expectRun(t, catch(`time.addDate(time.now(), 1, "", 2)`),
		nil, typeErr("3rd", "int", "str"))
	expectRun(t, catch(`time.addDate(time.now(), 1, 2, "")`),
		nil, typeErr("4th", "int", "str"))
	expectRun(t, catch(`time.after(1, 2)`), nil, gad.False)
	expectRun(t, catch(`time.before(1, 2)`), nil, gad.True)
	expectRun(t, catch(`time.appendFormat(1, 2, 3)`),
		nil, typeErr("2nd", "bytes", "int"))
	expectRun(t, catch(`time.appendFormat(time.now(), 1, 2)`),
		nil, typeErr("2nd", "bytes", "int"))
	expectRun(t, catch(`time.appendFormat(time.time(), bytes(), 1)`),
		nil, gad.Bytes{0x31})
	expectRun(t, catch(`time.in(1, 2)`),
		nil, typeErr("2nd", "Location", "int"))
	expectRun(t, catch(`time.in(time.now(), 2)`),
		nil, typeErr("2nd", "Location", "int"))
	expectRun(t, catch(`time.round(time.now(), "")`),
		nil, typeErr("2nd", "int", "str"))
	expectRun(t, catch(`time.truncate(time.now(), "")`),
		nil, typeErr("2nd", "int", "str"))
	expectRun(t, catch(`time.sleep("")`),
		nil, typeErr("1st", "int", "str"))

	expectRun(t, `mod := import("time"); return mod.@name`,
		nil, gad.Str("time"))

	tm := time.Now()
	expectRun(t, `
	param p1; time := import("time"); return time.format(p1, time.rfc3339Nano)`,
		newOpts().Args(&Time{Value: tm}), gad.Str(tm.Format(time.RFC3339Nano)))
	expectRun(t, `param p1; return p1.UnixNano`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.UnixNano()))

	expectRun(t, `
	param p1
	time := import("time")
	try {
		time.sleep(time.millisecond)
	} finally {
		dur := time.since(p1)
		return dur > 0 ? true: false 
	}
	`, newOpts().Args(&Time{Value: tm}), gad.True)

	expectRun(t, `return import("time").isTime(0)`, nil, gad.False)
	expectRun(t, `param p1; time := import("time"); return time.isTime(p1)`,
		newOpts().Args(&Time{Value: tm}), gad.True)
	expectRun(t, `time := import("time"); return time.isTime(time.now())`,
		nil, gad.True)
	expectRun(t, `
	time := import("time")
	return time.isLocation(time.fixedZone("abc", 3*60*60))`, nil, gad.True)
	expectRun(t, `param p1; return p1==p1`,
		newOpts().Args(&Time{Value: tm}), gad.True)
	expectRun(t, `param p1; time := import("time"); return time.now()==p1`,
		newOpts().Args(&Time{Value: tm}), gad.False)
	expectRun(t, `param p1; time := import("time"); return time.now()>=p1`,
		newOpts().Args(&Time{Value: tm}), gad.True)
	expectRun(t, `param p1; time := import("time"); return time.now()<p1`,
		newOpts().Args(&Time{Value: tm}), gad.False)
	expectRun(t, `param p1; time := import("time"); return time.now()>p1`,
		newOpts().Args(&Time{Value: tm}), gad.True)
	expectRun(t, `time := import("time"); return (time.now()+time.second)>=time.now()`, nil, gad.True)
	expectRun(t, `time := import("time"); return (time.now()+time.second)<=time.now()`, nil, gad.False)
	expectRun(t, `time := import("time"); return (time.now()-10*time.second)<=time.now()`, nil, gad.True)
	expectRun(t, `time := import("time"); return time.now() == nil`, nil, gad.False)
	expectRun(t, `time := import("time"); return time.now() > nil`, nil, gad.True)
	expectRun(t, `time := import("time"); return time.now() >= nil`, nil, gad.True)
	expectRun(t, `time := import("time"); return time.now() < nil`, nil, gad.False)
	expectRun(t, `time := import("time"); return time.now() <= nil`, nil, gad.False)
	expectRun(t, `
	time := import("time")
	t1 := time.now()
	t2 := t1 + time.second
	return t2 - t1
	`, nil, gad.Int(time.Second))

	// methods
	// .Add
	expectRun(t, `time := import("time"); return time.time().Add(10*time.second)`,
		nil, &Time{Value: time.Time{}.Add(10 * time.Second)})
	expectRun(t, catch(`time.time().Add()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Add(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Add(nil)`), nil, typeErr("1st", "int", "nil"))

	// .Sub
	expectRun(t, `time := import("time");
	t1 := time.time()
	t2 := time.time().Add(10*time.second)
	return t2.Sub(t1)`,
		nil, gad.Int(10*time.Second))
	expectRun(t, catch(`time.time().Sub()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Sub(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Sub(nil)`), nil, typeErr("1st", "time", "nil"))

	// .AddDate
	expectRun(t, `time := import("time"); return time.time().AddDate(1, 2, 3)`,
		nil, &Time{Value: time.Time{}.AddDate(1, 2, 3)})
	expectRun(t, catch(`time.time().AddDate()`), nil, nwrongArgs(3, -1, 0))
	expectRun(t, catch(`time.time().AddDate(1, 2)`), nil, nwrongArgs(3, -1, 2))
	expectRun(t, catch(`time.time().AddDate(1, 2, 3, 4)`), nil, nwrongArgs(3, -1, 4))
	expectRun(t, catch(`time.time().AddDate(nil, 2, 3)`), nil, typeErr("1st", "int", "nil"))

	// .After
	expectRun(t, `time := import("time"); return time.time().After(time.time())`, nil, gad.False)
	expectRun(t, `time := import("time"); return time.time().Add(time.second).After(time.time())`, nil, gad.True)
	expectRun(t, catch(`time.time().After()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().After(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().After(nil)`), nil, typeErr("1st", "time", "nil"))

	// .Before
	expectRun(t, `time := import("time"); return time.time().Before(time.time())`, nil, gad.False)
	expectRun(t, `time := import("time"); return time.time().Add(-time.second).Before(time.time())`, nil, gad.True)
	expectRun(t, catch(`time.time().Before()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Before(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Before(nil)`), nil, typeErr("1st", "time", "nil"))

	// .Format
	expectRun(t, `time := import("time"); return time.time().Format("2006-01-02")`, nil, gad.Str("0001-01-01"))
	expectRun(t, catch(`time.time().Format()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Format(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Format(nil)`), nil, typeErr("1st", "str", "nil"))

	// .AppendFormat
	expectRun(t, `time := import("time"); return time.time().AppendFormat("", "2006-01-02")`, nil, gad.Bytes("0001-01-01"))
	expectRun(t, catch(`time.time().AppendFormat()`), nil, nwrongArgs(2, -1, 0))
	expectRun(t, catch(`time.time().AppendFormat(1)`), nil, nwrongArgs(2, -1, 1))
	expectRun(t, catch(`time.time().AppendFormat(1, 2, 3)`), nil, nwrongArgs(2, -1, 3))
	expectRun(t, catch(`time.time().AppendFormat(nil, "2006-01-02")`), nil, typeErr("1st", "bytes", "nil"))

	// .In
	expectRun(t, `param p1; time := import("time"); return p1.In(time.utc())`,
		newOpts().Args(&Time{Value: tm}), &Time{Value: tm.In(time.UTC)})
	expectRun(t, catch(`time.time().In()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().In(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().In(nil)`), nil, typeErr("1st", "Location", "nil"))

	// .Round
	expectRun(t, `param p1; time := import("time"); return p1.Round(time.second)`,
		newOpts().Args(&Time{Value: tm}), &Time{Value: tm.Round(time.Second)})
	expectRun(t, catch(`time.time().Round()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Round(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Round(nil)`), nil, typeErr("1st", "int", "nil"))

	// .Truncate
	expectRun(t, `param p1; time := import("time"); return p1.Truncate(time.second)`,
		newOpts().Args(&Time{Value: tm}), &Time{Value: tm.Truncate(time.Second)})
	expectRun(t, catch(`time.time().Truncate()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Truncate(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Truncate(nil)`), nil, typeErr("1st", "int", "nil"))

	// .Equal
	expectRun(t, `time := import("time"); return time.time().Equal(time.time())`, nil, gad.True)
	expectRun(t, `param (p1,p2); return p1.Equal(p2)`,
		newOpts().Args(&Time{Value: tm}, &Time{Value: tm}), gad.True)
	expectRun(t, `param (p1,p2); return p1.Equal(p2)`,
		newOpts().Args(&Time{Value: tm}, &Time{Value: tm.Add(time.Second)}), gad.False)
	expectRun(t, catch(`time.time().Equal()`), nil, nwrongArgs(1, -1, 0))
	expectRun(t, catch(`time.time().Equal(1, 2)`), nil, nwrongArgs(1, -1, 2))
	expectRun(t, catch(`time.time().Equal(nil)`), nil, typeErr("1st", "time", "nil"))

	// .Date
	expectRun(t, `time := import("time"); return time.time().Date()`,
		nil, gad.Dict{"day": gad.Int(1), "month": gad.Int(1), "year": gad.Int(1)})
	expectRun(t, catch(`time.time().Date(1)`), nil, nwrongArgs(0, -1, 1))

	// .Clock
	hour, minute, second := tm.Clock()
	expectRun(t, `param p1; return p1.Clock()`,
		newOpts().Args(&Time{Value: tm}),
		gad.Dict{"hour": gad.Int(hour), "minute": gad.Int(minute), "second": gad.Int(second)})
	expectRun(t, catch(`time.time().Clock(1)`), nil, nwrongArgs(0, -1, 1))

	// .UTC
	expectRun(t, `param p1; return p1.UTC()`,
		newOpts().Args(&Time{Value: tm}), &Time{Value: tm.UTC()})
	expectRun(t, catch(`time.time().UTC(1)`), nil, nwrongArgs(0, -1, 1))

	// .Unix
	expectRun(t, `param p1; return p1.Unix()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Unix()))
	expectRun(t, catch(`time.time().Unix(1)`), nil, nwrongArgs(0, -1, 1))

	// .UnixNano
	expectRun(t, `param p1; return p1.UnixNano()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.UnixNano()))
	expectRun(t, catch(`time.time().UnixNano(1)`), nil, nwrongArgs(0, -1, 1))

	// .Year
	expectRun(t, `param p1; return p1.Year()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Year()))
	expectRun(t, catch(`time.time().Year(1)`), nil, nwrongArgs(0, -1, 1))

	// .Month
	expectRun(t, `param p1; return p1.Month()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Month()))
	expectRun(t, catch(`time.time().Month(1)`), nil, nwrongArgs(0, -1, 1))

	// .Day
	expectRun(t, `param p1; return p1.Day()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Day()))
	expectRun(t, catch(`time.time().Day(1)`), nil, nwrongArgs(0, -1, 1))

	// .Hour
	expectRun(t, `param p1; return p1.Hour()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Hour()))
	expectRun(t, catch(`time.time().Hour(1)`), nil, nwrongArgs(0, -1, 1))

	// .Minute
	expectRun(t, `param p1; return p1.Minute()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Minute()))
	expectRun(t, catch(`time.time().Minute(1)`), nil, nwrongArgs(0, -1, 1))

	// .Second
	expectRun(t, `param p1; return p1.Second()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Second()))
	expectRun(t, catch(`time.time().Second(1)`), nil, nwrongArgs(0, -1, 1))

	// .Nanosecond
	expectRun(t, `param p1; return p1.Nanosecond()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Nanosecond()))
	expectRun(t, catch(`time.time().Nanosecond(1)`), nil, nwrongArgs(0, -1, 1))

	// .Weekday
	expectRun(t, `param p1; return p1.Weekday()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.Weekday()))
	expectRun(t, catch(`time.time().Weekday(1)`), nil, nwrongArgs(0, -1, 1))

	// .ISOWeek
	year, week := tm.ISOWeek()
	expectRun(t, `param p1; return p1.ISOWeek()`,
		newOpts().Args(&Time{Value: tm}), gad.Dict{"year": gad.Int(year), "week": gad.Int(week)})
	expectRun(t, catch(`time.time().ISOWeek(1)`), nil, nwrongArgs(0, -1, 1))

	// .YearDay
	expectRun(t, `param p1; return p1.YearDay()`,
		newOpts().Args(&Time{Value: tm}), gad.Int(tm.YearDay()))
	expectRun(t, catch(`time.time().YearDay(1)`), nil, nwrongArgs(0, -1, 1))

	// .Location
	expectRun(t, `time := import("time"); return time.time().Location()`, nil, &Location{Value: time.Time{}.Location()})
	expectRun(t, catch(`time.time().Location(1)`), nil, nwrongArgs(0, -1, 1))

	// .Zone
	zone, offset := tm.Zone()
	expectRun(t, `param p1; return p1.Zone()`,
		newOpts().Args(&Time{Value: tm}), gad.Dict{"name": gad.Str(zone), "offset": gad.Int(offset)})
	expectRun(t, catch(`time.time().Zone(1)`), nil, nwrongArgs(0, -1, 1))
}

var IllegalType = gad.NewBuiltinObjType("illegal")

type illegalDur struct {
	gad.ObjectImpl
	Value time.Duration
}

func (*illegalDur) ToString() string     { return "illegal" }
func (*illegalDur) Type() gad.ObjectType { return IllegalType }

type Opts struct {
	global gad.IndexGetSetter
	args   []gad.Object
}

func newOpts() *Opts {
	return &Opts{}
}

func (o *Opts) Args(args ...gad.Object) *Opts {
	o.args = args
	return o
}

func (o *Opts) Globals(g gad.IndexGetSetter) *Opts {
	o.global = g
	return o
}

func expectRun(t *testing.T, script string, opts *Opts, expected gad.Object) {
	t.Helper()
	if opts == nil {
		opts = newOpts()
	}
	ret, err := run(script, &gad.RunOpts{Globals: opts.global, Args: gad.Args{opts.args}})
	require.NoError(t, err)
	require.Equal(t, expected, ret)
}

func run(script string, opts *gad.RunOpts) (ret gad.Object, err error) {
	_, ret, err = runV(script, opts)
	return
}

func runV(script string, opts *gad.RunOpts) (vm *gad.VM, ret gad.Object, err error) {
	mm := gad.NewModuleMap()
	mm.AddBuiltinModuleInit("time", ModuleInit)
	c := gad.CompileOptions{CompilerOptions: gad.DefaultCompilerOptions}
	c.ModuleMap = mm

	builtins := gad.NewBuiltins().Build()
	_, bc, err := gad.Compile(gad.NewSymbolTable(builtins.Builtins().NameSet), []byte(script), c)
	if err != nil {
		return
	}
	vm = gad.NewVM(builtins, bc)
	ret, err = vm.RunOpts(opts)
	return
}
