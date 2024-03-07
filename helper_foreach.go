package gad

type PipedInvokeIterator struct {
	it                 Iterator
	args               Array
	startArgValueIndex int
	caller             VMCaller
	handler            func(state *IteratorState) error
	preCall            func(k, v Object) (Object, error)
	postCall           func(state *IteratorState, ret Object) error
	typ                ObjectType
}

func NewPipedInvokeIterator(it Iterator, args Array, startArgValueIndex int, caller VMCaller) (fe *PipedInvokeIterator) {
	fe = &PipedInvokeIterator{
		it:                 it,
		args:               args,
		startArgValueIndex: startArgValueIndex,
		caller:             caller,
		handler: func(state *IteratorState) error {
			return nil
		},
	}
	return
}

func (f *PipedInvokeIterator) Repr(vm *VM) (s string, err error) {
	if s, err = f.it.Repr(vm); err != nil {
		return
	}
	var s2 Str
	if s2, err = ToRepr(vm, f.caller.Callee()); err != nil {
		return
	}
	s += " → " + string(s2)
	return ToReprTypedRS(vm, f.Type(), ReprQuote(s))
}

func (f *PipedInvokeIterator) SetType(typ ObjectType) *PipedInvokeIterator {
	f.typ = typ
	return f
}

func (f *PipedInvokeIterator) Type() ObjectType {
	if f.typ != nil {
		return f.typ
	}
	return TPipedInvokeIterator
}

func (f *PipedInvokeIterator) Input() Object {
	return f.it.Input()
}

func (f *PipedInvokeIterator) PreCall() func(k, v Object) (Object, error) {
	return f.preCall
}

func (f *PipedInvokeIterator) SetPreCall(preCall func(k, v Object) (Object, error)) *PipedInvokeIterator {
	f.preCall = preCall
	return f
}

func (f *PipedInvokeIterator) PostCall() func(state *IteratorState, ret Object) error {
	return f.postCall
}

func (f *PipedInvokeIterator) SetPostCall(postCall func(state *IteratorState, ret Object) error) *PipedInvokeIterator {
	f.postCall = postCall
	return f
}

func (f *PipedInvokeIterator) Handler() func(state *IteratorState) error {
	return f.handler
}

func (f *PipedInvokeIterator) SetHandler(handler func(state *IteratorState) error) *PipedInvokeIterator {
	f.handler = handler
	return f
}

func (f *PipedInvokeIterator) checkNext(vm *VM, state *IteratorState) (err error) {
try:
	if err = IteratorStateCheck(vm, f.it, state); err != nil || state.Mode == IteratorStateModeDone {
		return
	}
	if err = f.handler(state); err == nil {
		if err = f.Call(state); state.Mode != IteratorStateModeEntry {
			goto try
		}
	}
	return
}

func (f *PipedInvokeIterator) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = f.it.Start(vm); err != nil {
		return
	}
	err = f.checkNext(vm, state)
	return
}

func (f *PipedInvokeIterator) Next(vm *VM, state *IteratorState) (err error) {
	if err = f.it.Next(vm, state); err != nil {
		return
	}
	return f.checkNext(vm, state)
}

func (f *PipedInvokeIterator) Call(state *IteratorState) (err error) {
	f.args[f.startArgValueIndex] = state.Entry.V
	f.args[f.startArgValueIndex+1] = state.Entry.K

	if f.preCall != nil {
		if state.Entry.V, err = f.preCall(state.Entry.K, state.Entry.V); err != nil {
			return
		}
	}

	var ret Object
	if ret, err = f.caller.Call(); err == nil {
		if f.postCall != nil {
			return f.postCall(state, ret)
		}

		if e2, _ := ret.(*KeyValue); e2 != nil {
			state.Entry = *e2
		} else {
			state.Entry.V = ret
		}
	}
	return
}
