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

func (o *Obj) ToString() string {
	return o.typ.Name() + o.fields.ToString()
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
		_, err = DoCall(s.(CallerObject), Call{VM: vm, Args: Args{Array{o, value}}})
	} else {
		o.fields[name] = value
	}
	return
}

// IndexGet implements Object interface.
func (o *Obj) IndexGet(vm *VM, index Object) (Object, error) {
	name := index.ToString()
	if s := o.typ.GettersDict[name]; s != nil {
		return YieldCall(s.(CallerObject), &Call{VM: vm, Args: Args{Array{o}}}), nil
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
		return YieldCall(m.(CallerObject), &c), nil
	}
	var v Object
	if v, err = o.IndexGet(c.VM, Str(name)); err != nil {
		return
	}
	if Callable(v) {
		return YieldCall(v.(CallerObject), &c), nil
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
	TypeName       string
	FieldsDict     Dict
	SettersDict    Dict
	MethodsDict    Dict
	GettersDict    Dict
	Inherits       ObjectTypeArray
	calllerMethods MethodArgType
	new            Function
}

func NewObjType(typeName string) *ObjType {
	ot := &ObjType{TypeName: typeName}
	ot.new.Name = typeName + "#new"
	ot.new.Value = ot.NewCall
	ot.calllerMethods.Add(nil, &CallerMethod{
		Default:      true,
		CallerObject: &ot.new,
	}, false)
	return ot
}

func (o *ObjType) AddCallerMethod(vm *VM, types MultipleObjectTypes, handler CallerObject, override bool) error {
	if len(types) == 0 {
		// overrides default constructor. uses Type.new to instantiate.
		override = true
	}
	return o.calllerMethods.Add(types, &CallerMethod{
		CallerObject: handler,
	}, override)
}

func (o *ObjType) HasCallerMethods() bool {
	return !o.calllerMethods.IsZero()
}

func (o *ObjType) CallerMethods() *MethodArgType {
	return &o.calllerMethods
}

func (o *ObjType) CallerOf(args Args) (co CallerObject, ok bool) {
	var types []ObjectType
	args.Walk(func(i int, arg Object) any {
		if t, ok := arg.(ObjectType); ok {
			types = append(types, t)
		} else {
			types = append(types, arg.Type())
		}
		return nil
	})
	return o.CallerOfTypes(types)
}

func (o *ObjType) CallerOfTypes(types []ObjectType) (co CallerObject, validate bool) {
	if method := o.calllerMethods.GetMethod(types); method != nil {
		return method.CallerObject, false
	}
	return o, validate
}

func (o *ObjType) Caller() CallerObject {
	return &o.new
}

func (o *ObjType) New(_ *VM, fields Dict) (Object, error) {
	var obj = &Obj{typ: o, fields: fields}
	if fields == nil {
		if o.FieldsDict == nil {
			obj.fields = Dict{}
		} else {
			obj.fields = o.FieldsDict.Copy().(Dict)
		}
	}
	return obj, nil
}

func (o *ObjType) NewCall(c Call) (Object, error) {
	return &Obj{typ: o, fields: c.NamedArgs.Dict()}, nil
}

func (o *ObjType) Call(c Call) (_ Object, err error) {
	caller, validate := o.CallerOf(c.Args)
	c.SafeArgs = !validate
	return YieldCall(caller, &c), nil
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
	return TBase
}

func (o *ObjType) IsChildOf(t ObjectType) bool {
	for _, p := range o.Inherits {
		if t == p || p.IsChildOf(t) {
			return true
		}
	}
	return false
}

func (o *ObjType) CallName(name string, c Call) (ret Object, err error) {
	if name == "new" {
		return o.NewCall(c)
	}
	return nil, ErrInvalidIndex.NewError(name)
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
	if o.calllerMethods.IsZero() {
		return o.Name()
	}

	var (
		s strings.Builder
		i int
	)

	s.WriteString(o.Name())

	o.calllerMethods.WalkSorted(func(m *CallerMethod) any {
		if !m.Default {
			s.WriteString(fmt.Sprintf("  %d. ", i+1))
			s.WriteString(m.CallerObject.ToString())
			s.WriteByte('\n')
			i++
		}
		return nil
	})

	return strings.TrimRight(fmt.Sprintf(" with %d methods:\n", i)+s.String(), "\n")
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
