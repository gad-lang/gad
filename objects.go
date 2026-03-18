// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gad-lang/gad/internal/compat"
	"github.com/gad-lang/gad/quote"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

// Bool represents boolean values and implements Object interface.
type Bool bool

func (o Bool) Type() ObjectType {
	return TBool
}

func (o Bool) ToString() string {
	if o {
		return "true"
	}
	return "false"
}

// Equal implements Object interface.
func (o Bool) Equal(right Object) bool {
	if v, ok := right.(Bool); ok {
		return o == v
	}

	if v, ok := right.(Int); ok {
		return bool((o && v == 1) || (!o && v == 0))
	}

	if v, ok := right.(Uint); ok {
		return bool((o && v == 1) || (!o && v == 0))
	}
	return false
}

// IsFalsy implements Object interface.
func (o Bool) IsFalsy() bool { return bool(!o) }

// BinaryOp implements Object interface.
func (o Bool) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	bval := Int(0)
	if o {
		bval = Int(1)
	}
switchpos:
	switch v := right.(type) {
	case Int:
		switch tok {
		case token.Add:
			return bval + v, nil
		case token.Sub:
			return bval - v, nil
		case token.Mul:
			return bval * v, nil
		case token.Quo:
			if v == 0 {
				return nil, ErrZeroDivision
			}
			return bval / v, nil
		case token.Rem:
			return bval % v, nil
		case token.And:
			return bval & v, nil
		case token.Or:
			return bval | v, nil
		case token.Xor:
			return bval ^ v, nil
		case token.AndNot:
			return bval &^ v, nil
		case token.Shl:
			return bval << v, nil
		case token.Shr:
			return bval >> v, nil
		case token.Less:
			return Bool(bval < v), nil
		case token.LessEq:
			return Bool(bval <= v), nil
		case token.Greater:
			return Bool(bval > v), nil
		case token.GreaterEq:
			return Bool(bval >= v), nil
		}
	case Uint:
		bval := Uint(bval)
		switch tok {
		case token.Add:
			return bval + v, nil
		case token.Sub:
			return bval - v, nil
		case token.Mul:
			return bval * v, nil
		case token.Quo:
			if v == 0 {
				return nil, ErrZeroDivision
			}
			return bval / v, nil
		case token.Rem:
			return bval % v, nil
		case token.And:
			return bval & v, nil
		case token.Or:
			return bval | v, nil
		case token.Xor:
			return bval ^ v, nil
		case token.AndNot:
			return bval &^ v, nil
		case token.Shl:
			return bval << v, nil
		case token.Shr:
			return bval >> v, nil
		case token.Less:
			return Bool(bval < v), nil
		case token.LessEq:
			return Bool(bval <= v), nil
		case token.Greater:
			return Bool(bval > v), nil
		case token.GreaterEq:
			return Bool(bval >= v), nil
		}
	case Bool:
		if v {
			right = Int(1)
		} else {
			right = Int(0)
		}
		goto switchpos
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

// Format implements fmt.Formatter interface.
func (o Bool) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, bool(o))
}

type Flag bool

func (o Flag) Type() ObjectType {
	return TFlag
}

func (o Flag) ToString() string {
	if o {
		return "on"
	}
	return "off"
}

// Equal implements Object interface.
func (o Flag) Equal(right Object) bool {
	if v, ok := right.(Flag); ok {
		return o == v
	}
	return Bool(o).Equal(right)
}

// IsFalsy implements Object interface.
func (o Flag) IsFalsy() bool { return bool(!o) }

func (o Flag) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	if v, ok := right.(Flag); ok {
		right = Bool(v)
	}
	return Bool(o).BinaryOp(vm, tok, right)
}

// RawStr represents safe string values and implements Object interface.
type RawStr string

var (
	_ LengthGetter = RawStr("")
	_ ToWriter     = RawStr("")
)

func (o RawStr) Type() ObjectType {
	return TRawStr
}

func (o RawStr) ToString() string {
	return string(o)
}

func (o RawStr) Quoted() string {
	return quote.Quote(string(o), "`")
}

func (o RawStr) IsFalsy() bool {
	return len(o) == 0
}

func (o RawStr) Equal(right Object) bool {
	if v, ok := right.(RawStr); ok {
		return o == v
	}
	return false
}

func (o RawStr) IndexGet(_ *VM, index Object) (Object, error) {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
		if idx < 0 {
			idx = len(o) + idx
		}
	case Uint:
		idx = int(v)
	case Char:
		idx = int(v)
	default:
		return nil, NewIndexTypeError("int|uint|char", index.Type().Name())
	}
	if idx >= 0 && idx < len(o) {
		return Int(o[idx]), nil
	}
	return nil, ErrIndexOutOfBounds
}

// BinaryOp implements Object interface.
func (o RawStr) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Str:
		switch tok {
		case token.Add:
			return o + RawStr(v), nil
		case token.Less:
			return Bool(o < RawStr(v)), nil
		case token.LessEq:
			return Bool(o <= RawStr(v)), nil
		case token.Greater:
			return Bool(o > RawStr(v)), nil
		case token.GreaterEq:
			return Bool(o >= RawStr(v)), nil
		}
	case RawStr:
		switch tok {
		case token.Add:
			return o + v, nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Bytes:
		switch tok {
		case token.Add:
			var sb strings.Builder
			sb.WriteString(string(o))
			sb.Write(v)
			return Str(sb.String()), nil
		case token.Less:
			return Bool(string(o) < string(v)), nil
		case token.LessEq:
			return Bool(string(o) <= string(v)), nil
		case token.Greater:
			return Bool(string(o) > string(v)), nil
		case token.GreaterEq:
			return Bool(string(o) >= string(v)), nil
		}
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}

	if tok == token.Add {
		return o + RawStr(right.ToString()), nil
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

// Length implements LengthGetter interface.
func (o RawStr) Length() int {
	return len(o)
}

// Format implements fmt.Formatter interface.
func (o RawStr) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, string(o))
}

