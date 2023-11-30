package gad

import (
	"reflect"
)

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
