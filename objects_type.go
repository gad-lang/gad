package gad

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/token"
)

// Obj represents map of objects and implements Object interface.
type Obj struct {
	fields Dict
	typ    *ObjType
}

var (
	_ Object       = &Obj{}
	_ Copier       = &Obj{}
	_ LengthGetter = &Obj{}
	_ KeysGetter   = &Obj{}
	_ ValuesGetter = &Obj{}
	_ ItemsGetter  = &Obj{}
	_ IndexGetter  = &Obj{}
	_ IndexDeleter = &Obj{}
	_ IndexSetter  = &Obj{}
	_ ToWriter     = &Obj{}
)

func (o *Obj) Type() ObjectType {
	return o.typ
}

func (o *Obj) Fields() Dict {
	return o.fields
}

func (o *Obj) Stringer(c Call) (Str, error) {
	if o.typ.Stringer != nil {
		ret, err := o.typ.Stringer.Call(Call{VM: c.VM, Args: Args{Array{o}}})
		if err != nil {
			return "", err
		}
		s, _ := ToString(ret)
		return s, nil
	}
	return Str(o.ToString()), nil
}

func (o *Obj) ToString() string {
	var sb strings.Builder
	sb.WriteString(o.typ.Name())
	sb.WriteString("{")
	last := len(o.fields) - 1
	i := 0

	names := o.typ.FieldsDict.SortedKeys()

	for _, k := range names {
		ks := string(k.(Str))
		sb.WriteString(ks)
		sb.WriteString(": ")
		switch v := o.fields[ks].(type) {
		case Str:
			sb.WriteString(strconv.Quote(v.ToString()))
		case Char:
			sb.WriteString(strconv.QuoteRune(rune(v)))
		case Bytes:
			sb.WriteString(fmt.Sprint([]byte(v)))
		default:
			sb.WriteString(v.ToString())
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
func (o Obj) Copy() Object {
	o.fields = o.fields.Copy().(Dict)
	return &o
}

// DeepCopy implements DeepCopier interface.
func (o Obj) DeepCopy(vm *VM) (r Object, err error) {
	if r, err = o.fields.DeepCopy(vm); err != nil {
		return
	}
	o.fields = r.(Dict)
	return &o, nil
}

// IndexSet implements Object interface.
func (o *Obj) IndexSet(vm *VM, index, value Object) (err error) {
	name := index.ToString()
	if s := o.typ.SettersDict[name]; s != nil {
		_, err = s.(CallerObject).Call(Call{VM: vm, Args: Args{Array{o, value}}})
	} else {
		o.fields[name] = value
	}
	return
}

// IndexGet implements Object interface.
func (o *Obj) IndexGet(vm *VM, index Object) (Object, error) {
	name := index.ToString()
	if s := o.typ.GettersDict[name]; s != nil {
		return s.(CallerObject).Call(Call{VM: vm, Args: Args{Array{o}}})
	} else {
		v, ok := o.fields[name]
		if ok {
			return v, nil
		}
		return Nil, nil
	}
}

// Equal implements Object interface.
func (o *Obj) Equal(right Object) bool {
	v, ok := right.(*Obj)
	if !ok {
		return false
	}
	return o.typ.Equal(v.typ) && o.fields.Equal(v.fields)
}

// IsFalsy implements Object interface.
func (o *Obj) IsFalsy() bool { return len(o.fields) == 0 }

// BinaryOp implements Object interface.
func (o *Obj) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
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
		o.Type().Name(),
		right.Type().Name())
}

// Iterate implements Iterable interface.
func (o *Obj) Iterate(vm *VM) Iterator {
	return o.fields.Iterate(vm)
}

// IndexDelete tries to delete the string value of key from the map.
// IndexDelete implements IndexDeleter interface.
func (o *Obj) IndexDelete(_ *VM, key Object) error {
	delete(o.fields, key.ToString())
	return nil
}

// Len implements LengthGetter interface.
func (o *Obj) Len() int {
	return len(o.fields)
}

func (o *Obj) Items(vm *VM) (KeyValueArray, error) {
	return o.fields.Items(vm)
}

func (o *Obj) Keys() Array {
	return o.fields.Keys()
}

func (o *Obj) Values() Array {
	return o.fields.Values()
}

func (o *Obj) CallName(name string, c Call) (_ Object, err error) {
	if m := o.typ.MethodsDict[name]; m != nil {
		c.Args = append([]Array{{o}}, c.Args...)
		return m.(CallerObject).Call(c)
	}
	var v Object
	if v, err = o.IndexGet(c.VM, Str(name)); err != nil {
		return
	}
	if Callable(v) {
		return v.(CallerObject).Call(c)
	}
	return nil, ErrNotCallable.NewError("method " + strconv.Quote(name) + " of type " + v.Type().Name())
}

func (o *Obj) CastTo(vm *VM, t ObjectType) (Object, error) {
	return t.New(vm, o.fields)
}

func (o *Obj) CanWriteTo() bool {
	return o.typ.ToWriter != nil
}

func (o *Obj) WriteTo(vm *VM, w io.Writer) (int64, error) {
	ret, err := o.typ.ToWriter.Call(Call{VM: vm, Args: Args{Array{o, NewWriter(w)}}})
	i, _ := ToGoInt64(ret)
	return i, err
}

type ObjectTypeArray []ObjectType

func (o ObjectTypeArray) Type() ObjectType {
	return TObjectTypeArray
}

func (o ObjectTypeArray) ToString() string {
	return TObjectTypeArray.ToString() + ArrayToString(len(o), func(i int) Object {
		return o[i]
	})
}

func (o ObjectTypeArray) IsFalsy() bool {
	return len(o) == 0
}

func (o ObjectTypeArray) Equal(right Object) bool {
	if ta, ok := right.(ObjectTypeArray); ok {
		if len(ta) == len(o) {
			for i, ot := range o {
				if !ot.Equal(ta[i]) {
					return false
				}
			}
			return true
		}
	}
	return false
}

func (o ObjectTypeArray) Array() Array {
	arr := make(Array, len(o))
	for i, t := range o {
		arr[i] = t
	}
	return arr
}

// ObjType represents type objects and implements Object interface.
type ObjType struct {
	TypeName    string
	FieldsDict  Dict
	SettersDict Dict
	MethodsDict Dict
	GettersDict Dict
	Stringer    CallerObject
	Init        CallerObject
	ToWriter    CallerObject
	Inherits    ObjectTypeArray
}

func (o *ObjType) Fields() Dict {
	return o.FieldsDict
}

func (o *ObjType) Setters() Dict {
	return o.SettersDict
}

func (o *ObjType) Methods() Dict {
	return o.MethodsDict
}

func (o *ObjType) Getters() Dict {
	return o.GettersDict
}

func (o *ObjType) Name() string {
	return o.TypeName
}

func (o *ObjType) Type() ObjectType {
	return TNil
}

func (o *ObjType) IsChildOf(t ObjectType) bool {
	for _, p := range o.Inherits {
		if t == p || p.IsChildOf(t) {
			return true
		}
	}
	return false
}

func (o *ObjType) IndexGet(_ *VM, index Object) (value Object, err error) {
	switch index.ToString() {
	case "fields":
		return o.FieldsDict, nil
	case "getters":
		return o.GettersDict, nil
	case "setters":
		return o.SettersDict, nil
	case "methods":
		return o.MethodsDict, nil
	case "inherits":
		if o.Inherits == nil {
			return Array{}, nil
		}
		arr := make(Array, len(o.Inherits))
		for i, p := range o.Inherits {
			arr[i] = p
		}
		return arr, nil
	case "init":
		if o.Init == nil {
			return Nil, nil
		}
		return o.Init, nil
	case "name":
		return Str(o.TypeName), nil
	}
	return nil, ErrNotIndexable.NewError(index.ToString())
}

var (
	_ Object       = &ObjType{}
	_ CallerObject = &ObjType{}
)

func (o *ObjType) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v', 'S':
		if verb == 'v' && f.Flag('+') {
			f.Write([]byte(o.repr()))
			return
		}
	}
	f.Write([]byte(o.Name()))
}

