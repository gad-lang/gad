package fmt

import (
	"reflect"
	"strconv"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/registry"
	"github.com/gad-lang/gad/repr"
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
	// Arg must return either a pointer to a basic ToInterface type or implementations of
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

var scanArgType = &gad.BuiltinObjType{
	NameValue: "scanArg",
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

func (*scanArg) Type() gad.ObjectType { return scanArgType }

func (o *scanArg) ToString() string { return repr.Quote("scanArg") }

func (o *scanArg) IsFalsy() bool { return !o.ok }

func (o *scanArg) IndexGet(_ *gad.VM, index gad.Object) (gad.Object, error) {
	if o.ok && index.ToString() == "Value" {
		return o.Value(), nil
	}
	return gad.Nil, nil
}

func (o *scanArg) Set(scanned bool) { o.ok = scanned }

func newScanArgFunc(c gad.Call) (gad.Object, error) {
	typ := "str"
	if c.Args.Len() > 0 {
		v := c.Args.Get(0)
	do:
		if b, ok := v.(*gad.CallerObjectWithMethods); ok {
			v = b.CallerObject
			goto do
		} else if b, ok := v.(*gad.BuiltinFunction); ok {
			typ = b.Name
		} else if ot, ok := v.(gad.ObjectType); ok {
			typ = ot.Name()
		} else {
			typ = v.ToString()
		}
	}
	var scan scanArg
	switch typ {
	case "str":
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
	case "decimal":
		scan.argValue = &decimalType{}
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
	return gad.Str(st.v)
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

type decimalType struct {
	v string
}

func (dt *decimalType) Arg() any {
	return &dt.v
}

func (dt *decimalType) Value() gad.Object {
	return gad.MustDecimalFromString(gad.Str(dt.v))
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
