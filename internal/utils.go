package internal

import "reflect"

func TsType(v any, typ ...any) (ok bool) {
	ctyp := reflect.TypeOf(v)
	for _, t := range typ {
		if reflect.TypeOf(t) == ctyp {
			return true
		}
	}
	return false
}
