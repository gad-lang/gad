// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"time"
)

// gad:doc
// ## Types
// ### location
//
// ToInterface Type
//
// ```go
// // Location represents location values and implements Object interface.
// type Location struct {
//    ObjectImpl
//    Value *time.Location
// }
// ```

var TimeLocationType = NewBuiltinObjType("Location").WithNew(locationNew)

// Location represents location values and implements Object interface.
type Location struct {
	ObjectImpl
	Value *time.Location
}

func (*Location) Type() ObjectType {
	return TimeLocationType
}

// ToString implements Object interface.
func (o *Location) ToString() string {
	return o.Value.String()
}

// IsFalsy implements Object interface.
func (o *Location) IsFalsy() bool {
	return o.Value == nil
}

// Equal implements Object interface.
func (o *Location) Equal(right Object) bool {
	if v, ok := right.(*Location); ok {
		return v == o || v.ToString() == o.ToString()
	}
	if v, ok := right.(Str); ok {
		return o.ToString() == v.ToString()
	}
	return false
}
