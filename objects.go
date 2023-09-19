// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gad-lang/gad/internal/compat"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/token"
)

// Bool represents boolean values and implements Object interface.
type Bool bool

// TypeName implements Object interface.
func (Bool) TypeName() string {
	return "bool"
}

// String implements Object interface.
func (o Bool) String() string {
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
func (o Bool) BinaryOp(tok token.Token, right Object) (Object, error) {
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
		o.TypeName(),
		right.TypeName())
}

// Format implements fmt.Formatter interface.
func (o Bool) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, bool(o))
}

// String represents string values and implements Object interface.
type String string

var _ LengthGetter = String("")

// TypeName implements Object interface.
func (String) TypeName() string {
	return "string"
}

func (o String) String() string {
	return string(o)
}

// Iterate implements Object interface.
func (o String) Iterate() Iterator {
	return &StringIterator{V: o}
}

// IndexGet represents string values and implements Object interface.
func (o String) IndexGet(index Object) (Object, error) {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
	case Uint:
		idx = int(v)
	case Char:
		idx = int(v)
	default:
		return nil, NewIndexTypeError("int|uint|char", index.TypeName())
	}
	if idx >= 0 && idx < len(o) {
		return Int(o[idx]), nil
	}
	return nil, ErrIndexOutOfBounds
}

// Equal implements Object interface.
func (o String) Equal(right Object) bool {
	if v, ok := right.(String); ok {
		return o == v
	}
	if v, ok := right.(Bytes); ok {
		return string(o) == string(v)
	}
	return false
}

// IsFalsy implements Object interface.
func (o String) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o String) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch v := right.(type) {
	case String:
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
			return String(sb.String()), nil
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
		return o + String(right.String()), nil
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.TypeName(),
		right.TypeName())
}

// Len implements LengthGetter interface.
func (o String) Len() int {
	return len(o)
}

// Format implements fmt.Formatter interface.
func (o String) Format(s fmt.State, verb rune) {
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

// TypeName implements Object interface.
func (Bytes) TypeName() string {
	return "bytes"
}

func (o Bytes) String() string {
	return string(o)
}

// Copy implements Copier interface.
func (o Bytes) Copy() Object {
	cp := make(Bytes, len(o))
	copy(cp, o)
	return cp
}

// Iterate implements Object interface.
func (o Bytes) Iterate() Iterator {
	return &BytesIterator{V: o}
}

// IndexSet implements Object interface.
func (o Bytes) IndexSet(index, value Object) error {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
	case Uint:
		idx = int(v)
	default:
		return NewIndexTypeError("int|uint", index.TypeName())
	}

	if idx >= 0 && idx < len(o) {
		switch v := value.(type) {
		case Int:
			o[idx] = byte(v)
		case Uint:
			o[idx] = byte(v)
		default:
			return NewIndexValueTypeError("int|uint", value.TypeName())
		}
		return nil
	}
	return ErrIndexOutOfBounds
}

// IndexGet represents string values and implements Object interface.
func (o Bytes) IndexGet(index Object) (Object, error) {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
	case Uint:
		idx = int(v)
	default:
		return nil, NewIndexTypeError("int|uint|char", index.TypeName())
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

	if v, ok := right.(String); ok {
		return string(o) == string(v)
	}
	return false
}

// IsFalsy implements Object interface.
func (o Bytes) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o Bytes) BinaryOp(tok token.Token, right Object) (Object, error) {
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
	case String:
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
		o.TypeName(),
		right.TypeName())
}

// Len implements LengthGetter interface.
func (o Bytes) Len() int {
	return len(o)
}

// Format implements fmt.Formatter interface.
func (o Bytes) Format(s fmt.State, verb rune) {
	format := compat.FmtFormatString(s, verb)
	fmt.Fprintf(s, format, []byte(o))
}

// Function represents a function object and implements Object interface.
type Function struct {
	ObjectImpl
	Name  string
	Value func(Call) (Object, error)
}

var _ Object = (*Function)(nil)

// TypeName implements Object interface.
func (*Function) TypeName() string {
	return "function"
}

