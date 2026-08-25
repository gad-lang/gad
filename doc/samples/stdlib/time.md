
# `time` module

## Public API

### utc

```gad
utc() <Location>
```

Returns Universal Coordinated Time (UTC) location.

## Example

```gad
time.isLocation(time.utc())
>>> true
```

### local

```gad
local() <Location>
```

Returns the system's local time zone location.

## Example

```gad
time.isLocation(time.local())
>>> true
```

### monthString

```gad
monthString(m Months) <str>
```

Returns English name of the month m ("January", "February", ...).

## Example

```gad
time.monthString(1)   // 1 = January
>>> "January"
```

### weekdayString

```gad
weekdayString(w Weekdays) <str>
```

Returns English name of the int weekday w, note that 0 is Sunday.

## Example

```gad
time.weekdayString(0)
>>> "Sunday"
```

### durationString

```gad
durationString(d duration) <str>
```

Returns a string representing the duration d in the form "72h3m0.5s".

## Example

```gad
time.durationString(90 * time.Minute)
>>> "1h30m0s"
```

### durationNanoseconds

```gad
durationNanoseconds(d duration) <int>
```

Returns the duration d as an int nanosecond count.

## Example

```gad
time.durationNanoseconds(time.Second)
>>> 1000000000
```

### durationMicroseconds

```gad
durationMicroseconds(d duration) <int>
```

Returns the duration d as an int microsecond count.

## Example

```gad
time.durationMicroseconds(time.Millisecond)
>>> 1000
```

### durationMilliseconds

```gad
durationMilliseconds(d duration) <int>
```

Returns the duration d as an int millisecond count.

## Example

```gad
time.durationMilliseconds(time.Second)
>>> 1000
```

### durationSeconds

```gad
durationSeconds(d duration) <float>
```

Returns the duration d as a floating point number of seconds.

## Example

```gad
time.durationSeconds(90 * time.Minute)
>>> 5400
```

### durationMinutes

```gad
durationMinutes(d duration) <float>
```

Returns the duration d as a floating point number of minutes.

## Example

```gad
time.durationMinutes(90 * time.Minute)
>>> 90
```

### durationHours

```gad
durationHours(d duration) <float>
```

Returns the duration d as a floating point number of hours.

sleep(d duration)
Pauses the current goroutine for at least the duration.

## Example

```gad
time.durationHours(3 * time.Hour)
>>> 3
```

### parseDuration

```gad
parseDuration(s str) <duration>
```

Parses duration s and returns duration as int or error.

## Example

```gad
time.parseDuration("2h30m") == 2 * time.Hour + 30 * time.Minute
>>> true
```

### durationRound

```gad
durationRound(d duration, m duration) <duration>
```

Returns the result of rounding duration to the nearest multiple of m.

## Example

```gad
time.durationRound(time.parseDuration("1h17m30s"), time.Minute)
>>> dur 1h18m
```

### durationTruncate

```gad
durationTruncate(d duration, m duration) <duration>
```

Returns the result of rounding duration toward zero to a multiple of m.

## Example

```gad
time.durationTruncate(time.parseDuration("1h17m30s"), time.Minute)
>>> dur 1h17m
```

### fixedZone

```gad
fixedZone(name str, sec int) <Location>
```

Returns a Location that always uses the given zone name and offset
(seconds east of UTC).

## Example

```gad
time.isLocation(time.fixedZone("X+1", 3600))
>>> true
```

### loadLocation

```gad
loadLocation(name str) <Location>
```

Returns the Location with the given name.

## Example

```gad
time.isLocation(time.loadLocation("UTC"))
>>> true
```

### isLocation

```gad
isLocation(any) <bool>
```

Reports whether any value is of location type.

## Example

```gad
[time.isLocation(time.utc()), time.isLocation(5)]
>>> [true, false]
```

### time

```gad
time() <time>
```

Returns zero time.

## Example

```gad
time.isTime(time.time())
>>> true
```

### since

```gad
since(t time) <duration>
```

Returns the time elapsed since t.

### until

```gad
until(t time) <duration>
```

Returns the duration until t.

### date

```gad
date(year int, month Months, day int, hour int, min int, sec int, nsec int, loc Location) <time>
```

Returns the Time corresponding to yyyy-mm-dd hh:mm:ss + nsec nanoseconds
in the appropriate zone for that time in the given location. Zero values
of optional arguments are used if not provided.

## Example

```gad
time.format(time.date(2026, 1, 31, 9, 0, 0, 0, time.utc()), "2006-01-02")
>>> "2026-01-31"
```

### now

```gad
now() <time>
```

Returns the current local time.

## Example

```gad
time.isTime(time.now())
>>> true
```

### parse

```gad
parse(layout str, value str, loc Location) <time>
```

