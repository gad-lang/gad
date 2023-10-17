package gad

import "reflect"

// ReflectArrayIterator represents an iterator for the ReflectArray.
type ReflectArrayIterator struct {
	v    reflect.Value
	i, l int
}

func (it *ReflectArrayIterator) Next() bool {
	it.i++
	return it.i-1 < it.l
}

func (it *ReflectArrayIterator) Key() Object {
	return Int(it.i - 1)
}

func (it *ReflectArrayIterator) Value() (Object, error) {
	i := it.i - 1
	if i > -1 && i < it.l {
		return ToObject(it.v.Index(i).Interface())
	}
	return Nil, nil
}

// ReflectMapIterator represents an iterator for the ReflectMap.
type ReflectMapIterator struct {
	v    reflect.Value
	keys []reflect.Value
	i    int
}

var _ Iterator = (*ReflectMapIterator)(nil)

func (it *ReflectMapIterator) Next() bool {
	it.i++
	return it.i-1 < len(it.keys)
}

func (it *ReflectMapIterator) Key() Object {
	key, _ := ToObject(it.keys[it.i-1].Interface())
	return key
}

func (it *ReflectMapIterator) Value() (Object, error) {
	v := it.v.MapIndex(it.keys[it.i-1])
	if !v.IsValid() {
		return Nil, nil
	}
	return ToObject(v.Interface())
}

// ReflectStructIterator represents an iterator for the ReflectStruct.
type ReflectStructIterator struct {
	vm *VM
	v  *ReflectStruct
	i  int
}

var _ Iterator = (*ReflectStructIterator)(nil)

func (it *ReflectStructIterator) Next() bool {
	it.i++
	return it.i-1 < len(it.v.typ.fieldsNames)
}

func (it *ReflectStructIterator) Key() Object {
	return String(it.v.typ.fieldsNames[it.i-1])
}

func (it *ReflectStructIterator) Value() (Object, error) {
	return it.v.IndexGet(nil, it.Key())
}