func (o RawStr) WriteTo(_ *VM, w io.Writer) (int64, error) {
	n, err := w.Write([]byte(o))
	return int64(n), err
}

// Str represents string values and implements Object interface.
type Str string

var (
	_ LengthGetter = Str("")
)

func (o Str) Type() ObjectType {
	return TStr
}

func (o Str) ToString() string {
	return string(o)
}

func (o Str) Quoted() string {
	return strconv.Quote(string(o))
}

// IndexGet represents string values and implements Object interface.
func (o Str) IndexGet(_ *VM, index Object) (Object, error) {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
		if idx < 0 {
			idx = len(o) + idx
		}
	case Uint:
		idx = int(v)
	case Char:
		idx = int(v)
	default:
		return nil, NewIndexTypeError("int|uint|char", index.Type().Name())
	}
	if idx >= 0 && idx < len(o) {
		return Int(o[idx]), nil
	}
	return nil, ErrIndexOutOfBounds
}

// Equal implements Object interface.
func (o Str) Equal(right Object) bool {
	if v, ok := right.(Str); ok {
		return o == v
	}
	if v, ok := right.(Bytes); ok {
		return string(o) == string(v)
	}
	return false
}

// IsFalsy implements Object interface.
func (o Str) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o Str) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Str:
		switch tok {
		case token.Add:
			return o + v, nil
		case token.Less:
			return Bool(o < v), nil
		case token.LessEq:
			return Bool(o <= v), nil
		case token.Greater:
			return Bool(o > v), nil
		case token.GreaterEq:
			return Bool(o >= v), nil
		}
	case Bytes:
		switch tok {
		case token.Add:
			var sb strings.Builder
			sb.WriteString(string(o))
			sb.Write(v)
			return Str(sb.String()), nil
		case token.Less:
			return Bool(string(o) < string(v)), nil
		case token.LessEq:
			return Bool(string(o) <= string(v)), nil
		case token.Greater:
			return Bool(string(o) > string(v)), nil
		case token.GreaterEq:
			return Bool(string(o) >= string(v)), nil
		}
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}

	if tok == token.Add {
		return o + Str(right.ToString()), nil
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

// Length implements LengthGetter interface.
func (o Str) Length() int {
	return len(o)
}

// Format implements fmt.Formatter interface.
func (o Str) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, string(o))
}

func (o Str) Print(state *PrinterState) error {
	state.WriteString(strconv.Quote(string(o)))
	return nil
}

// Bytes represents byte slice and implements Object interface.
type Bytes []byte

var (
	_ Object       = Bytes{}
	_ Copier       = Bytes{}
	_ LengthGetter = Bytes{}
)

func (o Bytes) Type() ObjectType {
	return TBytes
}

func (o Bytes) ToString() string {
	return string(o)
}

// Copy implements Copier interface.
func (o Bytes) Copy() Object {
	cp := make(Bytes, len(o))
	copy(cp, o)
	return cp
}

// IndexSet implements Object interface.
func (o Bytes) IndexSet(_ *VM, index, value Object) error {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
		if idx < 0 {
			idx = len(o) + idx
		}
	case Uint:
		idx = int(v)
	default:
		return NewIndexTypeError("int|uint", index.Type().Name())
	}

	if idx >= 0 && idx < len(o) {
		switch v := value.(type) {
		case Int:
			o[idx] = byte(v)
		case Uint:
			o[idx] = byte(v)
		default:
			return NewIndexValueTypeError("int|uint", value.Type().Name())
		}
		return nil
	}
	return ErrIndexOutOfBounds
}

// IndexGet represents string values and implements Object interface.
func (o Bytes) IndexGet(_ *VM, index Object) (Object, error) {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
		if idx < 0 {
			idx = len(o) + idx
		}
	case Uint:
		idx = int(v)
	default:
		return nil, NewIndexTypeError("int|uint|char", index.Type().Name())
	}

	if idx >= 0 && idx < len(o) {
		return Int(o[idx]), nil
	}
	return nil, ErrIndexOutOfBounds
}

// Equal implements Object interface.
func (o Bytes) Equal(right Object) bool {
	if v, ok := right.(Bytes); ok {
		return string(o) == string(v)
	}

	if v, ok := right.(Str); ok {
		return string(o) == string(v)
	}
	return false
}

// IsFalsy implements Object interface.
func (o Bytes) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o Bytes) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case Bytes:
		switch tok {
		case token.Add:
			return append(o, v...), nil
		case token.Less:
			return Bool(bytes.Compare(o, v) == -1), nil
		case token.LessEq:
			cmp := bytes.Compare(o, v)
			return Bool(cmp == 0 || cmp == -1), nil
		case token.Greater:
			return Bool(bytes.Compare(o, v) == 1), nil
		case token.GreaterEq:
			cmp := bytes.Compare(o, v)
			return Bool(cmp == 0 || cmp == 1), nil
		}
	case Str:
		switch tok {
		case token.Add:
			return append(o, v...), nil
		case token.Less:
			return Bool(string(o) < string(v)), nil
		case token.LessEq:
			return Bool(string(o) <= string(v)), nil
		case token.Greater:
			return Bool(string(o) > string(v)), nil
		case token.GreaterEq:
			return Bool(string(o) >= string(v)), nil
		}
	case *NilType:
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

// Length implements LengthGetter interface.
func (o Bytes) Length() int {
	return len(o)
}

func (o Bytes) WriteTo(_ *VM, w io.Writer) (int64, error) {
	n, err := w.Write(o)
	return int64(n), err
}

func (o Bytes) ToStringF(w io.Writer) (err error) {
	if _, err = w.Write([]byte{'0', 'x'}); err == nil {
		_, err = hex.NewEncoder(w).Write(o)
	}
	return
}

type FunctionOption func(f *Function)

func FunctionWithParams(builder func(newParam func(name string) *ParamBuilder)) FunctionOption {
	return func(f *Function) {
		if f.Header == nil {
			f.Header = new(FunctionHeader)
		}
		f.Header.WithParams(builder)
	}
}
func FunctionWithNamedParams(builder func(newParam func(name string) *NamedParamBuilder)) FunctionOption {
	return func(f *Function) {
		if f.Header == nil {
			f.Header = new(FunctionHeader)
		}
		f.Header.WithNamedParams(builder)
	}
}

