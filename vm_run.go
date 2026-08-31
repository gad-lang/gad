package gad

import (
	"context"
	"errors"
	"io"
)

type SetupOpts struct {
	ObjectConverters *ObjectConverters
	ToRawStrHandler  func(vm *VM, s Str) RawStr
	Context          context.Context
}

// RunFlags is a bitmask of per-run VM options (RunOpts.Flags). Zero is the
// default behaviour.
type RunFlags uint64

const (
	// RunFlagSkipReceiverTypeCheck disables the type check of the `this` receiver
	// on class-instance method and property calls. The receiver is typed (a class
	// method's `this` is its class; a mixin method's `this` is the mixin's
	// `@interface`) and normally verified — memoised on the root VM's interface-sat
	// cache, so the cost is a lookup after the first check. Set this flag to skip
	// even that on a hot path, when the receiver's shape is already trusted.
	RunFlagSkipReceiverTypeCheck RunFlags = 1 << iota
)

// Has reports whether every bit in f is set.
func (f RunFlags) Has(bit RunFlags) bool { return f&bit == bit }

type RunOpts struct {
	Globals        IndexGetSetter
	Args           Args
	NamedArgs      *NamedArgs
	StdIn          io.Reader
	StdOut         io.Writer
	StdErr         io.Writer
	ObjectToWriter ObjectToWriter
	// Env is the VM-scoped environment variable table reachable via the `env`
	// keyword. When nil an empty env is used.
	Env *Env
	// Flags are per-run VM options (see RunFlags).
	Flags RunFlags
}

// Run runs VM and executes the instructions until the OpReturn Opcode or Abort call.
func (vm *VM) Run(args ...Object) (Object, error) {
	return vm.RunOpts(&RunOpts{Args: Args{args}})
}

// RunOpts runs VM and executes the instructions until the OpReturn Opcode or Abort call.
func (vm *VM) RunOpts(opts *RunOpts) (Object, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	return vm.initAndRun(opts)
}

// RunCompiledFunction runs given CompiledFunction as if it is Main function.
// Bytecode must be set before calling this method, because Fileset and Constants are copied.
func (vm *VM) RunCompiledFunction(
	f *CompiledFunction,
	args ...Object,
) (Object, error) {
	return vm.RunCompiledFunctionOpts(f, &RunOpts{Args: Args{args}})
}

func (vm *VM) safeRun() (rerun bool) {
	defer func() {
		if vm.noPanic {
			if r := recover(); r != nil {
				vm.handlePanic(r)
				rerun = vm.err == nil
				return
			}
		}
		vm.clearCurrentFrame()
	}()
	if vm.dbg != nil {
		vm.loopDebug()
	} else {
		vm.loop()
	}
	return
}

func (vm *VM) run() (Object, error) {
	for run := true; run; {
		run = vm.safeRun()
	}
	if vm.err != nil {
		return nil, vm.err
	}

	if vm.sp < stackSize {
		if vv, ok := vm.stack[vm.sp-1].(*ObjectPtr); ok {
			return Val(*vv.Value, nil)
		}
		return Val(vm.stack[vm.sp-1], nil)
	}
	return nil, ErrStackOverflow
}

// RunCompiledFunctionOpts runs given CompiledFunction as if it is Main function.
// Bytecode must be set before calling this method, because Fileset and Constants are copied.
func (vm *VM) RunCompiledFunctionOpts(
	f *CompiledFunction,
	opts *RunOpts,
) (Object, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.bytecode == nil {
		return nil, errors.New("invalid Bytecode")
	}

	vm.bytecode = &Bytecode{
		FileSet:    vm.bytecode.FileSet,
		Constants:  vm.constants,
		Main:       f,
		NumModules: vm.bytecode.NumModules,
		Modules:    vm.bytecode.Modules,
	}

	for i := range vm.stack {
		vm.stack[i] = nil
	}
	return vm.initAndRun(opts)
}
