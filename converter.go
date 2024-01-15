package gad

import "reflect"

type ObjectConverters struct {
	ToGoHandlers     map[ObjectType]func(vm *VM, v Object) any
	ToObjectHandlers map[reflect.Type]func(vm *VM, v any) (Object, error)
}

func NewObjectConverters() *ObjectConverters {
	return &ObjectConverters{
		ToGoHandlers:     make(map[ObjectType]func(vm *VM, v Object) any),
		ToObjectHandlers: make(map[reflect.Type]func(vm *VM, v any) (Object, error)),
	}
}

func (oc *ObjectConverters) Register(objType ObjectType, togo func(vm *VM, v Object) any, goType reflect.Type, toObject func(vm *VM, v any) (Object, error)) *ObjectConverters {
	if objType != nil {
		oc.ToGoHandlers[objType] = togo
	}
	if goType != nil {
		oc.ToObjectHandlers[goType] = toObject
	}
	return oc
}

func (oc *ObjectConverters) ToObject(vm *VM, v any) (Object, error) {
	typ := reflect.TypeOf(v)
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if h := oc.ToObjectHandlers[typ]; h != nil {
		return h(vm, v)
	}

	return ToObject(v)
}

func (oc *ObjectConverters) ToInterface(vm *VM, v Object) any {
	if h := oc.ToGoHandlers[v.Type()]; h != nil {
		return h(vm, v)
	}

	if otoi, _ := v.(ToIterfaceVMConverter); otoi != nil {
		return otoi.ToInterface(vm)
	}

	return ToInterface(v)
}

func (vm *VM) ToObject(v any) (Object, error) {
	if v == nil {
		return Nil, nil
	}
	return vm.ObjectConverters.ToObject(vm, v)
}

func (vm *VM) ToInterface(v Object) any {
	if v == Nil {
		return nil
	}
	return vm.ObjectConverters.ToInterface(vm, v)
}

func (vm *VM) ToInterfaceArray(v Array) (ret []any) {
	ret = make([]any, len(v))
	for i, o := range v {
		ret[i] = vm.ToInterface(o)
	}
	return
}

func (vm *VM) ToObjectArray(v []any) (ret Array, err error) {
	ret = make(Array, len(v))
	for i, o := range v {
		if ret[i], err = vm.ToObject(o); err != nil {
			return
		}
	}
	return
}
