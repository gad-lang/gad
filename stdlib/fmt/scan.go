package fmt

import (
	"reflect"
	"strconv"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/registry"
)

func init() {
	registry.RegisterAnyConverter(reflect.TypeOf((*scanArg)(nil)),
		func(in any) (any, bool) {
			sa := in.(*scanArg)
			if sa.argValue != nil {
				return sa.Arg(), true
			}
			return gad.Nil, false
		},
	)
}

// ScanArg is an interface that wraps methods required to scan argument with
// scan functions.
type ScanArg interface {
	// Set sets status of scanning. It is set false before scanning and true
	// after scanning if argument is scanned.
	Set(bool)
	// Arg must return either a pointer to a basic Go type or implementations of
	// fmt.Scanner interface.
	Arg() any
	// Value must return scanned, non-nil Gad Object.
	Value() gad.Object
}

// argValue is an interface implemented by the basic scannable types and used by
// scanArg type.
type argValue interface {
	Arg() any
	Value() gad.Object
}

// scanArg implements gad.Object and ScanArg interfaces to provide arguments to
// scan functions.
// "Value" selector in Gad scripts gives the scanned value if scan was successful.
type scanArg struct {
	gad.ObjectImpl
	argValue
	ok bool
}

var _ ScanArg = (*scanArg)(nil)

func (*scanArg) TypeName() string { return "scanArg" }

func (o *scanArg) String() string { return "<scanArg>" }

func (o *scanArg) IsFalsy() bool { return !o.ok }

func (o *scanArg) IndexGet(index gad.Object) (gad.Object, error) {
	if o.ok && index.String() == "Value" {
		return o.Value(), nil
	}
	return gad.Nil, nil
}

func (o *scanArg) Set(scanned bool) { o.ok = scanned }

func newScanArgFunc(c gad.Call) (gad.Object, error) {
	typ := "string"
	if c.Args.Len() > 0 {
		v := c.Args.Get(0)
		if b, ok := v.(*gad.BuiltinFunction); ok {
			typ = b.Name
		} else {
			typ = v.String()
		}
	}
	var scan scanArg
	switch typ {
	case "string":
		scan.argValue = &stringType{}
	case "int":
		scan.argValue = &intType{}
	case "uint":
		scan.argValue = &uintType{}
	case "float":
		scan.argValue = &floatType{}
	case "bool":
		scan.argValue = &boolType{}
	case "char":
		scan.argValue = &charType{}
	case "bytes":
		scan.argValue = &bytesType{}
	default:
		return nil, gad.ErrType.NewError(strconv.Quote(typ), "not implemented")
	}
	return &scan, nil
}

type stringType struct {
	v string
}

func (st *stringType) Arg() any {
	return &st.v
}

func (st *stringType) Value() gad.Object {
	return gad.String(st.v)
}

type bytesType struct {
	v []byte
}

func (bt *bytesType) Arg() any {
	return &bt.v
}

func (bt *bytesType) Value() gad.Object {
	return gad.Bytes(bt.v)
}

type intType struct {
	v int64
}

func (it *intType) Arg() any {
	return &it.v
}

func (it *intType) Value() gad.Object {
	return gad.Int(it.v)
}

type uintType struct {
	v uint64
}

func (ut *uintType) Arg() any {
	return &ut.v
}

func (ut *uintType) Value() gad.Object {
	return gad.Uint(ut.v)
}

type floatType struct {
	v float64
}

func (ft *floatType) Arg() any {
	return &ft.v
}

func (ft *floatType) Value() gad.Object {
	return gad.Float(ft.v)
}

type charType struct {
	v rune
}

func (ct *charType) Arg() any {
	return &ct.v
}

func (ct *charType) Value() gad.Object {
	return gad.Char(ct.v)
}

type boolType struct {
	v bool
}

func (bt *boolType) Arg() any {
	return &bt.v
}

func (bt *boolType) Value() gad.Object {
	return gad.Bool(bt.v)
}
