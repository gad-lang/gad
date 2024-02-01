// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"html/template"
	"strconv"
	"unicode/utf8"

	"github.com/gad-lang/gad/registry"
)

const (
	// AttrModuleName is a special attribute injected into modules to identify
	// the modules by name.
	AttrModuleName = "__module_name__"
)

// CallableFunc is a function signature for a callable function that accepts
// a Call struct.
type CallableFunc = func(Call) (ret Object, err error)

func MustToObject(v any) (ret Object) {
	var err error
	if ret, err = ToObject(v); err != nil {
		panic(err)
	}
	return
}

// ToObject is analogous to ToObject but it will always convert signed integers to
// Int and unsigned integers to Uint. It is an alternative to ToObject.
// Note that, this function is subject to change in the future.
func ToObject(v any) (ret Object, err error) {
	switch v := v.(type) {
	case nil:
		ret = Nil
	case string:
		ret = Str(v)
	case bool:
		if v {
			ret = True
		} else {
			ret = False
		}
	case int:
		ret = Int(v)
	case int64:
		ret = Int(v)
	case uint64:
		ret = Uint(v)
	case float64:
		ret = Float(v)
	case float32:
		ret = Float(v)
	case int32:
		ret = Int(v)
	case int16:
		ret = Int(v)
	case int8:
		ret = Int(v)
	case uint:
		ret = Uint(v)
	case uint32:
		ret = Uint(v)
	case uint16:
		ret = Uint(v)
	case uint8:
		ret = Uint(v)
	case uintptr:
		ret = Uint(v)
	case []byte:
		if v != nil {
			ret = Bytes(v)
		} else {
			ret = Bytes{}
		}
	case map[string]Object:
		if v != nil {
			ret = Dict(v)
		} else {
			ret = Dict{}
		}
	case []Object:
		if v != nil {
			ret = Array(v)
		} else {
			ret = Array{}
		}
	case Object:
		ret = v
	case CallableFunc:
		if v != nil {
			ret = &Function{Value: v}
		} else {
			ret = Nil
		}
	case error:
		ret = &Error{Message: v.Error(), Cause: v}
	case template.HTML:
		ret = RawStr(v)
	default:
		if out, ok := registry.ToObject(v); ok {
			ret, ok = out.(Object)
			if ok {
				return
			}
		}
		if ret, err = NewReflectValue(v); err == nil && ret == nil {
			ret = Nil
		}
	}
	return
}

// ToInterface tries to convert an Object o to an any value.
func ToInterface(o Object) (ret any) {
	switch o := o.(type) {
	case Int:
		ret = int64(o)
	case Str:
		ret = string(o)
	case Bytes:
		ret = []byte(o)
	case Array:
		arr := make([]any, len(o))
		for i, val := range o {
			arr[i] = ToInterface(val)
		}
		ret = arr
	case Dict:
		m := make(map[string]any, len(o))
		for key, v := range o {
			m[key] = ToInterface(v)
		}
		ret = m
	case Uint:
		ret = uint64(o)
	case Char:
		ret = rune(o)
	case Float:
		ret = float64(o)
	case Bool:
		ret = bool(o)
	case *SyncDict:
		if o == nil {
			return map[string]any{}
		}
		o.RLock()
		defer o.RUnlock()
		m := make(map[string]any, len(o.Value))
		for key, v := range o.Value {
			m[key] = ToInterface(v)
		}
		ret = m
	case *NilType:
		ret = nil
	case ToIterfaceConverter:
		ret = o.ToInterface()
	default:
		if out, ok := registry.ToInterface(o); ok {
			ret = out
		} else {
			ret = o
		}
	}
	return
}

// ToString will try to convert an Object to Gad string value.
func ToString(o Object) (v Str, ok bool) {
	if v, ok = o.(Str); ok {
		return
	}
	vv, ok := ToGoString(o)
	if ok {
		v = Str(vv)
	}
	return
}

// ToBytes will try to convert an Object to Gad bytes value.
func ToBytes(o Object) (v Bytes, ok bool) {
	if v, ok = o.(Bytes); ok {
		return
	}
	return ToGoByteSlice(o)
}

// ToInt will try to convert an Object to Gad int value.
func ToInt(o Object) (v Int, ok bool) {
	if v, ok = o.(Int); ok {
		return
	}
	vv, ok := ToGoInt64(o)
	if ok {
		v = Int(vv)
	}
	return
}

// ToUint will try to convert an Object to Gad uint value.
func ToUint(o Object) (v Uint, ok bool) {
	if v, ok = o.(Uint); ok {
		return
	}
	vv, ok := ToGoUint64(o)
	if ok {
		v = Uint(vv)
	}
	return
}

// ToFloat will try to convert an Object to Gad float value.
func ToFloat(o Object) (v Float, ok bool) {
	if v, ok = o.(Float); ok {
		return
	}
	vv, ok := ToGoFloat64(o)
	if ok {
		v = Float(vv)
	}
	return
}

// ToChar will try to convert an Object to Gad char value.
func ToChar(o Object) (v Char, ok bool) {
	if v, ok = o.(Char); ok {
		return
	}
	vv, ok := ToGoRune(o)
	if ok {
		v = Char(vv)
	}
	return
}

