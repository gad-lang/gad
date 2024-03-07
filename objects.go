// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gad-lang/gad/internal/compat"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/utils"
	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

// Bool represents boolean values and implements Object interface.
type Bool bool

func (o Bool) Type() ObjectType {
	return DetectTypeOf(o)
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

func (o RawStr) Repr(*VM) (string, error) {
	return repr.Quote("rawstr:" + o.Quoted()), nil
}

func (o RawStr) Quoted() string {
	return utils.Quote(string(o), '`')
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
	_ LengthGetter      = Str("")
	_ ObjectRepresenter = Str("")
)

func (o Str) Type() ObjectType {
	return DetectTypeOf(o)
}

func (o Str) ToString() string {
	return string(o)
}

func (o Str) Repr(*VM) (string, error) {
	return repr.Quote("str:" + o.Quoted()), nil
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

// Bytes represents byte slice and implements Object interface.
type Bytes []byte

var (
	_ Object       = Bytes{}
	_ Copier       = Bytes{}
	_ LengthGetter = Bytes{}
)

func (o Bytes) Type() ObjectType {
	return DetectTypeOf(o)
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

// Format implements fmt.Formatter interface.
func (o Bytes) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, []byte(o))
}

func (o Bytes) WriteTo(_ *VM, w io.Writer) (int64, error) {
	n, err := w.Write(o)
	return int64(n), err
}

// Function represents a function object and implements Object interface.
type Function struct {
	ObjectImpl
	Name  string
	Value func(Call) (Object, error)
	ToStr func() string
}

var _ Object = (*Function)(nil)

func (*Function) Type() ObjectType {
	return TFunction
}

func (o *Function) ToString() string {
	var name = o.Name
	if o.ToStr != nil {
		name = o.ToStr()
	}
	return fmt.Sprintf(ReprQuote("function:%s"), name)
}

// Copy implements Copier interface.
func (o *Function) Copy() Object {
	return &Function{
		Name:  o.Name,
		Value: o.Value,
	}
}

// Equal implements Object interface.
func (o *Function) Equal(right Object) bool {
	v, ok := right.(*Function)
	if !ok {
		return false
	}
	return v == o
}

// IsFalsy implements Object interface.
func (*Function) IsFalsy() bool { return false }

func (o *Function) Call(call Call) (Object, error) {
	return o.Value(call)
}

type ArgType []ObjectType

func (t ArgType) String() string {
	l := len(t)
	switch l {
	case 0:
		return ""
	case 1:
		return ":" + t[0].Name()
	default:
		var s = make([]string, l)
		for i, t2 := range t {
			s[i] = t2.Name()
		}
		return "::[" + strings.Join(s, ", ") + "]"
	}
}

type FunctionHeaderParam struct {
	Name  string
	Types []ObjectType
	Value string
}

func (p *FunctionHeaderParam) String() string {
	var (
		s = p.Name
		l = len(p.Types)
	)
	switch l {
	case 0:
	case 1:
		s += ":" + p.Types[0].Name()
	default:
		var s2 = make([]string, l)
		for i, t2 := range p.Types {
			s2[i] = t2.Name()
		}
		s += ":[" + strings.Join(s2, ", ") + "]"
	}
	if p.Value != "" {
		s += "=" + p.Value
	}
	return s
}

type FunctionHeader struct {
	Params      []Params
	NamedParams []Params
}

func (h *FunctionHeader) String() string {
	var s []string
	for _, param := range h.Params {
		s = append(s, param.String())
	}
	for _, param := range h.NamedParams {
		s = append(s, param.String())
	}
	return "(" + strings.Join(s, ", ") + ")"
}

// BuiltinFunction represents a builtin function object and implements Object interface.
type BuiltinFunction struct {
	ObjectImpl
	Name                  string
	Value                 func(Call) (Object, error)
	Header                FunctionHeader
	AcceptMethodsDisabled bool
}

var _ CallerObject = (*BuiltinFunction)(nil)

func (*BuiltinFunction) Type() ObjectType {
	return TBuiltinFunction
}

func (o *BuiltinFunction) ToString() string {
	return fmt.Sprintf(ReprQuote("builtinFunction:%s%s"), o.Name, o.Header.String())
}

func (o *BuiltinFunction) ParamTypes(*VM) (MultipleObjectTypes, error) {
	return nil, nil
}

// Copy implements Copier interface.
func (o *BuiltinFunction) Copy() Object {
	return &BuiltinFunction{
		Name:  o.Name,
		Value: o.Value,
	}
}

// Equal implements Object interface.
func (o *BuiltinFunction) Equal(right Object) bool {
	v, ok := right.(*BuiltinFunction)
	if !ok {
		return false
	}
	return v == o
}

// IsFalsy implements Object interface.
func (*BuiltinFunction) IsFalsy() bool { return false }

func (o *BuiltinFunction) Call(c Call) (Object, error) {
	return o.Value(c)
}

func (o *BuiltinFunction) MethodsDisabled() bool {
	return o.AcceptMethodsDisabled
}

// Array represents array of objects and implements Object interface.
type Array []Object

var (
	_ Object                = Array{}
	_ LengthGetter          = Array{}
	_ ToArrayAppenderObject = Array{}
	_ DeepCopier            = Array{}
	_ Copier                = Array{}
	_ Sorter                = Array{}
	_ KeysGetter            = Array{}
	_ ItemsGetter           = Array{}
	_ ObjectRepresenter     = Array{}
)

func (o Array) Type() ObjectType {
	return DetectTypeOf(o)
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

func (o Array) Repr(vm *VM) (string, error) {
	return ArrayRepr(o.Type().Name(), vm, len(o), func(i int) Object {
		return o[i]
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
		if idx >= 0 && idx < len(o) {
			o[v] = value
			return nil
		}
		return ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			o[v] = value
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
		if idx >= 0 && idx < len(o) {
			return o[v], nil
		}
		return nil, ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			return o[v], nil
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

func (o Array) AppendToArray(arr *Array) {
	*arr = append(*arr, o...)
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

func (o Array) Items(*VM) (arr KeyValueArray, _ error) {
	arr = make(KeyValueArray, len(o))
	for i, v := range o {
		arr[i] = &KeyValue{Str(strconv.Itoa(i)), v}
	}
	return arr, nil
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
	_ Object            = Dict{}
	_ Copier            = Dict{}
	_ IndexDeleter      = Dict{}
	_ LengthGetter      = Dict{}
	_ KeysGetter        = Dict{}
	_ ValuesGetter      = Dict{}
	_ ItemsGetter       = Dict{}
	_ ObjectRepresenter = Dict{}
)

func (o Dict) Type() ObjectType {
	return DetectTypeOf(o)
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

func (o Dict) Repr(vm *VM) (_ string, err error) {
	var (
		keys  = o.SortedKeys()
		last  = len(keys) - 1
		sb    strings.Builder
		do    = vm.Builtins.ArgsInvoker(BuiltinRepr, Call{VM: vm})
		repro Object
	)
	sb.WriteString(repr.QuotePrefix)
	sb.WriteString(o.Type().Name() + ":{")

	for i, k := range keys {
		k := string(k.(Str))
		if repro, err = do(o[k]); err != nil {
			return
		}

		if runehelper.IsIdentifierRunes([]rune(k)) {
			sb.WriteString(k)
		} else {
			sb.WriteString(strconv.Quote(k))
		}
		sb.WriteString(": ")
		sb.WriteString(repro.ToString())
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString("}")
	sb.WriteString(repr.QuoteSufix)
	return sb.String(), nil
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

func (o Dict) Items(*VM) (KeyValueArray, error) {
	var (
		arr = make(KeyValueArray, len(o))
		i   int
	)
	for key, value := range o {
		arr[i] = &KeyValue{Str(key), value}
		i++
	}
	return arr, nil
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
	return DetectTypeOf(o)
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

func (o *SyncDict) Items(vm *VM) (KeyValueArray, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Value.Items(vm)
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

// Error represents Error Object and implements error and Object interfaces.
type Error struct {
	Name    string
	Message string
	Cause   error
}

var (
	_ Object = (*Error)(nil)
	_ Copier = (*Error)(nil)
)

func (o *Error) Unwrap() error {
	return o.Cause
}

func (o *Error) Type() ObjectType {
	return DetectTypeOf(o)
}

func (o *Error) ToString() string {
	return o.Error()
}

// Copy implements Copier interface.
func (o *Error) Copy() Object {
	return &Error{
		Name:    o.Name,
		Message: o.Message,
		Cause:   o.Cause,
	}
}

// Error implements error interface.
func (o *Error) Error() string {
	name := o.Name
	if name == "" {
		name = "error"
	}
	return fmt.Sprintf("%s: %s", name, o.Message)
}

// Equal implements Object interface.
func (o *Error) Equal(right Object) bool {
	if v, ok := right.(*Error); ok {
		return v == o
	}
	return false
}

// IsFalsy implements Object interface.
func (o *Error) IsFalsy() bool { return true }

// IndexGet implements Object interface.
func (o *Error) IndexGet(_ *VM, index Object) (Object, error) {
	s := index.ToString()
	if s == "Literal" {
		return Str(o.Name), nil
	}

	if s == "Message" {
		return Str(o.Message), nil
	}

	if s == "New" {
		return &Function{
			Name: "New",
			Value: func(c Call) (Object, error) {
				l := c.Args.Length()
				switch l {
				case 1:
					return o.NewError(c.Args.Get(0).ToString()), nil
				case 0:
					return o.NewError(o.Message), nil
				default:
					msgs := make([]string, l)
					for i := range msgs {
						msgs[i] = c.Args.Get(i).ToString()
					}
					return o.NewError(msgs...), nil
				}
			},
		}, nil
	}
	return Nil, nil
}

// NewError creates a new Error and sets original Error as its cause which can be unwrapped.
func (o *Error) NewError(messages ...string) *Error {
	cp := o.Copy().(*Error)
	cp.Message = strings.Join(messages, " ")
	cp.Cause = o
	return cp
}

// RuntimeError represents a runtime error that wraps Error and includes trace information.
type RuntimeError struct {
	Err     *Error
	fileSet *parser.SourceFileSet
	Trace   []source.Pos
}

var (
	_ Object = (*RuntimeError)(nil)
	_ Copier = (*RuntimeError)(nil)
)

func (o *RuntimeError) Unwrap() error {
	if o.Err != nil {
		return o.Err
	}
	return nil
}

func (o *RuntimeError) addTrace(pos source.Pos) {
	if len(o.Trace) > 0 {
		if o.Trace[len(o.Trace)-1] == pos {
			return
		}
	}
	o.Trace = append(o.Trace, pos)
}

func (*RuntimeError) Type() ObjectType {
	return TError
}

func (o *RuntimeError) ToString() string {
	return o.Error()
}

// Copy implements Copier interface.
func (o *RuntimeError) Copy() Object {
	var err *Error
	if o.Err != nil {
		err = o.Err.Copy().(*Error)
	}

	return &RuntimeError{
		Err:     err,
		fileSet: o.fileSet,
		Trace:   append([]source.Pos{}, o.Trace...),
	}
}

// Error implements error interface.
func (o *RuntimeError) Error() string {
	if o.Err == nil {
		return ReprQuote("nil")
	}
	return o.Err.Error()
}

// Equal implements Object interface.
func (o *RuntimeError) Equal(right Object) bool {
	if o.Err != nil {
		return o.Err.Equal(right)
	}
	return false
}

// IsFalsy implements Object interface.
func (o *RuntimeError) IsFalsy() bool { return true }

// IndexGet implements Object interface.
func (o *RuntimeError) IndexGet(vm *VM, index Object) (Object, error) {
	if o.Err != nil {
		s := index.ToString()
		if s == "New" {
			return &Function{
				Name: "New",
				Value: func(c Call) (Object, error) {
					l := c.Args.Length()
					switch l {
					case 1:
						return o.NewError(c.Args.Get(0).ToString()), nil
					case 0:
						return o.NewError(o.Err.Message), nil
					default:
						msgs := make([]string, l)
						for i := range msgs {
							msgs[i] = c.Args.Get(i).ToString()
						}
						return o.NewError(msgs...), nil
					}
				},
			}, nil
		}
		return o.Err.IndexGet(vm, index)
	}

	return Nil, nil
}

// NewError creates a new Error and sets original Error as its cause which can be unwrapped.
func (o *RuntimeError) NewError(messages ...string) *RuntimeError {
	cp := o.Copy().(*RuntimeError)
	cp.Err.Message = strings.Join(messages, " ")
	cp.Err.Cause = o
	return cp
}

// StackTrace returns stack trace if set otherwise returns nil.
func (o *RuntimeError) StackTrace() StackTrace {
	if o.fileSet == nil {
		if o.Trace != nil {
			sz := len(o.Trace)
			trace := make(StackTrace, sz)
			j := 0
			for i := sz - 1; i >= 0; i-- {
				trace[j] = parser.SourceFilePos{
					Offset: int(o.Trace[i]),
				}
				j++
			}
			return trace
		}
		return nil
	}

	sz := len(o.Trace)
	trace := make(StackTrace, sz)
	j := 0
	for i := sz - 1; i >= 0; i-- {
		trace[j] = o.fileSet.Position(o.Trace[i])
		j++
	}
	return trace
}

// Format implements fmt.Formater interface.
func (o *RuntimeError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		switch {
		case s.Flag('+'):
			_, _ = io.WriteString(s, o.ToString())
			if len(o.Trace) > 0 {
				if v := o.StackTrace(); v != nil {
					_, _ = io.WriteString(s, fmt.Sprintf("%+v", v))
				} else {
					_, _ = io.WriteString(s, ReprQuote("nil stack trace"))
				}
			} else {
				_, _ = io.WriteString(s, ReprQuote("no stack trace"))
			}
			e := o.Unwrap()
			for e != nil {
				if e, ok := e.(*RuntimeError); ok && o != e {
					_, _ = fmt.Fprintf(s, "\n\t%+v", e)
				}
				if err, ok := e.(interface{ Unwrap() error }); ok {
					e = err.Unwrap()
				} else {
					break
				}
			}
		default:
			_, _ = io.WriteString(s, o.ToString())
		}
	case 'q':
		_, _ = io.WriteString(s, strconv.Quote(o.ToString()))
	}
}

// StackTrace is the stack of source file positions.
type StackTrace []parser.SourceFilePos

// Format formats the StackTrace to the fmt.Formatter interface.
func (st StackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		switch {
		case s.Flag('+'):
			for i, f := range st {
				if i > 0 {
					_, _ = io.WriteString(s, "\n\t   ")
				} else {
					_, _ = io.WriteString(s, "\n\tat ")
				}
				_, _ = fmt.Fprintf(s, "%+v", f)
			}
		default:
			_, _ = fmt.Fprintf(s, "%v", []parser.SourceFilePos(st))
		}
	}
}

type CallWrapper struct {
	Caller    CallerObject
	Args      Args
	NamedArgs KeyValueArray
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

type IndexGetProxy struct {
	GetIndex        func(vm *VM, index Object) (value Object, err error)
	ToStr           func() string
	It              func(vm *VM, na *NamedArgs) Iterator
	InterfaceValue  any
	CallNameHandler func(name string, c Call) (Object, error)
}

func (i *IndexGetProxy) CallName(name string, c Call) (Object, error) {
	if i.CallNameHandler == nil {
		return nil, ErrInvalidIndex.NewError(name)
	}
	return i.CallNameHandler(name, c)
}

func (i *IndexGetProxy) Iterate(vm *VM, na *NamedArgs) Iterator {
	return i.It(vm, na)
}

func (i *IndexGetProxy) CanIterate() bool {
	return i.It != nil
}

func (i *IndexGetProxy) Type() ObjectType {
	return TIndexGetProxy
}

func (i *IndexGetProxy) ToString() string {
	if i.ToStr != nil {
		return i.ToStr()
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
	return i.GetIndex(vm, index)
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

type IndexProxy struct {
	IndexGetProxy
	IndexSetProxy
	IndexDelProxy
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
	return &IndexGetProxy{GetIndex: func(vm *VM, index Object) (value Object, err error) {
		if s, ok := index.(Str); !ok {
			return nil, ErrInvalidIndex
		} else {
			return handler(vm, string(s))
		}
	}}
}