// Function represents a function object and implements Object interface.
type Function struct {
	ObjectImpl
	FuncName     string
	Value        func(Call) (Object, error)
	ToStringFunc func() string
	Header       *FunctionHeader
	pt           ParamsTypes
	Module       *Module
}

func (f *Function) SetModule(module *Module) {
	f.Module = module
}

func (f *Function) GetModule() *Module {
	return f.Module
}

func NewFunction(name string, v func(Call) (Object, error), opt ...FunctionOption) *Function {
	f := &Function{FuncName: name, Value: v}
	for _, opt := range opt {
		opt(f)
	}
	return f
}

func (f *Function) WithHeader(do func(h *FunctionHeader)) *Function {
	f.Header = &FunctionHeader{}
	do(f.Header)
	return f
}

func (f *Function) Name() string {
	return f.FuncName
}

func (f *Function) ParamTypes() (types ParamsTypes) {
	if f.pt == nil {
		if f.Header != nil {
			f.pt = f.Header.ParamTypes()
		}
	}
	return f.pt
}

func (f *Function) String() string {
	s := f.Name()
	if f.Header != nil {
		s += f.Header.String()
	} else {
		s += "(⋅⋅⋅)"
	}
	return s
}

var _ Object = (*Function)(nil)

func (*Function) Type() ObjectType {
	return TFunction
}

func (f *Function) ToString() string {
	if f.ToStringFunc != nil {
		return f.ToStringFunc()
	}
	return fmt.Sprintf(ReprQuote("function %s"), f.String())
}

// Copy implements Copier interface.
func (f *Function) Copy() Object {
	return &Function{
		FuncName: f.FuncName,
		Value:    f.Value,
	}
}

// Equal implements Object interface.
func (f *Function) Equal(right Object) bool {
	v, ok := right.(*Function)
	if !ok {
		return false
	}
	return v == f
}

// IsFalsy implements Object interface.
func (*Function) IsFalsy() bool { return false }

func (f *Function) Call(call Call) (Object, error) {
	return f.Value(call)
}

var (
	_ CallerObject = (*BuiltinFunction)(nil)
)

// BuiltinFunction represents a builtin function object and implements Object interface.
type BuiltinFunction struct {
	ObjectImpl
	Module                *Module
	FuncName              string
	Value                 func(Call) (Object, error)
	Header                *FunctionHeader
	AcceptMethodsDisabled bool
	Usage                 string
}

func NewBuiltinFunction(name string, value func(Call) (Object, error)) *BuiltinFunction {
	return &BuiltinFunction{FuncName: name, Value: value}
}

func (f *BuiltinFunction) Doc() string {
	var buf strings.Builder
	buf.WriteString("# ")
	buf.WriteString(f.FuncName)

	if f.Header != nil {
		buf.WriteString(f.Header.String())
	} else {
		buf.WriteString("(...)")
	}

	if f.Module != nil {
		fmt.Fprintf(&buf, "\n\n**Module:** [%s](/modules/%[1]s)", f.Module.info.Name)
	}

	if len(f.Usage) > 0 {
		buf.WriteString("\n\n")
		buf.WriteString(f.Usage)
	}
	return buf.String()
}

func (*BuiltinFunction) Type() ObjectType {
	return TBuiltinFunction
}

func (f *BuiltinFunction) Name() string {
	return f.FuncName
}

func (f *BuiltinFunction) String() string {
	return f.ToString()
}

func (f *BuiltinFunction) ToString() string {
	var header string
	if f.Header != nil {
		header = f.Header.String()
	}
	return fmt.Sprintf(ReprQuote("builtinFunction: %s%s"), f.FuncName, header)
}

func (f *BuiltinFunction) ParamTypes() ParamsTypes {
	if f.Header != nil {
		return f.Header.ParamTypes()
	}
	return nil
}

// Copy implements Copier interface.
func (f *BuiltinFunction) Copy() Object {
	return &BuiltinFunction{
		FuncName: f.FuncName,
		Value:    f.Value,
	}
}

// Equal implements Object interface.
func (f *BuiltinFunction) Equal(right Object) bool {
	v, ok := right.(*BuiltinFunction)
	if !ok {
		return false
	}
	return v == f
}

// IsFalsy implements Object interface.
func (*BuiltinFunction) IsFalsy() bool { return false }

func (f *BuiltinFunction) Call(c Call) (Object, error) {
	return f.Value(c)
}

func (f *BuiltinFunction) MethodsDisabled() bool {
	return f.AcceptMethodsDisabled
}

var (
	_ CallerObject = (*BuiltinFunctionWithMethods)(nil)
)

type BuiltinFunctionWithMethods struct {
	*FuncSpec
	name   string
	Module *Module
}

func NewBuiltinFunctionWithMethods(name string, module *Module) *BuiltinFunctionWithMethods {
	b := &BuiltinFunctionWithMethods{Module: module, name: name}
	b.FuncSpec = NewFuncSpec(b)
	return b
}

func (f *BuiltinFunctionWithMethods) Type() ObjectType {
	return TBuiltinFunction
}

func (f *BuiltinFunctionWithMethods) Equal(right Object) bool {
	return f == right
}

func (f BuiltinFunctionWithMethods) Copy() Object {
	cp := &f
	cp.FuncSpec = cp.FuncSpec.CopyWithTarget(cp)
	return cp
}

func (f *BuiltinFunctionWithMethods) GetModule() *Module {
	return f.Module
}

func (f *BuiltinFunctionWithMethods) Name() string {
	return f.name
}

func (f *BuiltinFunctionWithMethods) FullName() string {
	if f.Module != nil {
		return f.Module.info.Name + "." + f.name
	}
	return f.name
}

func (f *BuiltinFunctionWithMethods) FuncSpecName() string {
	return "builtin function " + ReprQuote(f.FullName())
}

