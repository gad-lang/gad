// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"reflect"
	"strconv"

	"github.com/gad-lang/gad/registry"
	"github.com/gad-lang/gad/repr"
)

func init() {
	registry.RegisterAnyConverter(reflect.TypeOf((*fmtScanArg)(nil)),
		func(in any) (any, bool) {
			sa := in.(*fmtScanArg)
			if sa.fmtArgValue != nil {
				return sa.Arg(), true
			}
			return Nil, false
		},
	)
}

// FmtScanArg is an interface that wraps methods required to scan argument with
// scan functions.
type FmtScanArg interface {
	// Set sets status of scanning. It is set false before scanning and true
	// after scanning if argument is scanned.
	Set(bool)
	// Arg must return either a pointer to a basic ToInterface type or implementations of
	// fmt.Scanner interface.
	Arg() any
	// Value must return scanned, non-nil Gad Object.
	Value() Object
}

// fmtArgValue is an interface implemented by the basic scannable types and used by
// fmtScanArg type.
type fmtArgValue interface {
	Arg() any
	Value() Object
}

var fmtScanArgType = NewBuiltinObjType("scanArg")

// fmtScanArg implements Object and FmtScanArg interfaces to provide arguments to
// scan functions.
// "Value" selector in Gad scripts gives the scanned value if scan was successful.
type fmtScanArg struct {
	ObjectImpl
	fmtArgValue
	ok bool
}

var _ FmtScanArg = (*fmtScanArg)(nil)

func (*fmtScanArg) Type() ObjectType { return fmtScanArgType }

func (o *fmtScanArg) ToString() string { return repr.Quote("scanArg") }

func (o *fmtScanArg) IsFalsy() bool { return !o.ok }

func (o *fmtScanArg) IndexGet(_ *VM, index Object) (Object, error) {
	if o.ok && index.ToString() == "Value" {
		return o.Value(), nil
	}
	return Nil, nil
}

func (o *fmtScanArg) Set(scanned bool) { o.ok = scanned }

func fmtNewScanArgFunc(c Call) (Object, error) {
	typ := "str"
	if c.Args.Length() > 0 {
		v := c.Args.Get(0)
	do:
		if b, ok := v.(*Func); ok {
			v = b.FuncSpec.CallerMethodDefault()
			goto do
		} else if b, ok := v.(*BuiltinFunction); ok {
			typ = b.FuncName
		} else if ot, ok := v.(ObjectType); ok {
			typ = ot.Name()
		} else {
			typ = v.ToString()
		}
	}
	var scan fmtScanArg
	switch typ {
	case "str":
		scan.fmtArgValue = &fmtStringType{}
	case "int":
		scan.fmtArgValue = &fmtIntType{}
	case "uint":
		scan.fmtArgValue = &fmtUintType{}
	case "float":
		scan.fmtArgValue = &fmtFloatType{}
	case "bool":
		scan.fmtArgValue = &fmtBoolType{}
	case "char":
		scan.fmtArgValue = &fmtCharType{}
	case "bytes":
		scan.fmtArgValue = &fmtBytesType{}
	case "decimal":
		scan.fmtArgValue = &fmtDecimalType{}
	default:
		return nil, ErrType.NewError(strconv.Quote(typ), "not implemented")
	}
	return &scan, nil
}

type fmtStringType struct {
	v string
}

func (st *fmtStringType) Arg() any {
	return &st.v
}

func (st *fmtStringType) Value() Object {
	return Str(st.v)
}

type fmtBytesType struct {
	v []byte
}

func (bt *fmtBytesType) Arg() any {
	return &bt.v
}

func (bt *fmtBytesType) Value() Object {
	return Bytes(bt.v)
}

type fmtIntType struct {
	v int64
}

func (it *fmtIntType) Arg() any {
	return &it.v
}

func (it *fmtIntType) Value() Object {
	return Int(it.v)
}

type fmtUintType struct {
	v uint64
}

func (ut *fmtUintType) Arg() any {
	return &ut.v
}

func (ut *fmtUintType) Value() Object {
	return Uint(ut.v)
}

type fmtFloatType struct {
	v float64
}

func (ft *fmtFloatType) Arg() any {
	return &ft.v
}

func (ft *fmtFloatType) Value() Object {
	return Float(ft.v)
}

type fmtDecimalType struct {
	v string
}

func (dt *fmtDecimalType) Arg() any {
	return &dt.v
}

func (dt *fmtDecimalType) Value() Object {
	return MustDecimalFromString(Str(dt.v))
}

type fmtCharType struct {
	v rune
}

func (ct *fmtCharType) Arg() any {
	return &ct.v
}

func (ct *fmtCharType) Value() Object {
	return Char(ct.v)
}

type fmtBoolType struct {
	v bool
}

func (bt *fmtBoolType) Arg() any {
	return &bt.v
}

func (bt *fmtBoolType) Value() Object {
	return Bool(bt.v)
}
