// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type IteratorStateMode uint8

const (
	IteratorStateModeEntry IteratorStateMode = iota
	IteratorStateModeContinue
	IteratorStateModeDone
)

type IteratorStateCollectMode uint8

const (
	IteratorStateCollectModePair IteratorStateCollectMode = iota
	IteratorStateCollectModeKeys
	IteratorStateCollectModeValues
)

func (m IteratorStateCollectMode) String() string {
	switch m {
	case IteratorStateCollectModePair:
		return "pair"
	case IteratorStateCollectModeValues:
		return "values"
	case IteratorStateCollectModeKeys:
		return "keys"
	}
	return fmt.Sprint(uint8(m))
}

type IteratorState struct {
	Mode        IteratorStateMode
	CollectMode IteratorStateCollectMode
	Entry       KeyValue
	Value       Object
}

func (s IteratorState) Get() Object {
	switch s.CollectMode {
	case IteratorStateCollectModeKeys:
		return s.Entry.K
	case IteratorStateCollectModePair:
		kv := s.Entry
		return &kv
	default:
		return s.Entry.V
	}
}

// Iterator wraps the methods required to iterate Objects in VM.
type Iterator interface {
	Representer
	Type() ObjectType
	Input() Object
	Start(vm *VM) (state *IteratorState, err error)
	Next(vm *VM, state *IteratorState) (err error)
}

type ObjectIterator interface {
	Object
	Iterator
	GetIterator() Iterator
}

type ValuesIterator interface {
	Iterator
	Values() Array
}

type LengthIterator interface {
	Iterator
	Length() int
}

type CollectableIterator interface {
	Iterator
	Collect(vm *VM) (Object, error)
}

type StartIterationHandler func(vm *VM) (state *IteratorState, err error)

type NextIterationHandler func(vm *VM, state *IteratorState) (err error)

type Iteration struct {
	itType       ObjectType
	StartHandler StartIterationHandler
	NextHandler  NextIterationHandler
	input        Object
}

var (
	_ Iterator = (*Iteration)(nil)
)

func (it *Iteration) SetInput(input Object) *Iteration {
	it.input = input
	return it
}

func (it *Iteration) Input() Object {
	if it.input != nil {
		return it.input
	}
	return Nil
}

func (it *Iteration) ItType() ObjectType {
	return it.itType
}

func (it *Iteration) SetItType(itType ObjectType) *Iteration {
	it.itType = itType
	return it
}

func (it *Iteration) Repr(vm *VM) (string, error) {
	return ToReprTypedRS(vm, it.itType, it.Input().ToString())
}

func (it *Iteration) Type() ObjectType {
	return TIterator
}

func (it *Iteration) Start(vm *VM) (state *IteratorState, err error) {
	return it.StartHandler(vm)
}

func (it *Iteration) Next(vm *VM, state *IteratorState) (err error) {
	return it.NextHandler(vm, state)
}

func NewIterator(start StartIterationHandler, next NextIterationHandler) *Iteration {
	return &Iteration{StartHandler: start, NextHandler: next}
}

type LimitedIterator struct {
	Iterator
	Len int
}

var (
	_ Iterator       = (*LimitedIterator)(nil)
	_ LengthIterator = (*LimitedIterator)(nil)
)

func NewLimitedIteration(it Iterator, len int) *LimitedIterator {
	return &LimitedIterator{Iterator: it, Len: len}
}

func (it *LimitedIterator) Length() int {
	return it.Len
}

type RangeIteration struct {
	It         Object
	ItType     ObjectType
	valid      func(i int) bool
	step       int
	start, end int
	Len        int
	ReadTo     func(e *KeyValue, i int) error
}

var (
	_ Iterator       = (*RangeIteration)(nil)
	_ LengthIterator = (*LimitedIterator)(nil)
)

func NewRangeIteration(typ ObjectType, o Object, len int, readTo func(e *KeyValue, i int) error) *RangeIteration {
	var (
		valid = func(i int) bool {
			return i >= 0 && i+1 < len
		}
	)
	return &RangeIteration{ItType: typ, It: o, valid: valid, step: 1, end: len - 1, Len: len, ReadTo: readTo}
}

func (it *RangeIteration) Type() ObjectType {
	return it.ItType
}