Parses a formatted string and returns the time value it represents.
If location is not provided, ToInterface's `time.Parse` function is called
otherwise `time.ParseInLocation` is called.

## Example

```gad
time.format(time.parse("2006-01-02", "2026-01-31", time.utc()), "2006/01/02")
>>> "2026/01/31"
```

### strToDate

```gad
strToDate(s str) <date>
```

Parses a date from "YYYYMMDD" or "YYYY-MM-DD".

## Example

```gad
time.strToDate("2026-01-31").year()
>>> 2026
```

### strToTime

```gad
strToTime(s str) <time>
```

Parses an RFC3339 timestamp or "YYYY-MM-DD[ HH:MM:SS]" (UTC when no
zone is present).

## Example

```gad
time.format(time.strToTime("2026-01-31T09:00:00Z"), "2006-01-02")
>>> "2026-01-31"
```

### strToCalendarTime

```gad
strToCalendarTime(s str) <calendarTime>
```

Parses a zone-less "YYYY-MM-DD[ HH:MM:SS[.frac]]" timestamp.

## Example

```gad
time.strToCalendarTime("2026-01-31 09:30:00.001").ns()
>>> 1000000
```

### strToDuration

```gad
strToDuration(s str) <duration>
```

Parses a Go duration string (e.g. "1h30m").

## Example

```gad
time.strToDuration("2h30m")
>>> dur 2h30m
```

### strToLocation

```gad
strToLocation(s str) <Location>
```

Parses a location from an offset ("-0300"/"-03:00") or an IANA name.

## Example

```gad
time.isLocation(time.strToLocation("UTC"))
>>> true
```

### unix

```gad
unix(sec int, nsec int) <time>
```

Returns the local time corresponding to the given Unix time,
sec seconds and nsec nanoseconds since January 1, 1970 UTC.
Zero values of optional arguments are used if not provided.

## Example

```gad
time.format(time.unix(1000000000, 0), "2006-01-02")
>>> "2001-09-08"
```

### add

```gad
add(t time, d duration) <time>
```

Deprecated: Use .Add method of time object.
Returns the time of t+duration.

## Example

```gad
time.format(time.add(time.strToTime("2026-01-31T09:00:00Z"), 24 * time.Hour), "2006-01-02")
>>> "2026-02-01"
```

### sub

```gad
sub(t1 time, t2 time) <duration>
```

Deprecated: Use .Sub method of time object.
Returns the duration of t1-t2.

## Example

```gad
time.sub(time.strToTime("2026-02-01T09:00:00Z"), time.strToTime("2026-01-31T09:00:00Z"))
>>> dur 24h
```

### addDate

```gad
addDate(t time, years int, months int, days int) <time>
```

Deprecated: Use .AddDate method of time object.
Returns the time corresponding to adding the given number of
years, months, and days to t.

## Example

```gad
time.format(time.addDate(time.strToTime("2026-01-31T09:00:00Z"), 1, 0, 0), "2006-01-02")
>>> "2027-01-31"
```

### after

```gad
after(t1 time, t2 time) <bool>
```

Deprecated: Use .After method of time object.
Reports whether the time t1 is after t2.

## Example

```gad
time.after(time.strToTime("2026-02-01T00:00:00Z"), time.strToTime("2026-01-31T09:00:00Z"))
>>> true
```

### before

```gad
before(t1 time, t2 time) <bool>
```

Deprecated: Use .Before method of time object.
Reports whether the time t1 is before t2.

## Example

```gad
time.before(time.strToTime("2026-01-31T09:00:00Z"), time.strToTime("2026-02-01T00:00:00Z"))
>>> true
```

### format

```gad
format(t time, layout str) <str>
```

Deprecated: Use .Format method of time object.
Returns a textual representation of the time value formatted according
to layout.

## Example

```gad
time.format(time.strToTime("2026-01-31T09:00:00Z"), "2006-01-02 15:04")
>>> "2026-01-31 09:00"
```

### appendFormat

```gad
appendFormat(t time, b bytes, layout str) <bytes>
```

Deprecated: Use .AppendFormat method of time object.
It is like `Format` but appends the textual representation to b and
returns the extended buffer.

## Example

```gad
str(time.appendFormat(time.strToTime("2026-01-31T09:00:00Z"), bytes(""), "2006"))
>>> "2026"
```

### round

```gad
round(t time, d duration) <time>
```

Deprecated: Use .Round method of time object.
Round returns the result of rounding t to the nearest multiple of
duration.

### truncate

```gad
truncate(t time, d duration) <time>
```

Deprecated: Use .Truncate method of time object.
Truncate returns the result of rounding t down to a multiple of duration.

## Example

```gad
time.format(time.truncate(time.strToTime("2026-01-31T09:37:52Z"), time.Hour), "15:04")
>>> "09:00"
```

### isTime

```gad
isTime(any) <bool>
```

Reports whether any value is of time type.

