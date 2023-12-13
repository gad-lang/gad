// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"context"
)

// Eval compiles and runs scripts within same scope.
// If executed script's return statement has no value to return or return is
// omitted, it returns last value on stack.
// Warning: Eval is not safe to use concurrently.
type Eval struct {
	Locals []Object
	*RunOpts
	Opts         CompilerOptions
	VM           *VM
	ModulesCache []Object
}

// NewEval returns new Eval object.
func NewEval(opts CompilerOptions, runOpts ...*RunOpts) *Eval {
	var runopts *RunOpts
	for _, runopts = range runOpts {
	}
	if runopts == nil {
		runopts = &RunOpts{}
	}
	if runopts.Globals == nil {
		runopts.Globals = Dict{}
	}
	if opts.SymbolTable == nil {
		opts.SymbolTable = NewSymbolTable(BuiltinsMap)
	}
	if opts.moduleStore == nil {
		opts.moduleStore = newModuleStore()
	}

	return &Eval{
		RunOpts: runopts,
		Opts:    opts,
		VM:      NewVM(nil).SetRecover(true),
	}
}

// Run compiles, runs given script and returns last value on stack.
func (r *Eval) Run(ctx context.Context, script []byte) (Object, *Bytecode, error) {
	bytecode, err := Compile(script, r.Opts)
	if err != nil {
		return nil, nil, err
	}

	r.Opts.Constants = bytecode.Constants
	r.fixOpPop(bytecode)
	r.VM.SetBytecode(bytecode)

	if ctx == nil {
		ctx = context.Background()
	}

	r.VM.modulesCache = r.ModulesCache
	ret, err := r.run(ctx)
	r.ModulesCache = r.VM.modulesCache
	r.Locals = r.VM.GetLocals(r.Locals)
	r.VM.Clear()

	if err != nil {
		return nil, bytecode, err
	}
	return ret, bytecode, nil
}

func (r *Eval) run(ctx context.Context) (ret Object, err error) {
	ret = Nil
	doneCh := make(chan struct{})
	// Always check whether context is done before running VM because
	// parser and compiler may take longer than expected or context may be
	// canceled for any reason before run, so use two selects.
	select {
	case <-ctx.Done():
		r.VM.Abort()
		err = ctx.Err()
	default:
		go func() {
			defer close(doneCh)
			copy(r.VM.stack[:], r.Locals)
			ret, err = r.VM.RunOpts(r.RunOpts)
		}()

		select {
		case <-ctx.Done():
			r.VM.Abort()
			<-doneCh
			if err == nil {
				err = ctx.Err()
			}
		case <-doneCh:
		}
	}
	return
}

// fixOpPop changes OpPop and OpReturn Opcodes to force VM to return last value on top of stack.
func (*Eval) fixOpPop(bytecode *Bytecode) {
	var prevOp byte
	var lastOp byte
	var fixPos int

	IterateInstructions(bytecode.Main.Instructions,
		func(pos int, opcode Opcode, operands []int, offset int) bool {
			if prevOp == 0 {
				prevOp = opcode
			} else {
				prevOp = lastOp
			}
			fixPos = -1
			lastOp = opcode
			if prevOp == OpPop && lastOp == OpReturn && operands[0] == 0 {
				fixPos = pos - 1
			}
			return true
		},
	)

	if fixPos > 0 {
		bytecode.Main.Instructions[fixPos] = OpNoOp // overwrite OpPop
		bytecode.Main.Instructions[fixPos+2] = 1    // set number of return to 1
	}
}