func (it *RangeIteration) Repr(vm *VM) (string, error) {
	var opts []string
	if it.end < it.start {
		opts = append(opts, "reversed")
	}
	if it.step != 1 && it.step != -1 {
		opts = append(opts, "step="+strconv.Itoa(it.step))
	}
	var s string
	if opts != nil {
		s = ";"
		s += strings.Join(opts, ",")
	}
	return ToReprTypedRS(vm, it.ItType, it.It.ToString()+s)
}

func (it *RangeIteration) SetReversed(v bool) *RangeIteration {
	if v {
		it.start = it.Len - 1
		it.end = 0
		it.step = -(it.step)
		it.valid = func(i int) bool {
			return i <= it.start && i >= it.end
		}
	} else {
		it.end = it.Len - 1
		it.step = +(it.step)
		it.valid = func(i int) bool {
			return i >= 0 && i <= it.end
		}
	}
	return it
}

func (it *RangeIteration) ParseNamedArgs(na *NamedArgs) *RangeIteration {
	if v := na.GetValue("step"); v.Type() == TInt {
		it.step = int(v.(Int))
	}
	it.SetReversed(!na.GetValue("reversed").IsFalsy())
	return it
}

func (it *RangeIteration) Input() Object {
	return it.It
}

func (it *RangeIteration) Start(*VM) (state *IteratorState, err error) {
	state = &IteratorState{}
	if it.Len > 0 {
		state.Value = Int(it.start)
		err = it.ReadTo(&state.Entry, it.start)
		return
	}
	state.Mode = IteratorStateModeDone
	return
}

func (it *RangeIteration) Next(_ *VM, state *IteratorState) (err error) {
	state.Mode = IteratorStateModeEntry
	if i, ok := state.Value.(Int); ok {
		newI := int(i) + it.step
		if it.valid(newI) {
			state.Value = Int(newI)
			return it.ReadTo(&state.Entry, newI)
		}
	}
	state.Mode = IteratorStateModeDone
	return
}

func (it *RangeIteration) Length() int {
	return it.Len
}

func SliceIteration[T any](typ ObjectType, o Object, items []T, get func(e *KeyValue, i Int, v T) error) *RangeIteration {
	return NewRangeIteration(typ, o, len(items), func(e *KeyValue, i int) (err error) {
		return get(e, Int(i), items[i])
	})
}

func SliceEntryIteration[T any](typ ObjectType, o Object, items []T, get func(v T) (key Object, val Object, err error)) *RangeIteration {
	return NewRangeIteration(typ, o, len(items), func(e *KeyValue, i int) (err error) {
		e.K, e.V, err = get(items[i])
		return
	})
}

type zipIterator struct {
	Iterators []Iterator
	itsCount  int
}

func (it *zipIterator) Type() ObjectType {
	return TZipIterator
}

func ZipIterator(its ...Iterator) Iterator {
	return &zipIterator{Iterators: its, itsCount: len(its)}
}

func (it *zipIterator) Repr(vm *VM) (_ string, err error) {
	var s = make([]string, len(it.Iterators))
	for i := range s {
		if s[i], err = it.Iterators[i].Repr(vm); err != nil {
			return
		}
	}
	return ToReprTypedRS(vm, it.Type(), strings.Join(s, ", "))
}

func (it *zipIterator) Input() Object {
	its := make(Array, len(it.Iterators))
	for i, it := range it.Iterators {
		its[i] = IteratorObject(it)
	}
	return its
}

func (it *zipIterator) Start(vm *VM) (state *IteratorState, err error) {
	state = &IteratorState{}
	if it.itsCount == 0 {
		state.Mode = IteratorStateModeDone
	} else {
		err = it.StartFrom(vm, state, 0)
	}
	return
}

func (it *zipIterator) StartFrom(vm *VM, state *IteratorState, start int) (err error) {
	state.Mode = 0
	if start == it.itsCount {
		state.Mode = IteratorStateModeDone
	} else {
		for i, iterator := range it.Iterators[start:] {
			var state2 *IteratorState
			if state2, err = iterator.Start(vm); err != nil {
				return
			}
			if state2.Mode == IteratorStateModeDone {
				continue
			}

			state.Entry = state2.Entry
			state.Value = Array{Int(start + i), state2.Value}
			return
		}
		state.Mode = IteratorStateModeDone
	}
	return
}

