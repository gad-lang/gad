package gad

import (
	"reflect"
)

var TBase = &Type{TypeName: "Base"}

type Type struct {
	TypeName string
	Parent   ObjectType
}

func (t *Type) IsFalsy() bool {
	return t.TypeName == ""
}

func (t *Type) Type() ObjectType {
	return t.Parent
}

func (t *Type) ToString() string {
	return t.TypeName
}

func (t *Type) Equal(right Object) bool {
	rt, _ := right.(*Type)
	return rt == t
}

func (Type) Call(Call) (Object, error) {
	return nil, ErrNotCallable
}

func (t Type) Name() string {
	return t.TypeName
}

func (Type) Getters() Dict {
	return nil
}

func (Type) Setters() Dict {
	return nil
}

func (Type) Methods() Dict {
	return nil
}

func (Type) Fields() Dict {
	return nil
}

func (Type) New(*VM, Dict) (Object, error) {
	return nil, ErrNotInitializable
}

func (t *Type) IsChildOf(ot ObjectType) bool {
	return ot == t.Parent
}

var Types = map[reflect.Type]ObjectType{}

func RegisterBuiltinType(typ BuiltinType, name string, val any, init CallableFunc) *BuiltinObjType {
	ot := &BuiltinObjType{NameValue: name, Value: init}
	BuiltinObjects[typ] = ot
	BuiltinsMap[name] = typ

	rt := reflect.TypeOf(val)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	Types[rt] = ot
	return ot
}

func TypeOf(arg Object) ObjectType {
	ot := arg.Type()
	if ot == nil {
		return DetectTypeOf(arg)
	}
	return ot
}

func DetectTypeOf(arg Object) ObjectType {
	rt := reflect.TypeOf(arg)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	ot := Types[rt]
	if ot == nil {
		ot = Nil.Type()
	}
	return ot
}