func (f *BuiltinFunctionWithMethods) ToString() string {
	return f.String()
}

func (f *BuiltinFunctionWithMethods) String() string {
	return string(MustToStr(nil, f))
}

func (f *BuiltinFunctionWithMethods) Print(state *PrinterState) (err error) {
	return f.PrintFuncWrapper(state, f)
}

// Array represents array of objects and implements Object interface.
type Array []Object

var (
	_ Object                    = (Array)(nil)
	_ LengthGetter              = (Array)(nil)
	_ ToArrayAppenderObject     = (Array)(nil)
	_ DeepCopier                = (Array)(nil)
	_ Copier                    = (Array)(nil)
	_ Sorter                    = (Array)(nil)
	_ KeysGetter                = (Array)(nil)
	_ ItemsGetter               = (Array)(nil)
	_ SelfAssignOperatorHandler = (Array)(nil)
	_ Printer                   = (Array)(nil)
)

func (o Array) Type() ObjectType {
	return TArray
}

func (o Array) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		f.Write([]byte(o.ToString()))
	}
}

func (o Array) ToString() string {
	return ArrayToString(len(o), func(i int) Object {
		return o[i]
	})
}

func (o Array) Print(state *PrinterState) (err error) {
	if state.IsRepr {
		defer state.WrapReprString(TArray.Name())()
	}
	state.QuoteNextStr(1)
	return o.PrintObject(state, nil)
}

func (o Array) PrintObject(state *PrinterState, obj Object) (err error) {
	if obj != nil && state.IsRepr {
		defer state.WrapRepr(obj)()
	}
	return state.PrintArray(o.Length(), func(i int) (Object, error) {
		return o[i], nil
	})
}

func (o Array) ToInterface(vm *VM) any {
	return o.ToAnyArray(vm)
}

func (o Array) ToAnyArray(vm *VM) []any {
	oi := make([]any, len(o))
	for i, v := range o {
		oi[i] = vm.ToInterface(v)
	}
	return oi
}

// Copy implements Copier interface.
func (o Array) Copy() Object {
	cp := make(Array, len(o))
	copy(cp, o)
	return cp
}

// DeepCopy implements DeepCopier interface.
func (o Array) DeepCopy(vm *VM) (_ Object, err error) {
	cp := make(Array, len(o))
	for i, v := range o {
		if v, err = DeepCopy(vm, v); err != nil {
			return
		}
		cp[i] = v
	}
	return cp, nil
}

// IndexSet implements Object interface.
func (o Array) IndexSet(_ *VM, index, value Object) error {
	switch v := index.(type) {
	case Int:
		idx := int(v)
		if idx < 0 {
			idx = len(o) + idx
		}
		if idx >= 0 && idx < len(o) {
			o[idx] = value
			return nil
		}
		return ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			o[idx] = value
			return nil
		}
		return ErrIndexOutOfBounds
	}
	return NewIndexTypeError("int|uint", index.Type().Name())
}

// IndexGet implements Object interface.
func (o Array) IndexGet(_ *VM, index Object) (Object, error) {
	switch v := index.(type) {
	case Int:
		idx := int(v)
		if idx < 0 {
			idx = len(o) + idx
		}
		if idx >= 0 && idx < len(o) {
			return o[idx], nil
		}
		return nil, ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			return o[idx], nil
		}
		return nil, ErrIndexOutOfBounds
	}
	return nil, NewIndexTypeError("int|uint", index.Type().Name())
}

// Equal implements Object interface.
func (o Array) Equal(right Object) bool {
	v, ok := right.(Array)
	if !ok {
		return false
	}

	if len(o) != len(v) {
		return false
	}

	for i := range o {
		if !o[i].Equal(v[i]) {
			return false
		}
	}
	return true
}

