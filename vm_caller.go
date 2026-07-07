package gad

import "context"

type VMCaller interface {
	Call() (Object, error)
	Close()
	Callee() CallerObject
}

type vmCompiledFuncCaller struct {
	vm        *VM
	args      Args
	namedArgs *NamedArgs
	closed    bool
	callee    CallerObject
	ctx       context.Context
}

func (r *vmCompiledFuncCaller) Callee() CallerObject {
	return r.callee
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

	// When the Invoker was given a context, cancellation aborts the VM tree from
	// the root (same guard as Invoker.Invoke); otherwise this runs inline.
	return runWithContext(r.ctx, rootOf(r.vm), r.vm.run)
}

func (r *vmCompiledFuncCaller) Close() {
	if r.closed {
		return
	}
	r.vm.clearCurrentFrame()
	r.closed = true
}

type vmObjectCaller struct {
	vm        *VM
	args      Args
	namedArgs *NamedArgs
	closed    bool
	callee    CallerObject
}

func (r *vmObjectCaller) Callee() CallerObject {
	return r.callee
}

func (r *vmObjectCaller) Call() (ret Object, err error) {
	return DoCall(r.callee, Call{
		VM:        r.vm,
		Args:      r.args,
		NamedArgs: *r.namedArgs,
	})
}

func (r *vmObjectCaller) Close() {
	r.closed = true
}
