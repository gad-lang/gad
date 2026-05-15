// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package encoder

import (
	"fmt"
	"reflect"

	"github.com/gad-lang/gad"
)

func Encode(w Writer, version byte, o any) (err error) {
	ed := Encoders.byVersion[version]
	if ed == nil {
		return fmt.Errorf("encoder of %d not supported", o)
	}
	if err = w.WriteByte(version); err != nil {
		return
	}
	return ed.Encode(w, o)
}

func EncodeObject(w Writer, o any) (err error) {
	rt := reflect.TypeOf(o)

	for rt.Kind() == reflect.Interface || rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	ed := Encoders.byType[rt]
	if ed == nil {
		return fmt.Errorf("encoder of %T not supported", o)
	}
	return Encode(w, ed.LastVersion, o)
}

func EncodeArray[T any](w Writer, arr []T) (err error) {
	return WriteArray(w, arr, func(w Writer, v T) error {
		return EncodeObject(w, v)
	})
}

func EncodeDict(w Writer, d gad.Dict) (err error) {
	if err = writeInt(w, len(d)); err != nil {
		return
	}

	for k, v := range d {
		if err = writeString(w, k); err != nil {
			return
		}
		if err = EncodeObject(w, v); err != nil {
			return
		}
	}
	return
}