// ToBool will try to convert an Object to Gad bool value.
func ToBool(o Object) (v Bool, ok bool) {
	if v, ok = o.(Bool); ok {
		return
	}
	vv, ok := ToGoBool(o)
	v = Bool(vv)
	return
}

// ToArray will try to convert an Object to Gad array value.
func ToArray(o Object) (v Array, ok bool) {
	v, ok = o.(Array)
	return
}

// ToMap will try to convert an Object to Gad map value.
func ToMap(o Object) (v Dict, ok bool) {
	v, ok = o.(Dict)
	return
}

// ToSyncMap will try to convert an Object to Gad syncMap value.
func ToSyncMap(o Object) (v *SyncDict, ok bool) {
	v, ok = o.(*SyncDict)
	return
}

// ToGoString will try to convert an Object to ToInterface string value.
func ToGoString(o Object) (v string, ok bool) {
	if o == Nil {
		return
	}
	v, ok = o.ToString(), true
	return
}

// ToGoByteSlice will try to convert an Object to ToInterface byte slice.
func ToGoByteSlice(o Object) (v []byte, ok bool) {
	switch o := o.(type) {
	case Bytes:
		v, ok = o, true
	case Str:
		v, ok = make([]byte, len(o)), true
		copy(v, o)
	case BytesConverter:
		var err error
		v, err = o.ToBytes()
		if err != nil {
			return
		}
		ok = true
	}
	return
}

// ToGoInt will try to convert a numeric, bool or string Object to ToInterface int value.
func ToGoInt(o Object) (v int, ok bool) {
	switch o := o.(type) {
	case Int:
		v, ok = int(o), true
	case Uint:
		v, ok = int(o), true
	case Float:
		v, ok = int(o), true
	case Char:
		v, ok = int(o), true
	case Bool:
		ok = true
		if o {
			v = 1
		}
	case Str:
		if o == "" {
			return
		}
		if vv, err := strconv.ParseInt(string(o), 0, 0); err == nil {
			v = int(vv)
			ok = true
		}
	}
	return
}

// ToGoInt64 will try to convert a numeric, bool or string Object to ToInterface int64
// value.
func ToGoInt64(o Object) (v int64, ok bool) {
	switch o := o.(type) {
	case Int:
		v, ok = int64(o), true
	case Uint:
		v, ok = int64(o), true
	case Float:
		v, ok = int64(o), true
	case Decimal:
		v, ok = o.Go().IntPart(), true
	case Char:
		v, ok = int64(o), true
	case Bool:
		ok = true
		if o {
			v = 1
		}
	case Str:
		if o == "" {
			return
		}
		if vv, err := strconv.ParseInt(string(o), 0, 64); err == nil {
			v = vv
			ok = true
		}
	}
	return
}

// ToGoUint64 will try to convert a numeric, bool or string Object to ToInterface uint64
// value.
func ToGoUint64(o Object) (v uint64, ok bool) {
	switch o := o.(type) {
	case Int:
		v, ok = uint64(o), true
	case Uint:
		v, ok = uint64(o), true
	case Float:
		v, ok = uint64(o), true
	case Decimal:
		v, ok = o.Go().BigInt().Uint64(), true
	case Char:
		v, ok = uint64(o), true
	case Bool:
		ok = true
		if o {
			v = 1
		}
	case Str:
		if o == "" {
			return
		}
		if vv, err := strconv.ParseUint(string(o), 0, 64); err == nil {
			v = vv
			ok = true
		}
	}
	return
}

// ToGoFloat64 will try to convert a numeric, bool or string Object to ToInterface
// float64 value.
func ToGoFloat64(o Object) (v float64, ok bool) {
	switch o := o.(type) {
	case Int:
		v, ok = float64(o), true
	case Uint:
		v, ok = float64(o), true
	case Float:
		v, ok = float64(o), true
	case Decimal:
		v, ok = o.Go().InexactFloat64(), true
	case Char:
		v, ok = float64(o), true
	case Bool:
		ok = true
		if o {
			v = 1
		}
	case Str:
		if o == "" {
			ok = true
			return
		}
		if vv, err := strconv.ParseFloat(string(o), 64); err == nil {
			v = vv
			ok = true
		}
	}
	return
}

// ToGoRune will try to convert a int like Object to ToInterface rune value.
func ToGoRune(o Object) (v rune, ok bool) {
	switch o := o.(type) {
	case Int:
		v, ok = rune(o), true
	case Uint:
		v, ok = rune(o), true
	case Char:
		v, ok = rune(o), true
	case Float:
		v, ok = rune(o), true
	case Decimal:
		v, ok = rune(o.Go().BigInt().Uint64()), true
	case Str:
		ok = true
		v, _ = utf8.DecodeRuneInString(string(o))
	case Bool:
		ok = true
		if o {
			v = 1
		}
	}
	return
}

// ToGoBool will try to convert an Object to ToInterface bool value.
func ToGoBool(o Object) (v bool, ok bool) {
	v, ok = !o.IsFalsy(), true
	return
}