// IsFalsy implements Object interface.
func (o Array) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o Array) BinaryOp(vm *VM, tok token.Token, right Object) (_ Object, err error) {
	switch tok {
	case token.Add:
		var arr Array
		switch t := right.(type) {
		case Str, RawStr:
			arr = make(Array, 0, len(o)+1)
			arr = append(arr, o...)
			arr = append(arr, t)
		case Iterabler:
			var values Array
			if values, err = ValuesOf(vm, t, &NamedArgs{}); err != nil {
				return
			}
			arr = make(Array, 0, len(o)+len(values))
			arr = append(arr, o...)
			arr = append(arr, values...)
		default:
			arr = make(Array, 0, len(o)+1)
			arr = append(arr, o...)
			arr = append(arr, t)
		}

		return arr, nil
	case token.Less, token.LessEq:
		if right == Nil {
			return False, nil
		}
	case token.Greater, token.GreaterEq:
		if right == Nil {
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o Array) AppendToArray(arr Array) Array {
	return append(arr, o...)
}

// Length implements LengthGetter interface.
func (o Array) Length() int {
	return len(o)
}

func (o Array) Keys() (arr Array) {
	arr = make(Array, len(o))
	for i := range o {
		arr[i] = Int(i)
	}
	return arr
}

func (o Array) Items(_ *VM, cb ItemsGetterCallback) (err error) {
	for i, v := range o {
		if err = cb(i, &KeyValue{Int(i), v}); err != nil {
			return
		}
	}
	return
}

func (o Array) Sort(vm *VM, less CallerObject) (_ Object, err error) {
	if less == nil {
		sort.Slice(o, func(i, j int) bool {
			if bo, _ := o[i].(BinaryOperatorHandler); bo != nil {
				v, e := bo.BinaryOp(vm, token.Less, o[j])
				if e != nil && err == nil {
					err = e
					return false
				}
				if v != nil {
					return !v.IsFalsy()
				}
			}
			return false
		})
	} else {
		var (
			args   = Array{Nil, Nil}
			caller VMCaller
		)

		if caller, err = NewInvoker(vm, less).Caller(Args{args}, nil); err != nil {
			return
		}

		sort.Slice(o, func(i, j int) bool {
			args[0] = o[i]
			args[1] = o[j]
			ret, _ := caller.Call()
			return !ret.IsFalsy()
		})
	}
	return o, err
}

func (o Array) SortReverse(vm *VM) (_ Object, err error) {
	sort.Slice(o, func(i, j int) bool {
		if bo, _ := o[j].(BinaryOperatorHandler); bo != nil {
			v, e := bo.BinaryOp(vm, token.Less, o[i])
			if e != nil && err == nil {
				err = e
				return false
			}
			if v != nil {
				return !v.IsFalsy()
			}
		}
		return false
	})
	return o, err
}

func (o *Array) Append(_ *VM, items ...Object) error {
	*o = append(*o, items...)
	return nil
}

func (o Array) AppendObjects(_ *VM, items ...Object) (Object, error) {
	o = append(o, items...)
	return o, nil
}

func (o Array) SelfAssignOp(vm *VM, tok token.Token, right Object) (ret Object, handled bool, err error) {
	switch tok {
	case token.Add:
		return append(o, right), true, nil
	case token.Inc:
		var other Array
		if other, err = ValuesOf(vm, right, &NamedArgs{}); err != nil {
			return
		}
		return append(o, other...), true, nil
	}
	return
}

// ObjectPtr represents a pointer variable.
type ObjectPtr struct {
	ObjectImpl
	Value *Object
}

var (
	_ Object     = (*ObjectPtr)(nil)
	_ Copier     = (*ObjectPtr)(nil)
	_ DeepCopier = (*ObjectPtr)(nil)
)

func (o *ObjectPtr) Type() ObjectType {
	return TObjectPtr
}

func (o *ObjectPtr) ToString() string {
	var v Object
	if o.Value != nil {
		v = *o.Value
	}
	return fmt.Sprintf(ReprQuote("objectPtr:%v"), v)
}

// Copy implements Copier interface.
func (o *ObjectPtr) Copy() Object {
	return o
}

// DeepCopy implements DeepCopier interface.
func (o *ObjectPtr) DeepCopy(*VM) (Object, error) {
	return o, nil
}

// IsFalsy implements Object interface.
func (o *ObjectPtr) IsFalsy() bool {
	return o.Value == nil
}

// Equal implements Object interface.
func (o *ObjectPtr) Equal(x Object) bool {
	return o == x
}

// BinaryOp implements Object interface.
func (o *ObjectPtr) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	if o.Value == nil {
		return nil, errors.New("nil pointer")
	}
	if bo, _ := (*o.Value).(BinaryOperatorHandler); bo != nil {
		return bo.BinaryOp(vm, tok, right)
	}
	return nil, ErrInvalidOperator
}

// CanCall implements Object interface.
func (o *ObjectPtr) CanCall() bool {
	if o.Value == nil {
		return false
	}
	return Callable(*o.Value)
}

// Call implements Object interface.
func (o *ObjectPtr) Call(c Call) (Object, error) {
	if o.Value == nil {
		return nil, errors.New("nil pointer")
	}
	return (*o.Value).(CallerObject).Call(c)
}

// Dict represents map of objects and implements Object interface.
type Dict map[string]Object

var (
	_ Object       = Dict{}
	_ Copier       = Dict{}
	_ IndexDeleter = Dict{}
	_ LengthGetter = Dict{}
	_ KeysGetter   = Dict{}
	_ ValuesGetter = Dict{}
	_ ItemsGetter  = Dict{}
	_ Printer      = Dict{}
)

func (o Dict) Type() ObjectType {
	return TDict
}

func (o Dict) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		f.Write([]byte(o.ToString()))
	}
}

func (o Dict) ToInterface(vm *VM) any {
	return o.ToInterfaceMap(vm)
}

func (o Dict) ToInterfaceMap(vm *VM) (m map[string]any) {
	m = make(map[string]any, len(o))
	for s, obj := range o {
		m[s] = vm.ToInterface(obj)
	}
	return m
}

func (o Dict) Filter(f func(k string, v Object) bool) Dict {
	cp := Dict{}
	for k, v := range o {
		if f(k, v) {
			cp[k] = v
		}
	}
	return cp
}

// Print prints object writing output to out writer.
// Options:
// - anonymous flag: include anonymous fields.
// - zeros flag: include zero fields.
// - sortKeys int = 0: fields sorting. 1: ASC, 2: DESC.
func (o Dict) Print(state *PrinterState) (err error) {
	return o.PrintObject(state, o)
}

func (o Dict) PrintObject(state *PrinterState, dot Object) (err error) {
	if dot != nil && state.IsRepr {
		defer state.WrapRepr(dot)()
	}

	type entry struct {
		name  string
		value Object
	}

	var (
		entries         []entry
		sortKeysType, _ = state.options.SortKeys()
	)

	for name, value := range o {
		entries = append(entries, entry{name, value})
	}

	switch sortKeysType {
	case 2:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].name > entries[j].name
		})
	default:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].name < entries[j].name
		})
	}

	return state.PrintDict(len(entries),
		func(i int) (Object, error) {
			return Str(entries[i].name), nil
		}, func(i int) (Object, error) {
			return entries[i].value, nil
		})
}

func (o Dict) ToString() string {
	var sb strings.Builder
	sb.WriteString("{")
	last := len(o) - 1
	i := 0

	var keys []string
	for k := range o {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		if runehelper.IsIdentifierRunes([]rune(k)) {
			sb.WriteString(k)
		} else {
			sb.WriteString(strconv.Quote(k))
		}
		sb.WriteString(": ")
		sb.WriteString(ToCode(o[k]))
		if i != last {
			sb.WriteString(", ")
		}
		i++
	}

	sb.WriteString("}")
	return sb.String()
}

// Copy implements Copier interface.
func (o Dict) Copy() Object {
	cp := make(Dict, len(o))
	for k, v := range o {
		cp[k] = v
	}
	return cp
}

// DeepCopy implements DeepCopier interface.
func (o Dict) DeepCopy(vm *VM) (_ Object, err error) {
	cp := make(Dict, len(o))
	for k, v := range o {
		if cp[k], err = DeepCopy(vm, v); err != nil {
			return
		}
	}
	return cp, nil
}

