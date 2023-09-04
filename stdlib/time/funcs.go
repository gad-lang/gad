package time

import (
	"time"

	"github.com/gad-lang/gad"
)

//go:generate go run ../../cmd/mkcallable -output zfuncs.go funcs.go

//gad:callable:convert *Location ToLocation
//gad:callable:convert *Time ToTime

// ToLocation will try to convert given gad.Object to *Location value.
func ToLocation(o gad.Object) (ret *Location, ok bool) {
	if v, isString := o.(gad.String); isString {
		var err error
		o, err = loadLocationFunc(string(v))
		if err != nil {
			return
		}
	}
	ret, ok = o.(*Location)
	return
}

// ToTime will try to convert given gad.Object to *Time value.
func ToTime(o gad.Object) (ret *Time, ok bool) {
	switch o := o.(type) {
	case *Time:
		ret, ok = o, true
	case gad.Int:
		v := time.Unix(int64(o), 0)
		ret, ok = &Time{Value: v}, true
	case gad.String:
		v, err := time.Parse(time.RFC3339Nano, string(o))
		if err != nil {
			v, err = time.Parse(time.RFC3339, string(o))
		}
		if err == nil {
			ret, ok = &Time{Value: v}, true
		}
	}
	return
}

// Since, Until
//
//gad:callable funcPTRO(t *Time) (ret gad.Object)

// Add, Round, Truncate
//
//gad:callable funcPTi64RO(t *Time, d int64) (ret gad.Object)

// Sub, After, Before
//
//gad:callable funcPTTRO(t1 *Time, t2 *Time) (ret gad.Object)

// AddDate
//
//gad:callable funcPTiiiRO(t *Time, i1 int, i2 int, i3 int) (ret gad.Object)

// Format
//
//gad:callable funcPTsRO(t *Time, s string) (ret gad.Object)

// AppendFormat
//
//gad:callable funcPTb2sRO(t *Time, b []byte, s string) (ret gad.Object)

// In
//
//gad:callable funcPTLRO(t *Time, loc *Location) (ret gad.Object)
