// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"reflect"
	"sort"
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
	IteratorStateCollectModeValues IteratorStateCollectMode = iota
	IteratorStateCollectModeKeys
	IteratorStateCollectModePair
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

type Iterators []Iterator

func (its Iterators) Print(s *PrinterState) (err error) {
	if err = s.WriteByte('['); err != nil {
		return
	}

	l := len(its)

	if l == 0 {
		return s.WriteByte(']')
	}

	defer func() {
		if err == nil {
			if s.Indented() && !s.SkipDepth() {
				s.PrintLine()
				if l > 0 {
					s.PrintIndent()
				}
			}
			err = s.WriteByte(']')
		}
	}()

	if s.SkipDepth() {
		_, err = s.Write([]byte("…"))
		return
	}

	defer s.Enter()()

	var (
		indexes, _ = s.options.Indexes()
		item       = func(i int) {
			err = its[i].Print(s)
		}
	)

	if indexes {
		item = func(i int) {
			_, _ = fmt.Fprintf(s, "%d 🠆 ", i)
			err = its[i].Print(s)
		}
	}

	var itemSep = []byte{','}

	if s.Indented() {
		itemSep = append(itemSep, '\n')
		s.PrintLine()

		old := item
		item = func(i int) {
			s.PrintIndent()
			old(i)
		}
	} else {
		itemSep = append(itemSep, ' ')
	}

	for i, last := 0, l-1; i <= last; i++ {
		if i > 0 {
			if _, err = s.Write(itemSep); err != nil {
				return
			}
		}
		item(i)
		if err != nil {
			return
		}
	}

	return
}