func (it *zipIterator) Next(vm *VM, state *IteratorState) (err error) {
	state.Mode = IteratorStateModeEntry

	if stateArr, ok := state.Value.(Array); ok && len(stateArr) == 2 {
		if i, ok := stateArr[0].(Int); ok && i >= 0 && i < Int(it.itsCount) {
			state.Value = stateArr[1]
			if err = it.Iterators[i].Next(vm, state); err != nil {
				return
			} else if state.Mode == IteratorStateModeDone {
				err = it.StartFrom(vm, state, int(i)+1)
				return
			}
			stateArr[1] = state.Value
			state.Value = stateArr
			return
		}
	}
	state.Mode = IteratorStateModeDone
	return
}

var _ Object = (*iteratorObject)(nil)

// iteratorObject is used in VM to make an iterable Object.
type iteratorObject struct {
	typ ObjectType
	ObjectImpl
	Iterator
}

func IteratorObject(it Iterator) Object {
	return &iteratorObject{Iterator: it}
}

func TypedIteratorObject(typ ObjectType, it Iterator) Object {
	return &iteratorObject{typ: typ, Iterator: it}
}

func (o *iteratorObject) Type() ObjectType {
	if o.typ != nil {
		return o.typ
	}
	return o.Iterator.Type()
}

func (o *iteratorObject) Repr(vm *VM) (string, error) {
	if o.typ != nil {
		return ToReprTypedRS(vm, o.typ, o.Iterator)
	}
	return o.Iterator.Repr(vm)
}

func (o *iteratorObject) GetIterator() Iterator {
	return o.Iterator
}

func (o *iteratorObject) ToString() string {
	return "iteratorObject of " + o.Input().ToString()
}

type StateIteratorObject struct {
	Iterator
	State *IteratorState
	VM    *VM
}

func (s *StateIteratorObject) IndexGet(vm *VM, index Object) (value Object, err error) {
	switch index.ToString() {
	case "entry":
		if s.State == nil {
			return Nil, err
		}
		return &s.State.Entry, nil
	case "k":
		if s.State == nil {
			return Nil, err
		}
		return s.State.Entry.K, nil
	case "v":
		if s.State == nil {
			return Nil, err
		}
		return s.State.Entry.V, nil
	case "started":
		if s.State == nil {
			return False, err
		}
		return True, nil
	case "done":
		if s.State == nil {
			return False, err
		}
		if s.State.Mode == IteratorStateModeDone {
			return True, nil
		}
		return False, nil
	case "next":
		var hasNext bool
		if hasNext, err = s.Read(); err != nil {
			return
		}
		if hasNext {
			return s.State.Get(), nil
		}
		return Nil, err
	}
	return nil, ErrInvalidIndex
}

func (s *StateIteratorObject) IsFalsy() bool {
	if s.State == nil {
		return false
	}
	return s.State.Mode == IteratorStateModeDone
}

func (s *StateIteratorObject) ToString() string {
	return "StateIterator: " + s.Info().ToString()
}

func (s *StateIteratorObject) Info() Dict {
	status := "wait"
	if s.State != nil {
		if s.State.Mode == IteratorStateModeDone {
			status = "done"
		}
	}
	d := Dict{
		"Status": Str(status),
	}
	if s.State != nil {
		d["Value"] = s.State.Value
		d["Entry"] = &s.State.Entry
		d["CollectMode"] = Str(s.State.CollectMode.String())
	}
	return d
}

func (s *StateIteratorObject) Equal(right Object) bool {
	if o, _ := right.(*StateIteratorObject); o != nil {
		return o == s
	}
	return false
}

func (s *StateIteratorObject) Type() ObjectType {
	return TStateIterator
}

func (s *StateIteratorObject) Repr(vm *VM) (r string, err error) {
	if r, err = s.Iterator.Repr(vm); err != nil {
		return
	}
	return "StateIterator:" + s.Info().ToString() + " of " + r, nil
}

func (s *StateIteratorObject) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = s.Iterator.Start(vm); err != nil {
		return
	}
	s.State = state
	err = IteratorStateCheck(s.VM, s.Iterator, s.State)
	return
}

