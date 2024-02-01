package gad

// Invoker invokes a given callee object (either a CompiledFunction or any other
// callable object) with the given arguments.
//
// Invoker creates a new VM instance if the callee is a CompiledFunction,
// otherwise it runs the callee directly. Every Invoker call checks if the VM is
// aborted. If it is, it returns ErrVMAborted.
//
// Invoker is not safe for concurrent use by multiple goroutines.
//
// Acquire and Release methods are used to acquire and release a VM from the
// pool. So it is possible to reuse a VM instance for multiple CallWrapper calls.
// This is useful when you want to execute multiple functions in a single VM.
// For example, you can use Acquire and Release to execute multiple functions
// in a single VM instance.
// Note that you should call Release after Acquire, if you want to reuse the VM.
// If you don't want to use the pool, you can just call CallWrapper method.
// It is unsafe to hold a reference to the VM after Release is called.
// Using VM pool is about three times faster than creating a new VM for each
// CallWrapper call.
type Invoker struct {
	vm         *VM
	child      *VM
	callee     Object
	isCompiled bool
	dorelease  bool
	validArgs  bool
}

// NewInvoker creates a new Invoker object.
func NewInvoker(vm *VM, callee Object) *Invoker {
	inv := &Invoker{vm: vm, callee: callee}
	_, inv.isCompiled = inv.callee.(*CompiledFunction)
	return inv
}

// Acquire acquires a VM from the pool.
func (inv *Invoker) Acquire() {
	inv.acquire(true)
}

func (inv *Invoker) ValidArgs(v bool) *Invoker {
	inv.validArgs = v
	return inv
}

func (inv *Invoker) acquire(usePool bool) {
	if !inv.isCompiled {
		inv.child = inv.vm
	}
	if inv.child != nil {
		return
	}
	inv.child = inv.vm.pool.acquire(
		inv.callee.(*CompiledFunction),
		usePool,
	)
	if usePool {
		inv.dorelease = true
	}
}

// Release releases the VM back to the pool if it was acquired from the pool.
func (inv *Invoker) Release() {
	if inv.child != nil && inv.dorelease {
		inv.child.pool.release(inv.child)
	}
	inv.child = nil
	inv.dorelease = false
}

// Invoke invokes the callee object with the given arguments.
func (inv *Invoker) Invoke(args Args, namedArgs *NamedArgs) (Object, error) {
	if inv.child == nil {
		inv.acquire(false)
	}
	if inv.child.Aborted() {
		return Nil, ErrVMAborted
	}

	inv.child.StdIn = inv.vm.StdIn
	inv.child.StdOut = inv.vm.StdOut
	inv.child.StdErr = inv.vm.StdErr

	if inv.isCompiled {
		cf := inv.callee.(*CompiledFunction)
		if !inv.validArgs {
			if err := cf.ValidateParamTypes(inv.vm, args); err != nil {
				return nil, err
			}
		}
		return inv.child.RunOpts(&RunOpts{Globals: inv.vm.globals, Args: args, NamedArgs: namedArgs})
	}
	return inv.invokeObject(inv.callee, args)
}

func (inv *Invoker) invokeObject(co Object, args Args) (Object, error) {
	callee, _ := co.(CallerObject)
	if callee == nil {
		return Nil, ErrNotCallable.NewError(co.Type().Name())
	}
	return Val(callee.Call(Call{
		VM:   inv.vm,
		Args: args,
	}))
}

// Caller create new VM caller object.
func (inv *Invoker) Caller(args Args, namedArgs *NamedArgs) (VMCaller, error) {
	var validate = true
do:
	if inv.isCompiled {
		if inv.child == nil {
			inv.acquire(false)
		}

		if inv.child.Aborted() {
			return nil, ErrVMAborted
		}

		inv.child.StdIn = inv.vm.StdIn
		inv.child.StdOut = inv.vm.StdOut
		inv.child.StdErr = inv.vm.StdErr

		if validate && !inv.validArgs {
			if err := inv.callee.(*CompiledFunction).ValidateParamTypes(inv.vm, args); err != nil {
				return nil, err
			}
		}

		if err := inv.child.init(&RunOpts{Globals: inv.vm.globals, Args: args, NamedArgs: namedArgs}); err != nil {
			return nil, err
		}

		return &vmCompiledFuncCaller{
			callee:    inv.callee.(CallerObject),
			vm:        inv.child,
			args:      args,
			namedArgs: namedArgs,
		}, nil
	}

	callee, _ := inv.callee.(CallerObject)
	if callee == nil {
		return nil, ErrNotCallable.NewError(inv.callee.Type().Name())
	}

	if cwm, _ := callee.(*CallerObjectWithMethods); cwm != nil {
		callee, _ = cwm.CallerOf(args)
		if cf, _ := callee.(*CompiledFunction); cf != nil {
			inv.isCompiled = true
			inv.callee = cf
			validate = !inv.validArgs
			goto do
		}
	}

	caller := &vmObjectCaller{
		vm:     inv.vm,
		args:   args,
		callee: callee,
	}
	if namedArgs != nil {
		caller.namedArgs = *namedArgs
	}
	return caller, nil
}
