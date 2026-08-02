package gad

// Call invokes co with the given args/namedArgs and returns its result.
//
// Call is the standard way to invoke a callable from Go while a VM is available;
// prefer it over co.Call(Call{VM: vm, ...}). When co resolves to a
// *CompiledFunction and this VM is currently running its instruction loop, Call
// runs the function as a synchronous sub-run on the SAME VM — it pushes a frame
// and drives a nested loop iteration until that frame returns, with no fork and
// no new VM. Any other callable (builtin, Go function, Prop, method wrapper …)
// runs through DoCall, which itself re-enters Call for the *CompiledFunction it
// eventually dispatches to.
//
// When the VM is not running (an external Go entry point that merely holds a
// *VM), Call falls back to the Invoker path, which forks a child VM for a
// *CompiledFunction — preserving the isolation embedding code relies on.
func (vm *VM) Call(co Object, args Args, namedArgs *NamedArgs) (Object, error) {
	if namedArgs == nil {
		namedArgs = NewNamedArgs()
	}
	if cf, ok := co.(*CompiledFunction); ok && vm.running {
		return vm.callCompiledInline(cf, args, namedArgs, true)
	}
	caller, ok := co.(CallerObject)
	if !ok {
		return Nil, ErrNotCallable.NewError(co.Type().Name())
	}
	if !vm.running {
		return NewInvoker(vm, co).Invoke(args, namedArgs)
	}
	return DoCall(caller, Call{VM: vm, Args: args, NamedArgs: *namedArgs})
}

// callCompiledInline runs cf on the same VM without forking: it pushes a boundary
// frame (reusing the normal call setup so argument binding, named params and
// param-type validation match a bytecode call) and drives a nested loop until
// that frame returns. The VM's loop state (ip / curFrame / curInsts /
// frameIndex / sp) is saved and restored around the sub-run.
func (vm *VM) callCompiledInline(cf *CompiledFunction, args Args, namedArgs *NamedArgs, validate bool) (_ Object, err error) {
	var (
		savedIP         = vm.ip
		savedFrame      = vm.curFrame
		savedInsts      = vm.curInsts
		savedFrameIndex = vm.frameIndex
		resultSlot      = vm.sp
	)

	// Layout expected by xOpCallCompiled: a callee slot (result lands here on
	// return), then the positional args, then — when present — the named-args
	// object flagged as the named-args-var.
	vm.stack[vm.sp] = cf
	vm.sp++
	posArgs := args.Values()
	for _, a := range posArgs {
		vm.stack[vm.sp] = a
		vm.sp++
	}
	var flags OpCallFlag
	if namedArgs != nil && !namedArgs.IsFalsy() {
		na := *namedArgs
		vm.stack[vm.sp] = &na
		vm.sp++
		flags |= OpCallFlagNamedArgsVar
	}

	restore := func() {
		for i := vm.sp - 1; i >= resultSlot; i-- {
			vm.stack[i] = nil
		}
		vm.sp = resultSlot
		vm.ip = savedIP
		vm.curFrame = savedFrame
		vm.curInsts = savedInsts
		vm.frameIndex = savedFrameIndex
	}

	vm.noValidateParams = !validate
	err = vm.xOpCallCompiled(cf, len(posArgs), flags)
	vm.noValidateParams = false
	if err != nil {
		restore()
		return nil, err
	}
	// xOpCallCompiled may perform a tail-call that reuses the current (outer)
	// frame instead of pushing a new one; that only happens for self-recursion,
	// which an inline call into a distinct function never triggers. The pushed
	// frame is the top one — mark it as the sub-run boundary.
	vm.curFrame.boundary = true

	// Drive the nested loop until the boundary frame returns.
	if vm.dbg != nil {
		vm.loopDebug()
	} else {
		vm.loop()
	}

	if vm.err != nil {
		err = vm.err
		vm.err = nil
		restore()
		return nil, err
	}
	if vm.Aborted() {
		restore()
		return Nil, ErrVMAborted
	}

	result := vm.stack[resultSlot]
	if ptr, ok := result.(*ObjectPtr); ok {
		result = *ptr.Value
	}
	restore()
	return result, nil
}