func (s *StateIteratorObject) Next(vm *VM, state *IteratorState) (err error) {
	s.State = state
	if err = s.Iterator.Next(vm, state); err == nil {
		err = IteratorStateCheck(s.VM, s.Iterator, s.State)
	}
	return
}

func NewStateIteratorObject(vm *VM, it Iterator) *StateIteratorObject {
	if si, _ := it.(*StateIteratorObject); si != nil {
		return si
	}
	return &StateIteratorObject{Iterator: it, VM: vm}
}

func (s *StateIteratorObject) GetIterator() Iterator {
	return s.Iterator
}

func (s *StateIteratorObject) Read() (_ bool, err error) {
	if s.State == nil {
		if s.State, err = s.Start(s.VM); err != nil {
			return
		}
	} else if err = s.Next(s.VM, s.State); err != nil {
		return
	}
	return s.State.Mode != IteratorStateModeDone, nil
}

func (s *StateIteratorObject) Key() Object {
	return s.State.Entry.K
}

func (s *StateIteratorObject) Value() Object {
	return s.State.Entry.V
}

var (
	_ Iterator = (*nilIteratorObject)(nil)
)

// nilIteratorObject is used in VM to make an non iterable Object.
type nilIteratorObject struct{}

func (*nilIteratorObject) Type() ObjectType {
	return TNilIterator
}

func (o *nilIteratorObject) Repr(vm *VM) (string, error) {
	return ReprQuote(o.Type().Name()), nil
}

func (*nilIteratorObject) Input() Object {
	return Nil
}

func (*nilIteratorObject) Start(*VM) (_ *IteratorState, _ error) {
	return &IteratorState{Mode: IteratorStateModeDone}, nil
}

func (o *nilIteratorObject) Next(_ *VM, state *IteratorState) error {
	state.Mode = IteratorStateModeDone
	return nil
}

var _ Object = (*iteratorObject)(nil)

func (o Str) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TStrIterator, o, []rune(o), func(e *KeyValue, i Int, v rune) error {
		e.K = i
		e.V = Char(v)
		return nil
	}).ParseNamedArgs(na)
}

func (o RawStr) Iterate(_ *VM, na *NamedArgs) Iterator {
	var r = []rune(o)
	return SliceIteration(TRawStrIterator, o, r, func(e *KeyValue, i Int, v rune) error {
		e.K = i
		e.V = Char(v)
		return nil
	}).ParseNamedArgs(na)
}

func (o Bytes) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TBytesIterator, o, o, func(e *KeyValue, i Int, v byte) error {
		e.K = i
		e.V = Int(v)
		return nil
	}).ParseNamedArgs(na)
}

func (o Array) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TArrayIterator, o, o, func(e *KeyValue, i Int, v Object) error {
		e.K = i
		e.V = v
		return nil
	}).ParseNamedArgs(na)
}

func (o KeyValueArray) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TKeyValueArrayIterator, o, o, func(e *KeyValue, i Int, v *KeyValue) error {
		*e = *v
		return nil
	}).ParseNamedArgs(na)
}

func (o KeyValueArrays) Iterate(_ *VM, na *NamedArgs) Iterator {
	return SliceIteration(TKeyValueArraysIterator, o, o, func(e *KeyValue, i Int, v KeyValueArray) error {
		e.K = i
		e.V = v
		return nil
	}).ParseNamedArgs(na)
}

func (o Dict) Iterate(_ *VM, na *NamedArgs) Iterator {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	if !na.GetValue("sorted").IsFalsy() || !na.MustGetValue("reversed").IsFalsy() {
		sort.Strings(keys)
	}
	return SliceEntryIteration(TDictIterator, o, keys, func(v string) (_, _ Object, _ error) {
		return Str(v), o[v], nil
	}).ParseNamedArgs(na)
}

func (o *SyncDict) Iterate(_ *VM, na *NamedArgs) Iterator {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.Value.Iterate(nil, na)
}

func (o *Buffer) Iterate(_ *VM, na *NamedArgs) Iterator {
	return Bytes(o.Bytes()).Iterate(nil, na)
}

// SyncIterator represents an iterator for the SyncDict.
type SyncIterator struct {
	mu sync.Mutex
	*Iteration
}

func (it *SyncIterator) StartIteration(vm *VM) (state *IteratorState, err error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	return it.Iteration.Start(vm)
}

