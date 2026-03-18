package gad

import (
	"reflect"

	"github.com/gad-lang/gad/zeroer"
)

func IsNil(value any) bool {
	if value == nil || value == Nil {
		return true
	}
	return zeroer.IsNil(reflect.ValueOf(value))
}