## Example

```gad
[time.isTime(time.now()), time.isTime(5)]
>>> [true, false]
```

## Example — `time.gad`

````gad
/**
Returns Universal Coordinated Time (UTC) location.

## Example

```gad
time.isLocation(time.utc())
>>> true
```
**/
export utc() <Location> => nil

/**
Returns the system's local time zone location.

## Example

```gad
time.isLocation(time.local())
>>> true
```
**/
export local() <Location> => nil

/**
Returns English name of the month m ("January", "February", ...).

## Example

```gad
time.monthString(1)   // 1 = January
>>> "January"
```
**/
export monthString(m Months) <str> => nil

/**
Returns English name of the int weekday w, note that 0 is Sunday.

## Example

```gad
time.weekdayString(0)
>>> "Sunday"
```
**/
export weekdayString(w Weekdays) <str> => nil

/**
Returns a string representing the duration d in the form "72h3m0.5s".

## Example

```gad
time.durationString(90 * time.Minute)
>>> "1h30m0s"
```
**/
export durationString(d duration) <str> => nil

/**
Returns the duration d as an int nanosecond count.

## Example

```gad
time.durationNanoseconds(time.Second)
>>> 1000000000
```
**/
export durationNanoseconds(d duration) <int> => nil

/**
Returns the duration d as an int microsecond count.

## Example

```gad
time.durationMicroseconds(time.Millisecond)
>>> 1000
```
**/
export durationMicroseconds(d duration) <int> => nil

/**
Returns the duration d as an int millisecond count.

## Example

```gad
time.durationMilliseconds(time.Second)
>>> 1000
```
**/
export durationMilliseconds(d duration) <int> => nil

/**
Returns the duration d as a floating point number of seconds.

## Example

```gad
time.durationSeconds(90 * time.Minute)
>>> 5400
```
**/
export durationSeconds(d duration) <float> => nil

/**
Returns the duration d as a floating point number of minutes.

## Example

```gad
time.durationMinutes(90 * time.Minute)
>>> 90
```
**/
export durationMinutes(d duration) <float> => nil

/**
Returns the duration d as a floating point number of hours.

sleep(d duration)
Pauses the current goroutine for at least the duration.

## Example

```gad
time.durationHours(3 * time.Hour)
>>> 3
```
**/
export durationHours(d duration) <float> => nil

/**
Parses duration s and returns duration as int or error.

## Example

```gad
time.parseDuration("2h30m") == 2 * time.Hour + 30 * time.Minute
>>> true
```
**/
export parseDuration(s str) <duration> => nil

/**
Returns the result of rounding duration to the nearest multiple of m.

## Example

```gad
time.durationRound(time.parseDuration("1h17m30s"), time.Minute)
>>> dur 1h18m
```
**/
export durationRound(d duration, m duration) <duration> => nil

/**
Returns the result of rounding duration toward zero to a multiple of m.

## Example

```gad
time.durationTruncate(time.parseDuration("1h17m30s"), time.Minute)
>>> dur 1h17m
```
**/
export durationTruncate(d duration, m duration) <duration> => nil

/**
Returns a Location that always uses the given zone name and offset
(seconds east of UTC).

## Example

```gad
time.isLocation(time.fixedZone("X+1", 3600))
>>> true
```
**/
export fixedZone(name str, sec int) <Location> => nil

/**
Returns the Location with the given name.

## Example

```gad
time.isLocation(time.loadLocation("UTC"))
>>> true
```
**/
export loadLocation(name str) <Location> => nil

/**
Reports whether any value is of location type.

## Example

```gad
[time.isLocation(time.utc()), time.isLocation(5)]
>>> [true, false]
```
**/
export isLocation(any) <bool> => nil

/**
Returns zero time.

## Example

```gad
time.isTime(time.time())
>>> true
```
**/
export time() <time> => nil

/**
Returns the time elapsed since t.
**/
export since(t time) <duration> => nil

/**
Returns the duration until t.
**/
export until(t time) <duration> => nil

/**
Returns the Time corresponding to yyyy-mm-dd hh:mm:ss + nsec nanoseconds
in the appropriate zone for that time in the given location. Zero values
of optional arguments are used if not provided.

## Example

```gad
time.format(time.date(2026, 1, 31, 9, 0, 0, 0, time.utc()), "2006-01-02")
>>> "2026-01-31"
```
**/
export date(year int, month Months, day int, hour int, min int, sec int, nsec int, loc Location) <time> => nil

/**
Returns the current local time.

## Example

```gad
time.isTime(time.now())
>>> true
```
**/
export now() <time> => nil

/**
Parses a formatted string and returns the time value it represents.
If location is not provided, ToInterface's `time.Parse` function is called
otherwise `time.ParseInLocation` is called.

## Example

```gad
time.format(time.parse("2006-01-02", "2026-01-31", time.utc()), "2006/01/02")
>>> "2026/01/31"
```
**/
export parse(layout str, value str, loc Location) <time> => nil