// IndexSet implements Object interface.
func (o Dict) IndexSet(_ *VM, index, value Object) error {
	o[index.ToString()] = value
	return nil
}

// IndexGet implements Object interface.
func (o Dict) IndexGet(_ *VM, index Object) (Object, error) {
	v, ok := o[index.ToString()]
	if ok {
		return v, nil
	}
	return Nil, nil
}

// Equal implements Object interface.
func (o Dict) Equal(right Object) bool {
	v, ok := right.(Dict)
	if !ok {
		return false
	}

	if len(o) != len(v) {
		return false
	}

	for k := range o {
		right, ok := v[k]
		if !ok {
			return false
		}
		if !o[k].Equal(right) {
			return false
		}
	}
	return true
}

// IsFalsy implements Object interface.
func (o Dict) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o Dict) BinaryOp(vm *VM, tok token.Token, right Object) (_ Object, err error) {
	if right == Nil {
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	} else {
		switch tok {
		case token.Add:
			err = IterateObject(vm, right, &NamedArgs{}, nil, func(e *KeyValue) error {
				o[e.K.ToString()] = e.V
				return nil
			})
			return o, err
		case token.Sub:
			switch t := right.(type) {
			case Array:
				for _, key := range t {
					delete(o, key.ToString())
				}
				return o, nil
			case Dict:
				for key := range t {
					delete(o, key)
				}
				return o, nil
			case Str:
				delete(o, string(t))
				return o, nil
			case KeyValueArray:
				for _, kv := range t {
					delete(o, kv.K.ToString())
				}
				return o, nil
			default:
				if Iterable(vm, right) {
					err = IterateObject(vm, right, &NamedArgs{}, nil, func(e *KeyValue) error {
						delete(o, e.K.ToString())
						return nil
					})
					return o, err
				}
			}
		}
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

// IndexDelete tries to delete the string value of key from the map.
// IndexDelete implements IndexDeleter interface.
func (o Dict) IndexDelete(_ *VM, key Object) error {
	delete(o, key.ToString())
	return nil
}

// Length implements LengthGetter interface.
func (o Dict) Length() int {
	return len(o)
}

func (o Dict) ToKeyValueArray() (arr KeyValueArray) {
	for k, v := range o {
		arr = append(arr, &KeyValue{Str(k), v})
	}
	return
}

func (o Dict) Items(_ *VM, cb ItemsGetterCallback) (err error) {
	var i int
	for k, v := range o {
		if err = cb(i, &KeyValue{Str(k), v}); err != nil {
			return
		}
		i++
	}
	return
}

func (o Dict) Keys() Array {
	var (
		arr = make(Array, len(o))
		i   int
	)
	for key := range o {
		arr[i] = Str(key)
		i++
	}
	return arr
}

func (o Dict) SortedKeys() Array {
	keys := o.Keys()
	keys.Sort(nil, nil)
	return keys
}

func (o Dict) Values() Array {
	var (
		arr = make(Array, len(o))
		i   int
	)
	for _, value := range o {
		arr[i] = value
		i++
	}
	return arr
}

func (o *Dict) Set(key string, value Object) {
	if *o == nil {
		*o = Dict{}
	}
	(*o)[key] = value
}

func (o Dict) ToNamedArgs() *NamedArgs {
	if o == nil {
		o = make(Dict, len(o))
	}

	items := o.ToKeyValueArray()

	return &NamedArgs{
		m:       o,
		ready:   Dict{},
		sources: KeyValueArrays{items},
	}
}

// Get gets object by key. Return then if exists or Nil.
func (o Dict) Get(key string) (r Object) {
	r = o[key]
	if r == nil {
		r = Nil
	}
	return
}

// Backup returns a handle function to restores key value.
func (o Dict) Backup(key string) func() {
	v := o[key]
	if v != nil {
		return func() {
			o[key] = v
		}
	} else {
		return func() {
			delete(o, key)
		}
	}
}

func (o Dict) ToDict() Dict {
	return o
}

// SyncDict represents map of objects and implements Object interface.
type SyncDict struct {
	mu    sync.RWMutex
	Value Dict
}

var (
	_ Object       = (*SyncDict)(nil)
	_ Copier       = (*SyncDict)(nil)
	_ IndexDeleter = (*SyncDict)(nil)
	_ LengthGetter = (*SyncDict)(nil)
	_ KeysGetter   = (*SyncDict)(nil)
	_ ValuesGetter = (*SyncDict)(nil)
	_ ItemsGetter  = (*SyncDict)(nil)
)

// RLock locks the underlying mutex for reading.
func (o *SyncDict) RLock() {
	o.mu.RLock()
}

// RUnlock unlocks the underlying mutex for reading.
func (o *SyncDict) RUnlock() {
	o.mu.RUnlock()
}

// Lock locks the underlying mutex for writing.
func (o *SyncDict) Lock() {
	o.mu.Lock()
}

// Unlock unlocks the underlying mutex for writing.
func (o *SyncDict) Unlock() {
	o.mu.Unlock()
}

func (o *SyncDict) Type() ObjectType {
	return TSyncDict
}

func (o *SyncDict) ToString() string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.ToString()
}

func (o *SyncDict) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('#') {
			f.Write([]byte("&" + reflect.TypeOf(o).Elem().String()))
		}
		o.Value.Format(f, verb)
	}
}

// Copy implements Copier interface.
func (o *SyncDict) Copy() Object {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return &SyncDict{
		Value: o.Value.Copy().(Dict),
	}
}

// DeepCopy implements DeepCopier interface.
func (o *SyncDict) DeepCopy(vm *VM) (v Object, err error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if v, err = o.Value.DeepCopy(vm); err != nil {
		return
	}

	return &SyncDict{
		Value: v.(Dict),
	}, nil
}

// IndexSet implements Object interface.
func (o *SyncDict) IndexSet(vm *VM, index, value Object) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.Value == nil {
		o.Value = Dict{}
	}
	return o.Value.IndexSet(vm, index, value)
}

