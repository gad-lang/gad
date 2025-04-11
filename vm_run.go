package gad

import (
	"context"
	"errors"
	"io"
)

type SetupOpts struct {
	ObjectConverters *ObjectConverters
	Builtins         *Builtins
	ToRawStrHandler  func(vm *VM, s Str) RawStr
	Context          context.Context
}

type RunOpts struct {
	Globals        IndexGetSetter
	Args           Args
	NamedArgs      *NamedArgs
	StdIn          io.Reader
	StdOut         io.Writer
	StdErr         io.Writer
	ObjectToWriter ObjectToWriter
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
	vm.loop()
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
			return *vv.Value, nil
		}
		return vm.stack[vm.sp-1], nil
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
		NumEmbeds:  vm.bytecode.NumEmbeds,
	}

	for i := range vm.stack {
		vm.stack[i] = nil
	}
	return vm.initAndRun(opts)
}