/**
Parses a date from "YYYYMMDD" or "YYYY-MM-DD".

## Example

```gad
time.strToDate("2026-01-31").year()
>>> 2026
```
**/
export strToDate(s str) <date> => nil

/**
Parses an RFC3339 timestamp or "YYYY-MM-DD[ HH:MM:SS]" (UTC when no
zone is present).

## Example

```gad
time.format(time.strToTime("2026-01-31T09:00:00Z"), "2006-01-02")
>>> "2026-01-31"
```
**/
export strToTime(s str) <time> => nil

/**
Parses a zone-less "YYYY-MM-DD[ HH:MM:SS[.frac]]" timestamp.

## Example

```gad
time.strToCalendarTime("2026-01-31 09:30:00.001").ns()
>>> 1000000
```
**/
export strToCalendarTime(s str) <calendarTime> => nil

/**
Parses a Go duration string (e.g. "1h30m").

## Example

```gad
time.strToDuration("2h30m")
>>> dur 2h30m
```
**/
export strToDuration(s str) <duration> => nil

/**
Parses a location from an offset ("-0300"/"-03:00") or an IANA name.

## Example

```gad
time.isLocation(time.strToLocation("UTC"))
>>> true
```
**/
export strToLocation(s str) <Location> => nil

/**
Returns the local time corresponding to the given Unix time,
sec seconds and nsec nanoseconds since January 1, 1970 UTC.
Zero values of optional arguments are used if not provided.

## Example

```gad
time.format(time.unix(1000000000, 0), "2006-01-02")
>>> "2001-09-08"
```
**/
export unix(sec int, nsec int) <time> => nil

/**
Deprecated: Use .Add method of time object.
Returns the time of t+duration.

## Example

```gad
time.format(time.add(time.strToTime("2026-01-31T09:00:00Z"), 24 * time.Hour), "2006-01-02")
>>> "2026-02-01"
```
**/
export add(t time, d duration) <time> => nil

/**
Deprecated: Use .Sub method of time object.
Returns the duration of t1-t2.

## Example

```gad
time.sub(time.strToTime("2026-02-01T09:00:00Z"), time.strToTime("2026-01-31T09:00:00Z"))
>>> dur 24h
```
**/
export sub(t1 time, t2 time) <duration> => nil

/**
Deprecated: Use .AddDate method of time object.
Returns the time corresponding to adding the given number of
years, months, and days to t.

## Example

```gad
time.format(time.addDate(time.strToTime("2026-01-31T09:00:00Z"), 1, 0, 0), "2006-01-02")
>>> "2027-01-31"
```
**/
export addDate(t time, years int, months int, days int) <time> => nil

/**
Deprecated: Use .After method of time object.
Reports whether the time t1 is after t2.

## Example

```gad
time.after(time.strToTime("2026-02-01T00:00:00Z"), time.strToTime("2026-01-31T09:00:00Z"))
>>> true
```
**/
export after(t1 time, t2 time) <bool> => nil

/**
Deprecated: Use .Before method of time object.
Reports whether the time t1 is before t2.

## Example

```gad
time.before(time.strToTime("2026-01-31T09:00:00Z"), time.strToTime("2026-02-01T00:00:00Z"))
>>> true
```
**/
export before(t1 time, t2 time) <bool> => nil

/**
Deprecated: Use .Format method of time object.
Returns a textual representation of the time value formatted according
to layout.

## Example

```gad
time.format(time.strToTime("2026-01-31T09:00:00Z"), "2006-01-02 15:04")
>>> "2026-01-31 09:00"
```
**/
export format(t time, layout str) <str> => nil

/**
Deprecated: Use .AppendFormat method of time object.
It is like `Format` but appends the textual representation to b and
returns the extended buffer.

## Example

```gad
str(time.appendFormat(time.strToTime("2026-01-31T09:00:00Z"), bytes(""), "2006"))
>>> "2026"
```
**/
export appendFormat(t time, b bytes, layout str) <bytes> => nil

/**
Deprecated: Use .Round method of time object.
Round returns the result of rounding t to the nearest multiple of
duration.
**/
export round(t time, d duration) <time> => nil

/**
Deprecated: Use .Truncate method of time object.
Truncate returns the result of rounding t down to a multiple of duration.

## Example

```gad
time.format(time.truncate(time.strToTime("2026-01-31T09:37:52Z"), time.Hour), "15:04")
>>> "09:00"
```
**/
export truncate(t time, d duration) <time> => nil

/**
Reports whether any value is of time type.

## Example

```gad
[time.isTime(time.now()), time.isTime(5)]
>>> [true, false]
```
**/
export isTime(any) <bool> => nil
````
