package gad

type VMCaller interface {
	Call() (Object, error)
	Close()
}

type vmCompiledFuncCaller struct {
	vm        *VM
	args      Args
	namedArgs *NamedArgs
	closed    bool
}

func (r *vmCompiledFuncCaller) Call() (ret Object, err error) {
	if r.closed {
		return nil, ErrVMAborted
	}

	r.vm.resetState(r.args, r.namedArgs)

	defer func() {
		if r.vm.Aborted() {
			r.Close()
		}
	}()

	return r.vm.run()
}

func (r *vmCompiledFuncCaller) Close() {
	if r.closed {
		return
	}
	r.vm.clearCurrentFrame()
	r.closed = true
	r.vm.mu.Unlock()
}

type vmObjectCaller struct {
	vm        *VM
	args      Args
	namedArgs NamedArgs
	closed    bool
	callee    CallerObject
}

func (r *vmObjectCaller) Call() (ret Object, err error) {
	return DoCall(r.callee, Call{
		VM:        r.vm,
		Args:      r.args,
		NamedArgs: r.namedArgs,
	})
}

func (r *vmObjectCaller) Close() {
	r.closed = true
}
