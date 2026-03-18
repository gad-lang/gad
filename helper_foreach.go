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

func (i *PipedInvokeIterator) SetType(typ ObjectType) *PipedInvokeIterator {
	i.typ = typ
	return i
}

func (i *PipedInvokeIterator) Type() ObjectType {
	if i.typ != nil {
		return i.typ
	}
	return TPipedInvokeIterator
}

func (i *PipedInvokeIterator) Input() Object {
	return i.it.Input()
}

func (i *PipedInvokeIterator) PreCall() func(k, v Object) (Object, error) {
	return i.preCall
}

func (i *PipedInvokeIterator) SetPreCall(preCall func(k, v Object) (Object, error)) *PipedInvokeIterator {
	i.preCall = preCall
	return i
}

func (i *PipedInvokeIterator) PostCall() func(state *IteratorState, ret Object) error {
	return i.postCall
}

func (i *PipedInvokeIterator) SetPostCall(postCall func(state *IteratorState, ret Object) error) *PipedInvokeIterator {
	i.postCall = postCall
	return i
}

func (i *PipedInvokeIterator) Handler() func(state *IteratorState) error {
	return i.handler
}

func (i *PipedInvokeIterator) SetHandler(handler func(state *IteratorState) error) *PipedInvokeIterator {
	i.handler = handler
	return i
}

func (i *PipedInvokeIterator) checkNext(vm *VM, state *IteratorState) (err error) {
try:
	if err = IteratorStateCheck(vm, i.it, state); err != nil || state.Mode == IteratorStateModeDone {
		return
	}
	if err = i.handler(state); err == nil {
		if err = i.Call(state); state.Mode != IteratorStateModeEntry {
			goto try
		}
	}
	return
}

func (i *PipedInvokeIterator) Start(vm *VM) (state *IteratorState, err error) {
	if state, err = i.it.Start(vm); err != nil {
		return
	}
	err = i.checkNext(vm, state)
	return
}

func (i *PipedInvokeIterator) Next(vm *VM, state *IteratorState) (err error) {
	if err = i.it.Next(vm, state); err != nil {
		return
	}
	return i.checkNext(vm, state)
}

func (i *PipedInvokeIterator) Call(state *IteratorState) (err error) {
	i.args[i.startArgValueIndex] = state.Entry.V
	if len(i.args) > 1 {
		i.args[i.startArgValueIndex+1] = state.Entry.K
	}

	if i.preCall != nil {
		if state.Entry.V, err = i.preCall(state.Entry.K, state.Entry.V); err != nil {
			return
		}
	}

	var ret Object
	if ret, err = Val(i.caller.Call()); err == nil {
		if i.postCall != nil {
			return i.postCall(state, ret)
		}

		if e2, _ := ret.(*KeyValue); e2 != nil {
			state.Entry = *e2
		} else {
			state.Entry.V = ret
		}
	}
	return
}

func (i *PipedInvokeIterator) Print(state *PrinterState) error {
	defer state.WrapIndentedReprString(i.Type().FullName())()
	defer state.Enter()()
	if state.Indented() {
		state.PrintLineIndent()
	}
	state.WriteString(i.caller.Callee().ToString())
	state.WriteString(" → ")
	if state.Indented() {
		state.PrintLineIndent()
	}
	return i.it.Print(state)
}
