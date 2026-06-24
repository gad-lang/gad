package gad

import (
	"time"

	"github.com/gad-lang/gad/token"
)

// leastTimeStep returns the duration of the least-significant non-zero clock
// component (nanosecond/second/minute/hour) of a time-of-day; at midnight (all
// zero) it returns one day. This is the step the `++` (increase) and `--`
// (decrease) unary operators apply to temporal values: a plain date steps by a
// day, `…T08:05:00` steps by a minute (seconds are zero), `…T08:05:30` by a
// second, and `…T08:00:00` by an hour.
func leastTimeStep(hour, min, sec, nsec int) time.Duration {
	switch {
	case nsec != 0:
		return time.Nanosecond
	case sec != 0:
		return time.Second
	case min != 0:
		return time.Minute
	case hour != 0:
		return time.Hour
	default:
		return 24 * time.Hour
	}
}

// --- *Time ---

func (o *Time) UnOpInc(*VM) (Object, error) {
	h, m, s := o.Value.Clock()
	return &Time{Value: o.Value.Add(leastTimeStep(h, m, s, o.Value.Nanosecond()))}, nil
}

func (o *Time) UnOpDec(*VM) (Object, error) {
	h, m, s := o.Value.Clock()
	return &Time{Value: o.Value.Add(-leastTimeStep(h, m, s, o.Value.Nanosecond()))}, nil
}

// --- CalendarDate ---

// A calendar date carries no time-of-day, so it always steps by one day.

func (o CalendarDate) UnOpInc(*VM) (Object, error) {
	return o.addDuration(24 * time.Hour), nil
}

func (o CalendarDate) UnOpDec(*VM) (Object, error) {
	return o.addDuration(-24 * time.Hour), nil
}

// --- CalendarTime ---

func (o CalendarTime) UnOpInc(*VM) (Object, error) {
	step := leastTimeStep(o.Hour(), o.Minute(), o.Second(), o.Nanosecond())
	return o.shift(token.Add, int64(step))
}

func (o CalendarTime) UnOpDec(*VM) (Object, error) {
	step := leastTimeStep(o.Hour(), o.Minute(), o.Second(), o.Nanosecond())
	return o.shift(token.Sub, int64(step))
}

// --- Duration ---

func (o Duration) UnOpSub(*VM) (Object, error) { return -o, nil }
func (o Duration) UnOpAdd(*VM) (Object, error) { return o, nil }