// IndexGet implements Object interface.
func (o *SyncDict) IndexGet(vm *VM, index Object) (Object, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.IndexGet(vm, index)
}

// Equal implements Object interface.
func (o *SyncDict) Equal(right Object) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.Equal(right)
}

// IsFalsy implements Object interface.
func (o *SyncDict) IsFalsy() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.IsFalsy()
}

// Get returns Object in map if exists.
func (o *SyncDict) Get(index string) (value Object, exists bool) {
	o.mu.RLock()
	value, exists = o.Value[index]
	o.mu.RUnlock()
	return
}

// Length returns the number of items in the dict.
func (o *SyncDict) Length() int {
	o.mu.RLock()
	n := len(o.Value)
	o.mu.RUnlock()
	return n
}

// IndexDelete tries to delete the string value of key from the map.
func (o *SyncDict) IndexDelete(vm *VM, key Object) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.Value.IndexDelete(vm, key)
}

// BinaryOp implements Object interface.
func (o *SyncDict) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.BinaryOp(vm, tok, right)
}

func (o *SyncDict) Items(vm *VM, cb ItemsGetterCallback) (err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Value.Items(vm, cb)
}

func (o *SyncDict) Keys() Array {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Value.Keys()
}

func (o *SyncDict) Values() Array {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Value.Values()
}

type CallWrapper struct {
	Caller    CallerObject
	Args      Args
	NamedArgs KeyValueArray
}

func (i *CallWrapper) Name() string {
	return "wrap"
}

func NewCallWrapper(caller CallerObject, args Args, namedArgs KeyValueArray) *CallWrapper {
	return &CallWrapper{Caller: caller, Args: args, NamedArgs: namedArgs}
}

func (i *CallWrapper) Call(c Call) (Object, error) {
	args := append(i.Args, c.Args...)
	nargs := NamedArgs{sources: KeyValueArrays{i.NamedArgs}}
	if len(c.NamedArgs.sources) > 0 {
		nargs.Add(c.NamedArgs.UnreadPairs())
	}
	return i.Caller.Call(Call{VM: c.VM, Args: args, NamedArgs: nargs, SafeArgs: c.SafeArgs})
}

func (i *CallWrapper) Type() ObjectType {
	return TCallWrapper
}

func (i *CallWrapper) ToString() string {
	return i.Type().ToString() + "{" + i.Caller.ToString() + "}"
}

func (CallWrapper) IsFalsy() bool {
	return false
}

func (CallWrapper) Equal(Object) bool {
	return false
}

var (
	_ IndexGetter      = (*IndexGetProxy)(nil)
	_ Iterabler        = (*IndexGetProxy)(nil)
	_ CanIterabler     = (*IndexGetProxy)(nil)
	_ NameCallerObject = (*IndexGetProxy)(nil)
)

type IndexGetProxy struct {
	GetIndexFunc   func(vm *VM, index Object) (value Object, err error)
	ToStrFunc      func() string
	PrintFunc      func(s *PrinterState) error
	IterateFunc    func(vm *VM, na *NamedArgs) Iterator
	InterfaceValue any
	CallNameFunc   func(name string, c Call) (Object, error)
}

func (i *IndexGetProxy) Print(state *PrinterState) error {
	if i.PrintFunc == nil {
		_, err := state.Write([]byte(i.ToString()))
		return err
	} else {
		return i.PrintFunc(state)
	}
}

func (i *IndexGetProxy) CallName(name string, c Call) (Object, error) {
	if i.CallNameFunc == nil {
		return nil, ErrInvalidIndex.NewError(name)
	}
	return i.CallNameFunc(name, c)
}

func (i *IndexGetProxy) Iterate(vm *VM, na *NamedArgs) Iterator {
	return i.IterateFunc(vm, na)
}

func (i *IndexGetProxy) CanIterate() bool {
	return i.IterateFunc != nil
}

func (i *IndexGetProxy) Type() ObjectType {
	return TIndexGetProxy
}

func (i *IndexGetProxy) ToString() string {
	if i.ToStrFunc != nil {
		return i.ToStrFunc()
	}
	return ReprQuote("indexGetProxy")
}

func (i *IndexGetProxy) IsFalsy() bool {
	return false
}

func (i *IndexGetProxy) Equal(right Object) bool {
	if ri, _ := right.(*IndexGetProxy); ri != nil {
		return &ri == &i
	}
	return false
}

func (i IndexGetProxy) IndexGet(vm *VM, index Object) (value Object, err error) {
	return i.GetIndexFunc(vm, index)
}

func (i *IndexGetProxy) ToInterface() any {
	return i.InterfaceValue
}

type IndexSetProxy struct {
	Set func(vm *VM, key, value Object) error
}

func (s *IndexSetProxy) IndexSet(vm *VM, index, value Object) error {
	return s.Set(vm, index, value)
}

type IndexDelProxy struct {
	Del func(vm *VM, key Object) error
}

func (p *IndexDelProxy) IndexDelete(vm *VM, key Object) error {
	return p.Del(vm, key)
}

var (
	_ IndexSetter = (*IndexProxy)(nil)
)

type IndexProxy struct {
	IndexGetProxy
	IndexSetProxy
}

