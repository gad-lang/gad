package gad

import (
	"strconv"
	"strings"
	"time"
)

// The literals of the time types — gad code that reads back as an equal
// value (issue #25): `2026-01-31D`, `2026-01-31t`, `2026-01-31T`,
// `1781609136.500U`, `dur 1h30m0s`; a value no literal writes, by the
// constructor that makes it.

// calendarDateLiteral is `YYYY-MM-DDD`.
func calendarDateLiteral(o CalendarDate) string {
	return o.ToString() + "D"
}

// calendarDateTimeLiteral is `YYYY-MM-DDDT` at midnight, else the date and
// its clock, `YYYY-MM-DDTHH:MM[:SS[.frac]]`; out of the years 0 to 9999,
// `time.strToCdtime("…")`.
func calendarDateTimeLiteral(o CalendarDateTime) string {
	w := o.wall()
	if w.Year() < 0 || w.Year() > 9999 {
		return "time.strToCdtime(" + strconv.Quote(o.ToString()) + ")"
	}
	if c := o.Clock(); c != 0 {
		return w.Format("2006-01-02") + "T" + strings.TrimSuffix(c.literal(), "T")
	}
	return w.Format("2006-01-02") + "DT"
}

// durationLiteral is `dur 1h30m0s`.
func durationLiteral(o Duration) string {
	return "dur " + o.ToString()
}

// timeLiteral is the time in UTC as `YYYY-MM-DDT` (midnight) or as unix
// seconds, `1781609136[.500]U`; in another location, that instant in it:
// `time.in(1781609136U, time.strToLocation("America/Sao_Paulo"))` — the
// machine's (Local) and a zone with no name it loads, by its offset:
// `"-03:00"`.
func timeLiteral(t time.Time) string {
	u := t.UTC()
	var lit string
	if u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0 && u.Year() >= 0 && u.Year() <= 9999 {
		lit = u.Format("2006-01-02") + "T"
	} else {
		lit = strconv.FormatInt(u.Unix(), 10)
		if ns := u.Nanosecond(); ns != 0 {
			frac := strings.TrimRight(strconv.Itoa(1e9 + ns)[1:], "0")
			// the U literal takes 3, 6 or 9 digits
			for len(frac)%3 != 0 {
				frac += "0"
			}
			lit += "." + frac
		}
		lit += "U"
	}
	loc := t.Location()
	if loc == time.UTC {
		return lit
	}
	name := loc.String()
	if _, err := time.LoadLocation(name); err != nil || name == "" || loc == time.Local {
		name = t.Format("-07:00")
	}
	return "time.in(" + lit + ", time.strToLocation(" + strconv.Quote(name) + "))"
}
