// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package encoder

import (
	"fmt"
	"reflect"

	"github.com/gad-lang/gad"
)

func Encode(ctx *WriteContext, typeID byte, version byte, o any) (err error) {
	versions := Encoders.byTypeVersion[typeID]
	if versions == nil {
		return fmt.Errorf("encoder of type %d not supported", typeID)
	}
	ed := versions[version]
	if ed == nil {
		return fmt.Errorf("encoder of type %d version %d not supported", typeID, version)
	}
	if err = ctx.WriteByte(typeID); err != nil {
		return
	}
	if err = ctx.WriteByte(version); err != nil {
		return
	}
	return ed.Encode(ctx, o)
}

func EncodeObject(ctx *WriteContext, o any) (err error) {
	rt := reflect.TypeOf(o)

	for rt.Kind() == reflect.Interface || rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	ed := Encoders.byType[rt]
	if ed == nil {
		return fmt.Errorf("encoder of %T not supported", o)
	}
	return Encode(ctx, ed.TypeID, ed.LastVersion, o)
}

func EncodeArray[T any](ctx *WriteContext, arr []T) (err error) {
	return WriteArray(ctx, arr, func(w Writer, v T) error {
		return EncodeObject(ctx, v)
	})
}

func EncodeDict(ctx *WriteContext, d gad.Dict) (err error) {
	if err = writeInt(ctx, len(d)); err != nil {
		return
	}

	for k, v := range d {
		if err = writeString(ctx, k); err != nil {
			return
		}
		if err = EncodeObject(ctx, v); err != nil {
			return
		}
	}
	return
}