func (it *SyncIterator) NextIteration(vm *VM, state *IteratorState) (err error) {
	it.mu.Lock()
	defer it.mu.Unlock()
	return it.Iteration.Next(vm, state)
}

func (o Args) Iterate(_ *VM, na *NamedArgs) Iterator {
	return NewRangeIteration(TArgsIterator, o, o.Length(), func(e *KeyValue, i int) error {
		e.K, e.V = Int(i), o.GetOnly(i)
		return nil
	}).ParseNamedArgs(na)
}

func (o *NamedArgs) Iterate(vm *VM, na *NamedArgs) Iterator {
	return o.Join().Iterate(vm, na)
}

func (o *ReflectArray) Iterate(vm *VM, na *NamedArgs) Iterator {
	return NewRangeIteration(TReflectArrayIterator, o, o.RValue.Len(), func(e *KeyValue, i int) (err error) {
		var v Object
		v, err = o.Get(vm, i)
		e.K = Int(i)
		e.V = v
		return
	}).ParseNamedArgs(na)
}

func (o *ReflectMap) Iterate(vm *VM, na *NamedArgs) Iterator {
	return SliceEntryIteration(TReflectMapIterator, o, o.RValue.MapKeys(), func(k reflect.Value) (key, val Object, err error) {
		rv := o.RValue.MapIndex(k)
		if rv.IsValid() {
			if key, err = vm.ToObject(k.Interface()); err == nil {
				val, err = vm.ToObject(rv.Interface())
			}
		}
		return
	}).ParseNamedArgs(na)
}

func (s *ReflectStruct) Iterate(vm *VM, na *NamedArgs) Iterator {
	return SliceEntryIteration(TReflectStructIterator, s, s.RType.FieldsNames, func(k string) (key, val Object, err error) {
		if val, err = s.IndexGetS(vm, k); err == nil {
			key = Str(k)
		}
		return
	}).ParseNamedArgs(na)
}

type wrapIterator struct {
	Iterator
	Wrap func(state *IteratorState) error
}

func WrapIterator(iterator Iterator, wrap func(state *IteratorState) error) *wrapIterator {
	return &wrapIterator{Iterator: iterator, Wrap: wrap}
}

func (f *wrapIterator) checkNext(vm *VM, state *IteratorState) (err error) {
try:
	if err = IteratorStateCheck(vm, f.Iterator, state); err != nil || state.Mode == IteratorStateModeDone {
		return
	}
	if err = f.Wrap(state); err == nil {
		if state.Mode != IteratorStateModeEntry {
			goto try
		}
	}
	return
}

func (f *wrapIterator) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = f.Iterator.Start(vm); err != nil {
		return
	}
	err = f.checkNext(vm, state)
	return
}

func (f *wrapIterator) Next(vm *VM, state *IteratorState) (err error) {
	if err = f.Iterator.Next(vm, state); err != nil {
		return
	}
	err = f.checkNext(vm, state)
	return
}

type collectModeIterator struct {
	Iterator
	mode IteratorStateCollectMode
}

func CollectModeIterator(iterator Iterator, mode IteratorStateCollectMode) Iterator {
	return &collectModeIterator{Iterator: iterator, mode: mode}
}

func (f *collectModeIterator) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = f.Iterator.Start(vm); err != nil {
		return
	}
	state.CollectMode = f.mode
	return
}

type itemsIterator struct {
	Iterator
}

func (it *itemsIterator) Type() ObjectType {
	return TItemsIterator
}

func (it *itemsIterator) Repr(vm *VM) (string, error) {
	return ToReprTypedRS(vm, it.Type(), it.Iterator)
}

func (it *itemsIterator) Collect(vm *VM) (_ Object, err error) {
	var ret KeyValueArray
	err = Iterate(vm, it.Iterator, nil, func(e *KeyValue) error {
		// copy key value
		kv := *e
		ret = append(ret, &kv)
		return nil
	})
	return ret, err
}

func IteratorStateCheck(vm *VM, it Iterator, state *IteratorState) (err error) {
	if state.Mode == IteratorStateModeDone {
		return
	}
	for state.Mode == IteratorStateModeContinue {
		if err = it.Next(vm, state); err != nil || state.Mode == IteratorStateModeDone {
			return
		}
	}
	return
}
