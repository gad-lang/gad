package gad

import (
	"fmt"
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
)

func (o *Obj) Type() ObjectType {
	return o.typ
}

func (o *Obj) Fields() Dict {
	return o.fields
}

func (o *Obj) Stringer(c Call) (String, error) {
	if o.typ.Stringer != nil {
		ret, err := o.typ.Stringer.Call(Call{VM: c.VM, Args: Args{Array{o}}})
		if err != nil {
			return "", err
		}
		s, _ := ToString(ret)
		return s, nil
	}
	return String(o.ToString()), nil
}

func (o *Obj) ToString() string {
	var sb strings.Builder
	sb.WriteString(o.typ.Name())
	sb.WriteString("{")
	last := len(o.fields) - 1
	i := 0

	names := o.typ.fields.SortedKeys()

	for _, k := range names {
		ks := string(k.(String))
		sb.WriteString(ks)
		sb.WriteString(": ")
		switch v := o.fields[ks].(type) {
		case String:
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
func (o Obj) DeepCopy() Object {
	o.fields = o.fields.DeepCopy().(Dict)
	return &o
}

// IndexSet implements Object interface.
func (o *Obj) IndexSet(vm *VM, index, value Object) (err error) {
	name := index.ToString()
	if s := o.typ.setters[name]; s != nil {
		_, err = s.(CallerObject).Call(Call{VM: vm, Args: Args{Array{o, value}}})
	} else {
		o.fields[name] = value
	}
	return
}

// IndexGet implements Object interface.
func (o *Obj) IndexGet(vm *VM, index Object) (Object, error) {
	name := index.ToString()
	if s := o.typ.getters[name]; s != nil {
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
func (o *Obj) BinaryOp(tok token.Token, right Object) (Object, error) {
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

func (o *Obj) Items() KeyValueArray {
	return o.fields.Items()
}

func (o *Obj) Keys() Array {
	return o.fields.Keys()
}

func (o *Obj) Values() Array {
	return o.fields.Values()
}

func (o *Obj) CallName(name string, c Call) (_ Object, err error) {
	if m := o.typ.methods[name]; m != nil {
		c.Args = append([]Array{{o}}, c.Args...)
		return m.(CallerObject).Call(c)
	}
	var v Object
	if v, err = o.IndexGet(c.VM, String(name)); err != nil {
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
	TypeName string
	fields   Dict
	setters  Dict
	methods  Dict
	getters  Dict
	Stringer CallerObject
	Init     CallerObject
	Inherits ObjectTypeArray
}

func (o *ObjType) Fields() Dict {
	return o.fields
}

func (o *ObjType) Setters() Dict {
	return o.setters
}

func (o *ObjType) Methods() Dict {
	return o.methods
}

func (o *ObjType) Getters() Dict {
	return o.getters
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
		return o.fields, nil
	case "getters":
		return o.getters, nil
	case "setters":
		return o.setters, nil
	case "methods":
		return o.methods, nil
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
		return String(o.TypeName), nil
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
	if len(o.fields) > 0 {
		m["fields"] = o.fields
	}
	if len(o.setters) > 0 {
		m["setters"] = o.setters
	}
	if len(o.getters) > 0 {
		m["getters"] = o.getters
	}
	if len(o.methods) > 0 {
		m["methods"] = o.methods
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

func (o *ObjType) BinaryOp(tok token.Token, right Object) (Object, error) {
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
		if o.fields == nil {
			obj.fields = Dict{}
		} else {
			obj.fields = o.fields.Copy().(Dict)
		}
	}
	return obj, nil
}

func (o *ObjType) Call(c Call) (obj Object, err error) {
	if o.Init != nil {
		obj, _ = o.New(c.VM, nil)
		if _, err = o.Init.Call(Call{c.VM, append(Args{Array{obj}}, c.Args...), c.NamedArgs}); err != nil {
			return
		}
	} else if c.NamedArgs.IsFalsy() {
		obj, _ = o.New(c.VM, nil)
	} else {
		obj, _ = o.New(c.VM, c.NamedArgs.Dict())
	}
	return
}