// String implements Object interface.
func (o *Function) String() string {
	return fmt.Sprintf("<function:%s>", o.Name)
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

// BuiltinFunction represents a builtin function object and implements Object interface.
type BuiltinFunction struct {
	ObjectImpl
	Name  string
	Value func(Call) (Object, error)
}

var _ CallerObject = (*BuiltinFunction)(nil)

// TypeName implements Object interface.
func (*BuiltinFunction) TypeName() string {
	return "builtinFunction"
}

// String implements Object interface.
func (o *BuiltinFunction) String() string {
	return fmt.Sprintf("<builtinFunction:%s>", o.Name)
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
)

// TypeName implements Object interface.
func (Array) TypeName() string {
	return "array"
}

// String implements Object interface.
func (o Array) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	last := len(o) - 1

	for i := range o {
		switch v := o[i].(type) {
		case String:
			sb.WriteString(strconv.Quote(v.String()))
		case Char:
			sb.WriteString(strconv.QuoteRune(rune(v)))
		case Bytes:
			sb.WriteString(fmt.Sprint([]byte(v)))
		default:
			sb.WriteString(v.String())
		}
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString("]")
	return sb.String()
}

// Copy implements Copier interface.
func (o Array) Copy() Object {
	cp := make(Array, len(o))
	copy(cp, o)
	return cp
}

// DeepCopy implements DeepCopier interface.
func (o Array) DeepCopy() Object {
	cp := make(Array, len(o))
	for i, v := range o {
		switch t := v.(type) {
		case DeepCopier:
			cp[i] = t.DeepCopy()
		case Copier:
			cp[i] = t.Copy()
		default:
			cp[i] = v
		}
	}
	return cp
}

// IndexSet implements Object interface.
func (o Array) IndexSet(index, value Object) error {
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
	return NewIndexTypeError("int|uint", index.TypeName())
}

// IndexGet implements Object interface.
func (o Array) IndexGet(index Object) (Object, error) {
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
	return nil, NewIndexTypeError("int|uint", index.TypeName())
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
func (o Array) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch tok {
	case token.Add:
		if v, ok := right.(Array); ok {
			arr := make(Array, 0, len(o)+len(v))
			arr = append(arr, o...)
			arr = append(arr, v...)
			return arr, nil
		}

		arr := make(Array, 0, len(o)+1)
		arr = append(arr, o...)
		arr = append(arr, right)
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
		o.TypeName(),
		right.TypeName())
}

func (o Array) AppendToArray(arr *Array) {
	*arr = append(*arr, o...)
}

// Iterate implements Iterable interface.
func (o Array) Iterate() Iterator {
	return &ArrayIterator{V: o}
}

// Len implements LengthGetter interface.
func (o Array) Len() int {
	return len(o)
}

func (o Array) Keys() (arr Array) {
	arr = make(Array, len(o))
	for i := range o {
		arr[i] = Int(i)
	}
	return arr
}

func (o Array) Items() (arr KeyValueArray) {
	arr = make(KeyValueArray, len(o))
	for i, v := range o {
		arr[i] = KeyValue{String(strconv.Itoa(i)), v}
	}
	return arr
}

func (o Array) Sort() (_ Object, err error) {
	sort.Slice(o, func(i, j int) bool {
		if bo, _ := o[i].(BinaryOperatorHandler); bo != nil {
			v, e := bo.BinaryOp(token.Less, o[j])
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

func (o Array) SortReverse() (_ Object, err error) {
	sort.Slice(o, func(i, j int) bool {
		if bo, _ := o[j].(BinaryOperatorHandler); bo != nil {
			v, e := bo.BinaryOp(token.Less, o[i])
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

// TypeName implements Object interface.
func (o *ObjectPtr) TypeName() string {
	return "objectPtr"
}

// String implements Object interface.
func (o *ObjectPtr) String() string {
	var v Object
	if o.Value != nil {
		v = *o.Value
	}
	return fmt.Sprintf("<objectPtr:%v>", v)
}

// Copy implements Copier interface.
func (o *ObjectPtr) Copy() Object {
	return o
}

// DeepCopy implements DeepCopier interface.
func (o *ObjectPtr) DeepCopy() Object {
	return o
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
func (o *ObjectPtr) BinaryOp(tok token.Token, right Object) (Object, error) {
	if o.Value == nil {
		return nil, errors.New("nil pointer")
	}
	if bo, _ := (*o.Value).(BinaryOperatorHandler); bo != nil {
		return bo.BinaryOp(tok, right)
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

// Map represents map of objects and implements Object interface.
type Map map[string]Object

var (
	_ Object       = Map{}
	_ Copier       = Map{}
	_ IndexDeleter = Map{}
	_ LengthGetter = Map{}
	_ KeysGetter   = Map{}
	_ ValuesGetter = Map{}
	_ ItemsGetter  = Map{}
)

// TypeName implements Object interface.
func (Map) TypeName() string {
	return "map"
}

// String implements Object interface.
func (o Map) String() string {
	var sb strings.Builder
	sb.WriteString("{")
	last := len(o) - 1
	i := 0

	for k := range o {
		sb.WriteString(strconv.Quote(k))
		sb.WriteString(": ")
		switch v := o[k].(type) {
		case String:
			sb.WriteString(strconv.Quote(v.String()))
		case Char:
			sb.WriteString(strconv.QuoteRune(rune(v)))
		case Bytes:
			sb.WriteString(fmt.Sprint([]byte(v)))
		default:
			sb.WriteString(v.String())
		}
		if i != last {
			sb.WriteString(", ")
		}
		i++
	}

	sb.WriteString("}")
	return sb.String()
}

// Copy implements Copier interface.
func (o Map) Copy() Object {
	cp := make(Map, len(o))
	for k, v := range o {
		cp[k] = v
	}
	return cp
}

// DeepCopy implements DeepCopier interface.
func (o Map) DeepCopy() Object {
	cp := make(Map, len(o))
	for k, v := range o {
		switch t := v.(type) {
		case DeepCopier:
			cp[k] = t.DeepCopy()
		case Copier:
			cp[k] = t.Copy()
		default:
			cp[k] = v
		}
	}
	return cp
}

// IndexSet implements Object interface.
func (o Map) IndexSet(index, value Object) error {
	o[index.String()] = value
	return nil
}

// IndexGet implements Object interface.
func (o Map) IndexGet(index Object) (Object, error) {
	v, ok := o[index.String()]
	if ok {
		return v, nil
	}
	return Nil, nil
}

// Equal implements Object interface.
func (o Map) Equal(right Object) bool {
	v, ok := right.(Map)
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
func (o Map) IsFalsy() bool { return len(o) == 0 }

// BinaryOp implements Object interface.
func (o Map) BinaryOp(tok token.Token, right Object) (Object, error) {
	if right == Nil {
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.TypeName(),
		right.TypeName())
}

// Iterate implements Iterable interface.
func (o Map) Iterate() Iterator {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	return &MapIterator{V: o, keys: keys}
}

// IndexDelete tries to delete the string value of key from the map.
// IndexDelete implements IndexDeleter interface.
func (o Map) IndexDelete(key Object) error {
	delete(o, key.String())
	return nil
}

// Len implements LengthGetter interface.
func (o Map) Len() int {
	return len(o)
}

func (o Map) Items() KeyValueArray {
	var (
		arr = make(KeyValueArray, len(o))
		i   int
	)
	for key, value := range o {
		arr[i] = KeyValue{String(key), value}
		i++
	}
	return arr
}

func (o Map) Keys() Array {
	var (
		arr = make(Array, len(o))
		i   int
	)
	for key := range o {
		arr[i] = String(key)
		i++
	}
	return arr
}

func (o Map) Values() Array {
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

// SyncMap represents map of objects and implements Object interface.
type SyncMap struct {
	mu    sync.RWMutex
	Value Map
}

var (
	_ Object       = (*SyncMap)(nil)
	_ Copier       = (*SyncMap)(nil)
	_ IndexDeleter = (*SyncMap)(nil)
	_ LengthGetter = (*SyncMap)(nil)
	_ KeysGetter   = (*SyncMap)(nil)
	_ ValuesGetter = (*SyncMap)(nil)
	_ ItemsGetter  = (*SyncMap)(nil)
)

// RLock locks the underlying mutex for reading.
func (o *SyncMap) RLock() {
	o.mu.RLock()
}

// RUnlock unlocks the underlying mutex for reading.
func (o *SyncMap) RUnlock() {
	o.mu.RUnlock()
}

// Lock locks the underlying mutex for writing.
func (o *SyncMap) Lock() {
	o.mu.Lock()
}

// Unlock unlocks the underlying mutex for writing.
func (o *SyncMap) Unlock() {
	o.mu.Unlock()
}

// TypeName implements Object interface.
func (*SyncMap) TypeName() string {
	return "syncMap"
}

// String implements Object interface.
func (o *SyncMap) String() string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.String()
}

// Copy implements Copier interface.
func (o *SyncMap) Copy() Object {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return &SyncMap{
		Value: o.Value.Copy().(Map),
	}
}

// DeepCopy implements DeepCopier interface.
func (o *SyncMap) DeepCopy() Object {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return &SyncMap{
		Value: o.Value.DeepCopy().(Map),
	}
}

// IndexSet implements Object interface.
func (o *SyncMap) IndexSet(index, value Object) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.Value == nil {
		o.Value = Map{}
	}
	return o.Value.IndexSet(index, value)
}

// IndexGet implements Object interface.
func (o *SyncMap) IndexGet(index Object) (Object, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.IndexGet(index)
}

// Equal implements Object interface.
func (o *SyncMap) Equal(right Object) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.Equal(right)
}

// IsFalsy implements Object interface.
func (o *SyncMap) IsFalsy() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.IsFalsy()
}

// Iterate implements Iterable interface.
func (o *SyncMap) Iterate() Iterator {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return &SyncIterator{Iterator: o.Value.Iterate()}
}

// Get returns Object in map if exists.
func (o *SyncMap) Get(index string) (value Object, exists bool) {
	o.mu.RLock()
	value, exists = o.Value[index]
	o.mu.RUnlock()
	return
}

// Len returns the number of items in the map.
// Len implements LengthGetter interface.
func (o *SyncMap) Len() int {
	o.mu.RLock()
	n := len(o.Value)
	o.mu.RUnlock()
	return n
}

// IndexDelete tries to delete the string value of key from the map.
func (o *SyncMap) IndexDelete(key Object) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.Value.IndexDelete(key)
}

// BinaryOp implements Object interface.
func (o *SyncMap) BinaryOp(tok token.Token, right Object) (Object, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return o.Value.BinaryOp(tok, right)
}

func (o *SyncMap) Items() KeyValueArray {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Value.Items()
}

func (o *SyncMap) Keys() Array {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.Value.Keys()
}

func (o *SyncMap) Values() Array {
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

// TypeName implements Object interface.
func (*Error) TypeName() string {
	return "error"
}

// String implements Object interface.
func (o *Error) String() string {
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
func (o *Error) IndexGet(index Object) (Object, error) {
	s := index.String()
	if s == "Literal" {
		return String(o.Name), nil
	}

	if s == "Message" {
		return String(o.Message), nil
	}

	if s == "New" {
		return &Function{
			Name: "New",
			Value: func(c Call) (Object, error) {
				l := c.Args.Len()
				switch l {
				case 1:
					return o.NewError(c.Args.Get(0).String()), nil
				case 0:
					return o.NewError(o.Message), nil
				default:
					msgs := make([]string, l)
					for i := range msgs {
						msgs[i] = c.Args.Get(i).String()
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
	Trace   []parser.Pos
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

func (o *RuntimeError) addTrace(pos parser.Pos) {
	if len(o.Trace) > 0 {
		if o.Trace[len(o.Trace)-1] == pos {
			return
		}
	}
	o.Trace = append(o.Trace, pos)
}

// TypeName implements Object interface.
func (*RuntimeError) TypeName() string {
	return "error"
}

// String implements Object interface.
func (o *RuntimeError) String() string {
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
		Trace:   append([]parser.Pos{}, o.Trace...),
	}
}

// Error implements error interface.
func (o *RuntimeError) Error() string {
	if o.Err == nil {
		return "<nil>"
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
func (o *RuntimeError) IndexGet(index Object) (Object, error) {
	if o.Err != nil {
		s := index.String()
		if s == "New" {
			return &Function{
				Name: "New",
				Value: func(c Call) (Object, error) {
					l := c.Args.Len()
					switch l {
					case 1:
						return o.NewError(c.Args.Get(0).String()), nil
					case 0:
						return o.NewError(o.Err.Message), nil
					default:
						msgs := make([]string, l)
						for i := range msgs {
							msgs[i] = c.Args.Get(i).String()
						}
						return o.NewError(msgs...), nil
					}
				},
			}, nil
		}
		return o.Err.IndexGet(index)
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
			_, _ = io.WriteString(s, o.String())
			if len(o.Trace) > 0 {
				if v := o.StackTrace(); v != nil {
					_, _ = io.WriteString(s, fmt.Sprintf("%+v", v))
				} else {
					_, _ = io.WriteString(s, "<nil stack trace>")
				}
			} else {
				_, _ = io.WriteString(s, "<no stack trace>")
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
			_, _ = io.WriteString(s, o.String())
		}
	case 'q':
		_, _ = io.WriteString(s, strconv.Quote(o.String()))
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
