package gad

import "errors"

type RunOpts struct {
	Globals   IndexGetSetter
	Args      Args
	NamedArgs *NamedArgs
}

// Run runs VM and executes the instructions until the OpReturn Opcode or Abort call.
func (vm *VM) Run(args ...Object) (Object, error) {
	return vm.RunOpts(&RunOpts{Args: Args{args}})
}

// RunOpts runs VM and executes the instructions until the OpReturn Opcode or Abort call.
func (vm *VM) RunOpts(opts *RunOpts) (Object, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	return vm.init(opts)
}

// RunCompiledFunction runs given CompiledFunction as if it is Main function.
// Bytecode must be set before calling this method, because Fileset and Constants are copied.
func (vm *VM) RunCompiledFunction(
	f *CompiledFunction,
	args ...Object,
) (Object, error) {
	return vm.RunCompiledFunctionOpts(f, &RunOpts{Args: Args{args}})
}

func (vm *VM) run() (rerun bool) {
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
	}

	for i := range vm.stack {
		vm.stack[i] = nil
	}
	return vm.init(opts)
}
