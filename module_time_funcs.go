package gad

import (
	"time"
)

//go:generate go run ../../cmd/mkcallable -output zfuncs.go funcs.go

//gad:callable:convert *Location ToLocation
//gad:callable:convert *Time ToTime

// ToLocation will try to convert given Object to *Location value.
func ToLocation(o Object) (ret *Location, ok bool) {
	if v, isString := o.(Str); isString {
		var err error
		o, err = TimeLoadLocationFunc(string(v))
		if err != nil {
			return
		}
	}
	ret, ok = o.(*Location)
	return
}

// ToTime will try to convert given Object to *Time value.
func ToTime(o Object) (ret *Time, ok bool) {
	switch o := o.(type) {
	case *Time:
		ret, ok = o, true
	case Int:
		v := time.Unix(int64(o), 0)
		ret, ok = &Time{Value: v}, true
	case Str:
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
//gad:callable FuncPTRO(t *Time) (ret Object)

// Add, Round, Truncate
//
//gad:callable FuncPTi64RO(t *Time, d int64) (ret Object)

// Sub, After, Before
//
//gad:callable FuncPTTRO(t1 *Time, t2 *Time) (ret Object)

// AddDate
//
//gad:callable FuncPTiiiRO(t *Time, i1 int, i2 int, i3 int) (ret Object)

// Format
//
//gad:callable FuncPTsRO(t *Time, s string) (ret Object)

// AppendFormat
//
//gad:callable FuncPTb2sRO(t *Time, b []byte, s string) (ret Object)

// In
//
//gad:callable FuncPTLRO(t *Time, loc *Location) (ret Object)
