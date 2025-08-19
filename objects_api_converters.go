package gad

type ToArrayConveter interface {
	ToArray() Array
}

type ToDictConveter interface {
	ToDict(out Dict) Dict
}

// ConvertToArray convert objects to Array. If success return then, otherwise return error.
func ConvertToArray(vm *VM, o ...Object) (ret Array, err error) {
	for i, o := range o {
		switch obj := o.(type) {
		case Array:
			if i == 0 {
				ret = obj
			}
		case ToArrayAppenderObject:
			ret = obj.AppendToArray(ret)
		default:
			if err = ItemsOfCb(vm, nil, func(kv *KeyValue) error {
				ret = append(ret, kv.V)
				return nil
			}, o); err != nil {
				return
			}
		}
	}
	return
}

// MustConvertToArray convert objects to Array and return then if success, otherwise panics.
func MustConvertToArray(vm *VM, o ...Object) Array {
	d, err := ConvertToArray(vm, o...)
	if err != nil {
		panic(err)
	}
	return d
}

// ConvertToDict convert objects to Dict. If success return then, otherwise return error.
func ConvertToDict(vm *VM, o ...Object) (ret Dict, err error) {
	var retObj Object
	if retObj, err = NewDictFunc(Call{VM: vm, Args: Args{o}}); err == nil {
		ret = retObj.(Dict)
	}
	return
}

// MustConvertToDict convert objects to Dict and return then if success, otherwise panics.
func MustConvertToDict(vm *VM, o ...Object) Dict {
	d, err := ConvertToDict(vm, o...)
	if err != nil {
		panic(err)
	}
	return d
}

// ConvertToKeyValueArray convert objects to KeyValueArray. If success return then, otherwise return error.
func ConvertToKeyValueArray(vm *VM, o ...Object) (ret KeyValueArray, err error) {
	var retObj Object
	retObj, err = NewKeyValueArrayFunc(Call{VM: vm, Args: Args{o}})
	if err == nil {
		ret = retObj.(KeyValueArray)
	}
	return
}

// MustConvertToKeyValueArray convert objects to KeyValueArray and return then if success, otherwise panics.
func MustConvertToKeyValueArray(vm *VM, o ...Object) (ret KeyValueArray) {
	var err error
	if ret, err = ConvertToKeyValueArray(vm, o...); err != nil {
		panic(err)
	}
	return
}