// BinaryOp implements Object interface.
func (o *IndexProxy) BinaryOp(vm *VM, tok token.Token, right Object) (_ Object, err error) {
	if right == Nil {
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	} else {
		switch tok {
		case token.Add:
			if Iterable(vm, right) {
				err = IterateObject(vm, right, &NamedArgs{}, nil, func(e *KeyValue) error {
					return o.Set(vm, e.K, e.V)
				})
				return o, err
			}
		}
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

var (
	_ Indexer               = (*IndexDeleteProxy)(nil)
	_ BinaryOperatorHandler = (*IndexDeleteProxy)(nil)
)

type IndexDeleteProxy struct {
	IndexGetProxy
	IndexSetProxy
	IndexDelProxy
}

// BinaryOp implements Object interface.
func (o *IndexDeleteProxy) BinaryOp(vm *VM, tok token.Token, right Object) (_ Object, err error) {
	if right == Nil {
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	} else {
		switch tok {
		case token.Add:
			if Iterable(vm, right) {
				err = IterateObject(vm, right, &NamedArgs{}, nil, func(e *KeyValue) error {
					return o.Set(vm, e.K, e.V)
				})
				return o, err
			}
		case token.Sub:
			switch t := right.(type) {
			case Array:
				for _, key := range t {
					if err = o.Del(vm, key); err != nil {
						return
					}
				}
				return o, nil
			case Dict:
				for key := range t {
					if err = o.Del(vm, Str(key)); err != nil {
						return
					}
				}
				return o, nil
			case KeyValueArray:
				for _, kv := range t {
					if err = o.Del(vm, kv.K); err != nil {
						return
					}
				}
				return o, nil
			default:
				if Iterable(vm, right) {
					err = IterateObject(vm, right, &NamedArgs{}, nil, func(e *KeyValue) error {
						return o.Del(vm, e.K)
					})
					return o, err
				}
			}
		}
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func StringIndexGetProxy(handler func(vm *VM, index string) (value Object, err error)) *IndexGetProxy {
	return &IndexGetProxy{
		GetIndexFunc: func(vm *VM, index Object) (value Object, err error) {
			if s, ok := index.(Str); !ok {
				return nil, ErrInvalidIndex
			} else {
				return handler(vm, string(s))
			}
		},
	}
}

var (
	_ Object      = (*MixedParams)(nil)
	_ IndexGetter = (*MixedParams)(nil)
)

type MixedParams struct {
	Positional Array
	Named      KeyValueArray
}

func (m *MixedParams) IndexGet(vm *VM, index Object) (value Object, err error) {
	switch t := index.(type) {
	case Str:
		switch t {
		case "positional":
			return m.Positional, nil
		case "named":
			return m.Named, nil
		}
		return Nil, ErrInvalidIndex.NewError(string(t))
	default:
		return m.Positional.IndexGet(vm, t)
	}
}

func (m *MixedParams) IsFalsy() bool {
	return m.Positional.IsFalsy() && m.Named.IsFalsy()
}

func (m *MixedParams) Type() ObjectType {
	return TMixedParams
}

func (m *MixedParams) ToDict() (d Dict) {
	d = Dict{}
	m.UpdateDict(d)
	return
}

func (m *MixedParams) UpdateDict(d Dict) {
	d["positional"] = m.Positional
	d["named"] = m.Named
}

func (m *MixedParams) ToString() string {
	return m.ToDict().ToString()
}

func (m *MixedParams) Equal(right Object) bool {
	switch t := right.(type) {
	case *MixedParams:
		return m.Positional.Equal(t.Positional) && m.Named.Equal(t.Named)
	default:
		return false
	}
}

func (m *MixedParams) Print(state *PrinterState) error {
	return m.ToDict().Print(state)
}

var (
	_ Object         = (*TypedIdent)(nil)
	_ IndexGetSetter = (*TypedIdent)(nil)
	_ Printer        = (*TypedIdent)(nil)
)

type TypedIdent struct {
	Name  string
	Types Array
}

func (t *TypedIdent) IndexSet(_ *VM, index, value Object) (err error) {
	name := index.ToString()
	switch name {
	case "name":
		if s, ok := value.(Str); ok {
			t.Name = string(s)
		} else {
			err = ErrType
		}
	case "types":
		if arr, ok := value.(Array); ok {
			for i, o := range arr {
				if !IsType(o) {
					err = ErrType.NewErrorf("type[%d]: isn't type", i)
					return
				}
			}
			t.Types = arr
		} else {
			err = ErrType
		}
	default:
		err = ErrInvalidIndex.NewError(name)
	}
	return
}

func (t *TypedIdent) IndexGet(_ *VM, index Object) (value Object, err error) {
	name := index.ToString()
	switch name {
	case "name":
		value = Str(t.Name)
	case "types":
		value = t.Types
	default:
		err = ErrInvalidIndex.NewError(name)
	}
	return
}

func (t *TypedIdent) IsFalsy() bool {
	return len(t.Name) == 0
}

func (t *TypedIdent) Type() ObjectType {
	return TTypedIdent
}

func (t *TypedIdent) ToString() string {
	return ReprQuote("typedIdent " + t.Name + " " + t.Types.ToString())
}

func (t *TypedIdent) Equal(right Object) bool {
	if r, _ := right.(*TypedIdent); r != nil {
		if r == t {
			return true
		}
		if t.Name == r.Name {
			return t.Types.Equal(r.Types)
		}
	}
	return false
}

func (t *TypedIdent) Print(state *PrinterState) error {
	return Dict{"name": Str(t.Name), "types": t.Types}.PrintObject(state, t)
}

var (
	_ Object       = (*ComputedValue)(nil)
	_ CallerObject = (*ComputedValue)(nil)
)

type ComputedValue struct {
	CallerObject CallerObject
}

func (v *ComputedValue) Call(c Call) (Object, error) {
	return v.CallerObject.Call(c)
}

func (v *ComputedValue) Name() string {
	return "computed value " + repr.Quote(v.CallerObject.Name())
}

func (v *ComputedValue) IsFalsy() bool {
	return false
}

func (v *ComputedValue) ToString() string {
	return v.String()
}

func (v *ComputedValue) Equal(right Object) bool {
	if r, _ := right.(*ComputedValue); r != nil {
		return r == v || r.CallerObject.Equal(v.CallerObject)
	}
	return false
}

func (ComputedValue) Type() ObjectType {
	return TComputedValue
}

func (v *ComputedValue) String() string {
	return string(MustToStr(nil, v))
}

func (v *ComputedValue) ReprTypeName() string {
	return "computed value"
}

func (v *ComputedValue) Print(state *PrinterState) error {
	return state.WithRepr(func(s *PrinterState) error {
		defer state.WrapRepr(v)()
		return s.Print(v.CallerObject)
	})
}
