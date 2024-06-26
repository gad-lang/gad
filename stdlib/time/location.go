// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package time

import (
	"time"

	"github.com/gad-lang/gad"
)

// gad:doc
// ## Types
// ### location
//
// ToInterface Type
//
// ```go
// // Location represents location values and implements gad.Object interface.
// type Location struct {
//    gad.ObjectImpl
//    Value *time.Location
// }
// ```

var LocationType = &gad.BuiltinObjType{
	NameValue: "location",
}

// Location represents location values and implements gad.Object interface.
type Location struct {
	gad.ObjectImpl
	Value *time.Location
}

func (*Location) Type() gad.ObjectType {
	return LocationType
}

// ToString implements gad.Object interface.
func (o *Location) ToString() string {
	return o.Value.String()
}

// IsFalsy implements gad.Object interface.
func (o *Location) IsFalsy() bool {
	return o.Value == nil
}

// Equal implements gad.Object interface.
func (o *Location) Equal(right gad.Object) bool {
	if v, ok := right.(*Location); ok {
		return v == o || v.ToString() == o.ToString()
	}
	if v, ok := right.(gad.Str); ok {
		return o.ToString() == v.ToString()
	}
	return false
}