func (o *ObjType) ToString() string {
	return o.Name()
}

func (o *ObjType) repr() string {
	m := Dict{}
	if len(o.FieldsDict) > 0 {
		m["fields"] = o.FieldsDict
	}
	if len(o.SettersDict) > 0 {
		m["setters"] = o.SettersDict
	}
	if len(o.GettersDict) > 0 {
		m["getters"] = o.GettersDict
	}
	if len(o.MethodsDict) > 0 {
		m["methods"] = o.MethodsDict
	}
	if len(o.Inherits) > 0 {
		m["inherits"] = o.Inherits
	}
	if o.Init != nil {
		m["init"] = o.Init
	}
	if o.Stringer != nil {
		m["toString"] = o.Stringer
	}
	return o.TypeName + m.ToString()
}

// Equal implements Object interface.
func (o *ObjType) Equal(right Object) bool {
	v, ok := right.(*ObjType)
	if !ok {
		return false
	}
	return v == o
}

func (ObjType) IsFalsy() bool { return false }

func (o *ObjType) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
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
		o.Type().Name(),
		right.Type().Name())
}

func (o *ObjType) New(_ *VM, fields Dict) (Object, error) {
	obj := &Obj{typ: o, fields: fields}
	if fields == nil {
		if o.FieldsDict == nil {
			obj.fields = Dict{}
		} else {
			obj.fields = o.FieldsDict.Copy().(Dict)
		}
	}
	return obj, nil
}

func (o *ObjType) Call(c Call) (obj Object, err error) {
	if o.Init != nil {
		obj, _ = o.New(c.VM, nil)
		if _, err = o.Init.Call(Call{
			VM:        c.VM,
			Args:      append(Args{Array{obj}}, c.Args...),
			NamedArgs: c.NamedArgs,
			SafeArgs:  c.SafeArgs,
		}); err != nil {
			return
		}
	} else if c.NamedArgs.IsFalsy() {
		obj, _ = o.New(c.VM, nil)
	} else {
		obj, _ = o.New(c.VM, c.NamedArgs.Dict())
	}
	return
}