// Iterator wraps the methods required to iterate Objects in VM.
type Iterator interface {
	Printabler
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

var (
	_ Iterator = (*Iteration)(nil)
)

type Iteration struct {
	itType       ObjectType
	StartHandler StartIterationHandler
	NextHandler  NextIterationHandler
	input        Object
}

func NewIterator(start StartIterationHandler, next NextIterationHandler) *Iteration {
	return &Iteration{StartHandler: start, NextHandler: next}
}

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

func (it *Iteration) Type() ObjectType {
	return TIterator
}

func (it *Iteration) Start(vm *VM) (state *IteratorState, err error) {
	return it.StartHandler(vm)
}

func (it *Iteration) Next(vm *VM, state *IteratorState) (err error) {
	return it.NextHandler(vm, state)
}

func (it *Iteration) Print(state *PrinterState) error {
	if it.itType != nil {
		defer state.WrapReprString(it.itType.String())()
	}
	return state.Print(it.input)
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

func (it *RangeIteration) Print(state *PrinterState) error {
	defer state.WrapReprString(it.ItType.FullName())()
	return state.Print(it.It)
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

var (
	_ Iterator = (*zipIterator)(nil)
)

type zipIterator struct {
	Iterators Iterators
	itsCount  int
}

func (it *zipIterator) Type() ObjectType {
	return TZipIterator
}

func ZipIterator(its ...Iterator) Iterator {
	return &zipIterator{Iterators: its, itsCount: len(its)}
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

func (it *zipIterator) Print(state *PrinterState) error {
	defer state.WrapReprString(it.Type().String())()
	if !state.IsRepr || state.SkipNexDepth() {
		fmt.Fprintf(state, "%d of %d iterators", it.itsCount, len(it.Iterators))
		return nil
	}
	return it.Iterators.Print(state)
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
	if stateIt, _ := it.(*StateIteratorObject); stateIt != nil {
		return stateIt
	}
	return &iteratorObject{typ: typ, Iterator: it}
}

func (o *iteratorObject) Type() ObjectType {
	if o.typ != nil {
		return o.typ
	}
	return o.Iterator.Type()
}

func (o *iteratorObject) GetIterator() Iterator {
	return o.Iterator
}

func (o *iteratorObject) ToString() string {
	return "iteratorObject of " + o.Input().ToString()
}

func (o *iteratorObject) Print(state *PrinterState) error {
	if o.typ != nil {
		defer state.WrapReprString(o.typ.FullName())()
	}
	return o.Iterator.Print(state)
}

type StateIteratorObject struct {
	Iterator
	State         *IteratorState
	VM            *VM
	StartHandlers []func(s *StateIteratorObject)
	// pooled marks an internal for-in iterator (created by vm.acquireIter at
	// OpIterInit) that may be returned to vm.iterPool when the loop finishes. It
	// is never set on SIOs handed to user code (e.g. by the iterator() builtin),
	// so those are never pooled or reused.
	pooled bool
}

// acquireIter wraps it in a StateIteratorObject for a for-in loop, reusing one
// from the per-VM free list when possible. If it is already a StateIteratorObject
// (e.g. from the iterator() builtin) it is used as-is and NOT pooled, so a
// user-held iterator is never recycled underneath the user.
func (vm *VM) acquireIter(it Iterator) *StateIteratorObject {
	if si, _ := it.(*StateIteratorObject); si != nil {
		return si
	}
	var s *StateIteratorObject
	if n := len(vm.iterPool); n > 0 {
		s, vm.iterPool[n-1] = vm.iterPool[n-1], nil
		vm.iterPool = vm.iterPool[:n-1]
	} else {
		s = &StateIteratorObject{}
	}
	s.Iterator, s.State, s.VM = it, nil, vm
	s.StartHandlers = s.StartHandlers[:0]
	s.pooled = true
	return s
}

// releaseIter returns a pooled for-in iterator to the free list once its loop is
// done. It is a no-op for non-pooled (user-visible) iterators. The SIO is only
// released after its last use — OpIterNext has already replaced it on the stack
// with the loop condition and the internal `:it` local is dead — so nothing
// dereferences it after release.
func (vm *VM) releaseIter(s *StateIteratorObject) {
	if !s.pooled {
		return
	}
	s.pooled = false
	s.Iterator, s.State, s.VM = nil, nil, nil
	vm.iterPool = append(vm.iterPool, s)
}

func (s *StateIteratorObject) AddStartHandler(f func(s *StateIteratorObject)) {
	s.StartHandlers = append(s.StartHandlers, f)
	if s.State != nil {
		f(s)
	}
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

func (s *StateIteratorObject) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = s.Iterator.Start(vm); err != nil {
		return
	}
	s.State = state
	err = IteratorStateCheck(s.VM, s.Iterator, s.State)
	if err == nil && s.State.Mode != IteratorStateModeDone {
		for _, handler := range s.StartHandlers {
			handler(s)
		}
	}
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

func (o *nilIteratorObject) Print(state *PrinterState) error {
	state.WrapReprString(o.Type().String())()
	return nil
}

func (*nilIteratorObject) Type() ObjectType {
	return TNilIterator
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

// arrayIterator is a closure-free iterator over an Array. It replaces the
// generic RangeIteration/SliceIteration machinery for the common `for k, v in
// arr` case: one struct allocation instead of a RangeIteration plus the several
// closures those helpers capture (valid/readTo/get), which dominated the
// per-loop allocations. Semantics match SliceIteration(...).ParseNamedArgs: it
// honours the `step` and `reversed` named arguments, and the valid index range
// is simply 0 <= i < len in either direction.
type arrayIterator struct {
	arr   Array
	step  int
	start int
	state IteratorState // embedded so Start does not allocate a separate state
}

var (
	_ Iterator       = (*arrayIterator)(nil)
	_ LengthIterator = (*arrayIterator)(nil)
)

// iterStepStart derives the (step, start) of a forward/reverse index walk over n
// elements from the `step` and `reversed` named arguments, shared by the
// concrete slice iterators. The valid index range is 0 <= i < n in either
// direction.
func iterStepStart(n int, na *NamedArgs) (step, start int) {
	step = 1
	if na != nil {
		if v := na.GetValue("step"); v.Type() == TInt {
			step = int(v.(Int))
		}
		if !na.GetValue("reversed").IsFalsy() {
			start = n - 1
			step = -step
		}
	}
	return step, start
}

func newArrayIterator(arr Array, na *NamedArgs) *arrayIterator {
	step, start := iterStepStart(len(arr), na)
	return &arrayIterator{arr: arr, step: step, start: start}
}

func (it *arrayIterator) Type() ObjectType { return TArrayIterator }

func (it *arrayIterator) Input() Object { return it.arr }

func (it *arrayIterator) Length() int { return len(it.arr) }

func (it *arrayIterator) Start(*VM) (state *IteratorState, err error) {
	it.state = IteratorState{}
	state = &it.state
	if len(it.arr) == 0 {
		state.Mode = IteratorStateModeDone
		return
	}
	i := it.start
	state.Value = Int(i)
	state.Entry.K = Int(i)
	state.Entry.V = it.arr[i]
	return
}

func (it *arrayIterator) Next(_ *VM, state *IteratorState) (err error) {
	state.Mode = IteratorStateModeEntry
	if cur, ok := state.Value.(Int); ok {
		i := int(cur) + it.step
		if i >= 0 && i < len(it.arr) {
			state.Value = Int(i)
			state.Entry.K = Int(i)
			state.Entry.V = it.arr[i]
			return
		}
	}
	state.Mode = IteratorStateModeDone
	return
}

func (it *arrayIterator) Print(state *PrinterState) error {
	defer state.WrapReprString(TArrayIterator.FullName())()
	return state.Print(it.arr)
}

func (o Array) Iterate(_ *VM, na *NamedArgs) Iterator {
	return newArrayIterator(o, na)
}

// dictIterator is a closure-free iterator over a Dict's key/value pairs. The
// keys slice (built and, when requested, sorted by Dict.Iterate) fixes the
// order; like arrayIterator it avoids the RangeIteration plus the three closures
// SliceEntryIteration would allocate per loop. The keys slice itself remains
// (a stable order needs it), as does per-key Str boxing.
type dictIterator struct {
	dict  Dict
	keys  []string
	step  int
	start int
	state IteratorState // embedded so Start does not allocate a separate state
}

var (
	_ Iterator       = (*dictIterator)(nil)
	_ LengthIterator = (*dictIterator)(nil)
)

func newDictIterator(dict Dict, keys []string, na *NamedArgs) *dictIterator {
	step, start := iterStepStart(len(keys), na)
	return &dictIterator{dict: dict, keys: keys, step: step, start: start}
}

func (it *dictIterator) Type() ObjectType { return TDictIterator }

func (it *dictIterator) Input() Object { return it.dict }

func (it *dictIterator) Length() int { return len(it.keys) }

// at fills state with the i-th key/value pair.
func (it *dictIterator) at(i int, state *IteratorState) {
	k := it.keys[i]
	state.Value = Int(i)
	state.Entry.K = Str(k)
	state.Entry.V = it.dict[k]
}

func (it *dictIterator) Start(*VM) (state *IteratorState, err error) {
	it.state = IteratorState{}
	state = &it.state
	if len(it.keys) == 0 {
		state.Mode = IteratorStateModeDone
		return
	}
	it.at(it.start, state)
	return
}

func (it *dictIterator) Next(_ *VM, state *IteratorState) (err error) {
	state.Mode = IteratorStateModeEntry
	if cur, ok := state.Value.(Int); ok {
		i := int(cur) + it.step
		if i >= 0 && i < len(it.keys) {
			it.at(i, state)
			return
		}
	}
	state.Mode = IteratorStateModeDone
	return
}

func (it *dictIterator) Print(state *PrinterState) error {
	defer state.WrapReprString(TDictIterator.FullName())()
	return state.Print(it.dict)
}

// kvArrayIterator is a closure-free iterator over a KeyValueArray (its entries
// are the stored key/value pairs). Like arrayIterator it avoids the
// RangeIteration + closures SliceIteration would allocate per loop.
type kvArrayIterator struct {
	arr   KeyValueArray
	step  int
	start int
	state IteratorState
}

var _ Iterator = (*kvArrayIterator)(nil)

func (it *kvArrayIterator) Type() ObjectType { return TKeyValueArrayIterator }
func (it *kvArrayIterator) Input() Object    { return it.arr }
func (it *kvArrayIterator) Length() int      { return len(it.arr) }

func (it *kvArrayIterator) Start(*VM) (*IteratorState, error) {
	it.state = IteratorState{}
	if len(it.arr) == 0 {
		it.state.Mode = IteratorStateModeDone
		return &it.state, nil
	}
	it.state.Value = Int(it.start)
	it.state.Entry = *it.arr[it.start]
	return &it.state, nil
}

func (it *kvArrayIterator) Next(_ *VM, state *IteratorState) error {
	state.Mode = IteratorStateModeEntry
	if cur, ok := state.Value.(Int); ok {
		i := int(cur) + it.step
		if i >= 0 && i < len(it.arr) {
			state.Value = Int(i)
			state.Entry = *it.arr[i]
			return nil
		}
	}
	state.Mode = IteratorStateModeDone
	return nil
}

func (it *kvArrayIterator) Print(state *PrinterState) error {
	defer state.WrapReprString(TKeyValueArrayIterator.FullName())()
	return state.Print(it.arr)
}

func (o KeyValueArray) Iterate(_ *VM, na *NamedArgs) Iterator {
	step, start := iterStepStart(len(o), na)
	return &kvArrayIterator{arr: o, step: step, start: start}
}

// kvArraysIterator is a closure-free iterator over a KeyValueArrays: entries are
// (index, element).
type kvArraysIterator struct {
	arr   KeyValueArrays
	step  int
	start int
	state IteratorState
}

var _ Iterator = (*kvArraysIterator)(nil)

func (it *kvArraysIterator) Type() ObjectType { return TKeyValueArraysIterator }
func (it *kvArraysIterator) Input() Object    { return it.arr }
func (it *kvArraysIterator) Length() int      { return len(it.arr) }

func (it *kvArraysIterator) Start(*VM) (*IteratorState, error) {
	it.state = IteratorState{}
	if len(it.arr) == 0 {
		it.state.Mode = IteratorStateModeDone
		return &it.state, nil
	}
	it.state.Value = Int(it.start)
	it.state.Entry.K, it.state.Entry.V = Int(it.start), it.arr[it.start]
	return &it.state, nil
}

func (it *kvArraysIterator) Next(_ *VM, state *IteratorState) error {
	state.Mode = IteratorStateModeEntry
	if cur, ok := state.Value.(Int); ok {
		i := int(cur) + it.step
		if i >= 0 && i < len(it.arr) {
			state.Value = Int(i)
			state.Entry.K, state.Entry.V = Int(i), it.arr[i]
			return nil
		}
	}
	state.Mode = IteratorStateModeDone
	return nil
}

func (it *kvArraysIterator) Print(state *PrinterState) error {
	defer state.WrapReprString(TKeyValueArraysIterator.FullName())()
	return state.Print(it.arr)
}

func (o KeyValueArrays) Iterate(_ *VM, na *NamedArgs) Iterator {
	step, start := iterStepStart(len(o), na)
	return &kvArraysIterator{arr: o, step: step, start: start}
}

func (o Dict) Iterate(_ *VM, na *NamedArgs) Iterator {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	if !na.GetValue("sorted").IsFalsy() || !na.MustGetValue("reversed").IsFalsy() {
		sort.Strings(keys)
	}
	return newDictIterator(o, keys, na)
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

// argsIterator is a closure-free iterator over Args: entries are (index,
// positional value).
type argsIterator struct {
	args  Args
	n     int
	step  int
	start int
	state IteratorState
}

var _ Iterator = (*argsIterator)(nil)

func (it *argsIterator) Type() ObjectType { return TArgsIterator }
func (it *argsIterator) Input() Object    { return it.args }
func (it *argsIterator) Length() int      { return it.n }

func (it *argsIterator) Start(*VM) (*IteratorState, error) {
	it.state = IteratorState{}
	if it.n == 0 {
		it.state.Mode = IteratorStateModeDone
		return &it.state, nil
	}
	it.state.Value = Int(it.start)
	it.state.Entry.K, it.state.Entry.V = Int(it.start), it.args.GetOnly(it.start)
	return &it.state, nil
}

func (it *argsIterator) Next(_ *VM, state *IteratorState) error {
	state.Mode = IteratorStateModeEntry
	if cur, ok := state.Value.(Int); ok {
		i := int(cur) + it.step
		if i >= 0 && i < it.n {
			state.Value = Int(i)
			state.Entry.K, state.Entry.V = Int(i), it.args.GetOnly(i)
			return nil
		}
	}
	state.Mode = IteratorStateModeDone
	return nil
}

func (it *argsIterator) Print(state *PrinterState) error {
	defer state.WrapReprString(TArgsIterator.FullName())()
	return state.Print(it.args)
}

func (o Args) Iterate(_ *VM, na *NamedArgs) Iterator {
	n := o.Length()
	step, start := iterStepStart(n, na)
	return &argsIterator{args: o, n: n, step: step, start: start}
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
	if stateIt, _ := iterator.(*StateIteratorObject); stateIt != nil {
		stateIt.AddStartHandler(func(s *StateIteratorObject) {
			s.State.CollectMode = mode
		})
		return iterator
	}
	return &collectModeIterator{Iterator: iterator, mode: mode}
}

func (f *collectModeIterator) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = f.Iterator.Start(vm); err != nil {
		return
	}
	state.CollectMode = f.mode
	return
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
