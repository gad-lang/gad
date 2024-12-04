// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

const (
	stackSize = 2048
	frameSize = 1024
)

// VM executes the instructions in Bytecode.
type VM struct {
	abort        int64
	sp           int
	ip           int
	curInsts     []byte
	constants    []Object
	stack        [stackSize]Object
	frames       [frameSize]frame
	curFrame     *frame
	frameIndex   int
	bytecode     *Bytecode
	modulesCache []Object
	globals      IndexGetSetter
	pool         vmPool
	mu           sync.Mutex
	err          error
	noPanic      bool

	StdOut, StdErr *StackWriter
	StdIn          *StackReader
	ObjectToWriter ObjectToWriter

	*SetupOpts
}

// NewVM creates a VM object.
func NewVM(bc *Bytecode) *VM {
	var constants []Object
	if bc != nil {
		constants = bc.Constants
	}
	vm := &VM{
		bytecode:  bc,
		constants: constants,
	}
	vm.pool.root = vm
	return vm
}

// SetRecover recovers panic when Run panics and returns panic as an error.
// If error handler is present `try-catch-finally`, VM continues to run from catch/finally.
func (vm *VM) SetRecover(v bool) *VM {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.noPanic = v
	return vm
}

// SetBytecode enables to set a new Bytecode.
func (vm *VM) SetBytecode(bc *Bytecode) *VM {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.bytecode = bc
	vm.constants = bc.Constants
	vm.modulesCache = nil
	return vm
}

// Clear clears stack by setting nil to stack indexes and removes modules cache.
func (vm *VM) Clear() *VM {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	for i := range vm.stack {
		vm.stack[i] = nil
	}
	vm.pool.clear()
	vm.modulesCache = nil
	vm.globals = nil
	return vm
}

// GetGlobals returns global variables.
func (vm *VM) GetGlobals() Object {
	return vm.globals
}

// GetLocals returns variables from stack up to the NumLocals of given Bytecode.
// This must be called after Run() before Clear().
func (vm *VM) GetLocals(locals []Object) []Object {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if locals != nil {
		locals = locals[:0]
	} else {
		locals = make([]Object, 0, vm.bytecode.Main.NumLocals)
	}

	for i := range vm.stack[:vm.bytecode.Main.NumLocals] {
		locals = append(locals, vm.stack[i])
	}

	return locals
}

// Abort aborts the VM execution. It is safe to call this method from another
// goroutine.
func (vm *VM) Abort() {
	vm.pool.abort(vm)
	atomic.StoreInt64(&vm.abort, 1)
}

// Aborted reports whether VM is aborted. It is safe to call this method from
// another goroutine.
func (vm *VM) Aborted() bool {
	return atomic.LoadInt64(&vm.abort) == 1
}

func (vm *VM) init(opts *RunOpts) error {
	if vm.bytecode == nil || vm.bytecode.Main == nil {
		return errors.New("invalid Bytecode")
	}

	vm.Setup(SetupOpts{})

	if opts.StdIn != nil {
		if s, _ := opts.StdIn.(*StackReader); s != nil {
			vm.StdIn = s
		} else {
			vm.StdIn = NewStackReader(opts.StdIn)
		}
	}
	if opts.StdOut != nil {
		if s, _ := opts.StdOut.(*StackWriter); s != nil {
			vm.StdOut = s
		} else {
			vm.StdOut = NewStackWriter(opts.StdOut)
		}
	}
	if opts.StdErr != nil {
		if s, _ := opts.StdErr.(*StackWriter); s != nil {
			vm.StdErr = s
		} else {
			vm.StdErr = NewStackWriter(opts.StdErr)
		}
	}

	if opts.ObjectToWriter != nil {
		vm.ObjectToWriter = opts.ObjectToWriter
	}

	// Resize modules cache or create it if not exists.
	// Note that REPL can set module cache before running, don't recreate it add missing indexes.
	if diff := vm.bytecode.NumModules - len(vm.modulesCache); diff > 0 {
		for i := 0; i < diff; i++ {
			vm.modulesCache = append(vm.modulesCache, nil)
		}
	}

	vm.initGlobals(opts.Globals)
	vm.resetState(opts.Args, opts.NamedArgs)

	return nil
}

func (vm *VM) resetState(args Args, namedArgs *NamedArgs) {
	vm.err = nil
	atomic.StoreInt64(&vm.abort, 0)
	vm.initCurrentFrame(args, namedArgs)
	vm.frameIndex = 1
}

func (vm *VM) Init() *VM {
	return vm.Setup(SetupOpts{})
}

func (vm *VM) Setup(opts SetupOpts) *VM {
	if vm.SetupOpts != nil {
		return vm
	}

	if vm.StdIn == nil {
		vm.StdIn, vm.StdOut, vm.StdErr = NewStackReader(os.Stdin), NewStackWriter(os.Stdout), NewStackWriter(os.Stderr)
	}

	vm.SetupOpts = &opts

	if vm.Builtins == nil {
		vm.Builtins = NewBuiltins()
	}

	if opts.Context == nil {
		opts.Context = context.Background()
	}

	vm.Builtins.Objects = vm.Builtins.Objects.Build()

	if vm.ObjectConverters == nil {
		vm.ObjectConverters = NewObjectConverters()
	}

	if vm.ObjectToWriter == nil {
		vm.ObjectToWriter = DefaultObjectToWrite
	}

	if vm.ToRawStrHandler == nil {
		vm.ToRawStrHandler = vm.toRawStr
	}

	return vm
}

func (vm *VM) toRawStr(_ *VM, s Str) RawStr {
	return RawStr(s)
}

func (vm *VM) initAndRun(opts *RunOpts) (Object, error) {
	if vm.bytecode == nil || vm.bytecode.Main == nil {
		return nil, errors.New("invalid Bytecode")
	}

	vm.Setup(SetupOpts{})

	vm.err = nil
	atomic.StoreInt64(&vm.abort, 0)
	vm.initGlobals(opts.Globals)
	vm.initCurrentFrame(opts.Args, opts.NamedArgs)
	vm.frameIndex = 1

	if opts.StdIn != nil {
		if s, _ := opts.StdIn.(*StackReader); s != nil {
			vm.StdIn = s
		} else {
			vm.StdIn = NewStackReader(opts.StdIn)
		}
	}
	if opts.StdOut != nil {
		if s, _ := opts.StdOut.(*StackWriter); s != nil {
			vm.StdOut = s
		} else {
			vm.StdOut = NewStackWriter(opts.StdOut)
		}
	}
	if opts.StdErr != nil {
		if s, _ := opts.StdErr.(*StackWriter); s != nil {
			vm.StdErr = s
		} else {
			vm.StdErr = NewStackWriter(opts.StdErr)
		}
	}

	if opts.ObjectToWriter != nil {
		vm.ObjectToWriter = opts.ObjectToWriter
	}

	// Resize modules cache or create it if not exists.
	// Note that REPL can set module cache before running, don't recreate it add missing indexes.
	if diff := vm.bytecode.NumModules - len(vm.modulesCache); diff > 0 {
		for i := 0; i < diff; i++ {
			vm.modulesCache = append(vm.modulesCache, nil)
		}
	}

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

func (vm *VM) initGlobals(globals IndexGetSetter) {
	if globals == nil {
		globals = Dict{}
	}
	vm.globals = globals
}

func (vm *VM) initLocals(args Array, namedArgs *NamedArgs) {
	vm.sp = 0
	// init all params as nil
	main := vm.bytecode.Main
	numParams := len(main.Params)
	numNamedParams := main.NamedParams.len
	locals := vm.stack[:main.NumLocals]

	vm.initLocalsOfFunc(main, args)

	if numNamedParams > 0 {
		if namedArgs == nil {
			namedArgs = &NamedArgs{}
		}
		if main.NamedParams.variadic {
			locals[numParams+numNamedParams-1] = namedArgs
			for i, p := range main.NamedParams.Params[:numNamedParams-1] {
				if v := namedArgs.GetValueOrNil(p.Name); v != nil {
					locals[numParams+i] = v
				}
			}
		} else {
			for i, p := range main.NamedParams.Params {
				if v := namedArgs.GetValueOrNil(p.Name); v != nil {
					locals[numParams+i] = v
				}
			}
		}
	}
}

func (vm *VM) initLocalsOfFunc(main *CompiledFunction, args Array) {
	numParams := len(main.Params)
	numNamedParams := main.NamedParams.len
	locals := vm.stack[vm.sp : vm.sp+main.NumLocals]

	// fix num reveived args < num expected args
	if diff := numParams - len(args); diff > 0 {
		if main.Params.Var() {
			diff--
		}
		for ; diff > 0; diff-- {
			args = append(args, Nil)
		}
	}

	for i := 0; i < main.NumLocals; i++ {
		if locals[i] == nil {
			locals[i] = Nil
		}
	}

	if numParams <= 0 && numNamedParams <= 0 {
		return
	}

	if len(args) < numParams {
		if main.Params.Var() {
			locals[numParams-1] = Array{}
		}
		copy(locals, args)
		return
	}

	if main.Params.Var() {
		locals[numParams-1] = args[numParams-1:]
	} else if numParams > 0 {
		locals[numParams-1] = args[numParams-1]
	}

	if numParams > 0 {
		copy(locals, args[:numParams-1])
	}

	vm.sp += main.NumLocals
}

func (vm *VM) initCurrentFrame(args Args, named *NamedArgs) {
	vm.initLocals(args.Values(), named)

	// initialize frame and pointers
	vm.curInsts = vm.bytecode.Main.Instructions
	vm.curFrame = &(vm.frames[0])
	vm.curFrame.fn = vm.bytecode.Main
	vm.curFrame.args = args
	vm.curFrame.namedArgs = named

	if vm.curFrame.fn.Free != nil {
		// Assign free variables if exists in compiled function.
		// This is required to run compiled functions returned from VM using RunCompiledFunction().
		vm.curFrame.freeVars = vm.curFrame.fn.Free
	}

	vm.curFrame.errHandlers = nil
	vm.curFrame.basePointer = 0
	vm.sp = vm.curFrame.fn.NumLocals
	vm.ip = -1
}

func (vm *VM) clearCurrentFrame() {
	for _, f := range vm.curFrame.defers {
		f()
	}
	vm.curFrame.defers = nil
	vm.curFrame.freeVars = nil
	vm.curFrame.fn = nil
	vm.curFrame.errHandlers = nil
	vm.curFrame.args = nil
	vm.curFrame.namedArgs = nil
}

func (vm *VM) handlePanic(r any) {
	if vm.sp < stackSize && vm.frameIndex <= frameSize && vm.err == nil {
		if err := vm.throwGenErr(fmt.Errorf("%v", r)); err != nil {
			vm.err = err
			gostack := debugStack()
			if vm.err != nil {
				vm.err = fmt.Errorf("panic: %v %w\nGo Stack:\n%s",
					r, vm.err, gostack)
			}
		}
		return
	}

	gostack := debugStack()

	if vm.err != nil {
		vm.err = fmt.Errorf("panic: %v error: %w\nGo Stack:\n%s",
			r, vm.err, gostack)
		return
	}
	vm.err = fmt.Errorf("panic: %v\nGo Stack:\n%s", r, gostack)
}

func (vm *VM) GetSymbolValue(symbol *SymbolInfo) (value Object, err error) {
	switch symbol.Scope {
	case ScopeGlobal:
		index := vm.constants[symbol.Index]
		value, err = Val(vm.globals.IndexGet(vm, index))
	case ScopeLocal:
		value = vm.stack[vm.curFrame.basePointer+symbol.Index]
		if v, ok := value.(*ObjectPtr); ok {
			value = *v.Value
		}
	case ScopeBuiltin:
		value = vm.Builtins.Objects[BuiltinType(symbol.Index)]
	case ScopeFree:
		value = *vm.curFrame.freeVars[symbol.Index].Value
	}
	return
}

func (vm *VM) xIndexGet(numSel int, target Object) (value Object, null, abort bool) {
	value = Nil

	for ; numSel > 0; numSel-- {
		ptr := vm.sp - numSel
		index := vm.stack[ptr]
		vm.stack[ptr] = nil
		if ig, _ := target.(IndexGetter); ig != nil {
			v, err := Val(ig.IndexGet(vm, index))
			if err != nil {
				switch err {
				case ErrNotIndexable:
					err = ErrNotIndexable.NewError(target.Type().Name())
				case ErrIndexOutOfBounds:
					err = ErrIndexOutOfBounds.NewError(index.ToString())
				}
				if err = vm.throwGenErr(err); err != nil {
					vm.err = err
					abort = true
					return
				}
				null = true
				return
			}
			target = v
			value = v
		} else {
			if err := vm.throwGenErr(ErrNotIndexable.NewError(target.Type().Name())); err != nil {
				vm.err = err
				abort = true
				return
			}
			null = true
			return
		}
	}

	return
}

func (vm *VM) xOpSetupTry() {
	catch := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
	finally := int(vm.curInsts[vm.ip+4]) | int(vm.curInsts[vm.ip+3])<<8

	ptrs := errHandler{
		sp:      vm.sp,
		catch:   catch,
		finally: finally,
	}

	if vm.curFrame.errHandlers == nil {
		vm.curFrame.errHandlers = &errHandlers{
			handlers: []errHandler{ptrs},
		}
	} else {
		vm.curFrame.errHandlers.handlers = append(
			vm.curFrame.errHandlers.handlers, ptrs)
	}

	vm.ip += 4
}

func (vm *VM) xOpSetupCatch() {
	value := Nil
	errHandlers := vm.curFrame.errHandlers

	if errHandlers.hasHandler() {
		// set 0 to last catch position
		hdl := errHandlers.last()
		hdl.catch = 0

		if errHandlers.err != nil {
			value = errHandlers.err
			errHandlers.err = nil
		}
	}

	vm.stack[vm.sp] = value
	vm.sp++
	// Either OpSetLocal or OpPop is generated by compiler to handle error
}

func (vm *VM) xOpSetupFinally() {
	errHandlers := vm.curFrame.errHandlers

	if errHandlers.hasHandler() {
		hdl := errHandlers.last()
		hdl.catch = 0
		hdl.finally = 0
	}
}

func (vm *VM) xOpThrow() error {
	op := vm.curInsts[vm.ip+1]
	vm.ip++

	switch op {
	case 0: // system
		errHandlers := vm.curFrame.errHandlers
		if errHandlers.hasError() {
			errHandlers.pop()
			// do not put position info to error for re-throw after finally.
			if err := vm.throw(errHandlers.err, true); err != nil {
				return err
			}
		} else if pos := errHandlers.hasReturnTo(); pos > 0 {
			// go to OpReturn if it is set
			handler := errHandlers.last()
			errHandlers.pop()
			handler.returnTo = 0

			if vm.sp >= handler.sp {
				for i := vm.sp; i >= handler.sp; i-- {
					vm.stack[i] = nil
				}
			}
			vm.sp = handler.sp
			vm.ip = pos - 1
		}
	case 1: // user
		obj := vm.stack[vm.sp-1]
		vm.stack[vm.sp-1] = nil
		vm.sp--

		if err := vm.throw(vm.newErrorFromObject(obj), false); err != nil {
			return err
		}
	default:
		return fmt.Errorf("wrong operand for OpThrow:%d", op)
	}

	return nil
}

func (vm *VM) SourcePos() source.SourceFilePos {
	p := vm.getSourcePos()
	return vm.bytecode.FileSet.File(p).Position(p)
}

func (vm *VM) throwGenErr(err error) error {
	if e, ok := err.(*RuntimeError); ok {
		if e.fileSet == nil {
			e.fileSet = vm.bytecode.FileSet
		}
		return vm.throw(e, true)
	} else if e, ok := err.(*Error); ok {
		return vm.throw(vm.newError(e), false)
	}

	return vm.throw(vm.newErrorFromError(err), false)
}

func (vm *VM) throw(err *RuntimeError, noTrace bool) error {
	if !noTrace {
		err.addTrace(vm.getSourcePos())
	}

	// firstly check our frame has error handler
	if vm.curFrame.errHandlers.hasHandler() {
		return vm.handleThrownError(vm.curFrame, err)
	}

	// find previous frames having error handler
	var frame *frame
	index := vm.frameIndex - 2

	for index >= 0 {
		f := &(vm.frames[index])
		err.addTrace(getFrameSourcePos(f))
		if f.errHandlers.hasHandler() {
			frame = f
			break
		}
		f.freeVars = nil
		f.fn = nil
		f.args = nil
		f.namedArgs = nil
		index--
	}

	if frame == nil || index < 0 {
		// not handled, exit
		return err
	}

	vm.frameIndex = index + 1

	if e := vm.handleThrownError(frame, err); e != nil {
		return e
	}

	vm.curFrame = frame
	vm.curFrame.fn = frame.fn
	vm.curInsts = frame.fn.Instructions

	return nil
}

func (vm *VM) handleThrownError(frame *frame, err *RuntimeError) error {
	frame.errHandlers.err = err
	handler := frame.errHandlers.last()

	// if we have catch>0 goto catch else follow finally (one of them must be set)
	if handler.catch > 0 {
		vm.ip = handler.catch - 1
	} else if handler.finally > 0 {
		vm.ip = handler.finally - 1
	} else {
		frame.errHandlers.pop()
		return vm.throw(err, false)
	}

	if vm.sp >= handler.sp {
		for i := vm.sp; i >= handler.sp; i-- {
			vm.stack[i] = nil
		}
	}

	vm.sp = handler.sp
	return nil
}

func (vm *VM) xOpCallName() (err error) {
	var (
		numArgs     = int(vm.curInsts[vm.ip+1])
		flags       = OpCallFlag(vm.curInsts[vm.ip+2]) // 0 or 1
		basePointer = vm.sp - numArgs - 1
		kwCount     int
		expandArgs  int
	)

	if flags.Has(OpCallFlagVarArgs) {
		expandArgs++
	}
	if flags.Has(OpCallFlagNamedArgs) {
		kwCount++
		basePointer--
	}
	if flags.Has(OpCallFlagVarNamedArgs) {
		kwCount++
		basePointer--
	}

	name := vm.stack[vm.sp-1]
	obj := vm.stack[basePointer-1]

	vm.sp--
	vm.stack[vm.sp] = nil

	if nameCaller, ok := obj.(NameCallerObject); ok {
		c := Call{
			VM:   vm,
			Args: []Array{nil},
		}

		if flags.Has(OpCallFlagVarArgs) {
			switch t := vm.stack[basePointer+numArgs-1].(type) {
			case Array:
				c.Args = append(c.Args, t)
			default:
				var values Object
				if values, err = Val(vm.Builtins.Call(BuiltinValues, Call{VM: vm, Args: Args{Array{t}}})); err != nil {
					return
				}
				c.Args = append(c.Args, values.(Array))
			}
		}

		if flags.Has(OpCallFlagNamedArgs) || flags.Has(OpCallFlagVarNamedArgs) {
			if c.NamedArgs, err = vm.getCalledNamedArgs(flags); err != nil {
				return
			}
		}

		c.Args[0] = vm.stack[vm.sp-numArgs-kwCount : vm.sp-expandArgs-kwCount]
		ret, err := Val(nameCaller.CallName(name.ToString(), c))
		for i := 0; i < numArgs+kwCount; i++ {
			vm.sp--
			vm.stack[vm.sp] = nil
		}
		if err != nil {
			return err
		}

		vm.stack[vm.sp-1] = ret
		vm.ip += 2
		return nil
	}

	var v Object
	if ig, _ := obj.(IndexGetter); ig != nil {
		if v, err = Val(ig.IndexGet(vm, name)); err != nil {
			return
		}
	} else {
		return ErrNotIndexable
	}

	vm.stack[vm.sp-numArgs-kwCount-1] = v
	return vm.xOpCallAny(v, numArgs, flags)
}

func (vm *VM) xOpCall() error {
	numArgs := int(vm.curInsts[vm.ip+1])
	flags := OpCallFlag(vm.curInsts[vm.ip+2])
	kwCount := 0
	if flags.Has(OpCallFlagNamedArgs) {
		kwCount++
	}
	if flags.Has(OpCallFlagVarNamedArgs) {
		kwCount++
	}
	callee := vm.stack[vm.sp-numArgs-kwCount-1]
	return vm.xOpCallAny(callee, numArgs, flags)
}

func (vm *VM) xOpCallAny(callee Object, numArgs int, flags OpCallFlag) error {
do:
	switch t := callee.(type) {
	case *CompiledFunction:
		return vm.xOpCallCompiled(t, numArgs, flags)
	case MethodCaller:
		if !t.HasCallerMethods() {
			callee = t.Caller()
			goto do
		}
	}
	return vm.xOpCallObject(callee, numArgs, flags)
}

func (vm *VM) xOpCallCompiled(cfunc *CompiledFunction, numArgs int, flags OpCallFlag) (err error) {
	var (
		basePointer = vm.sp - numArgs
		numLocals   = cfunc.NumLocals
		numParams   = len(cfunc.Params)
		namedParams NamedArgs
		args        = Args{nil, nil}
	)

	if flags.Has(OpCallFlagNamedArgs) {
		basePointer--
	}
	if flags.Has(OpCallFlagVarNamedArgs) {
		basePointer--
	}

	args[0] = vm.stack[basePointer : basePointer+numArgs]

	if flags.Has(OpCallFlagNamedArgs) || flags.Has(OpCallFlagVarNamedArgs) {
		if namedParams, err = vm.getCalledNamedArgs(flags); err != nil {
			return
		}
		if !cfunc.NamedParams.variadic && namedParams.sources != nil {
			if err := namedParams.CheckNamesFromSet(cfunc.NamedParamsMap); err != nil {
				return err
			}
		}

		if len(cfunc.NamedParams.Params) > 0 {
			for _, param := range cfunc.NamedParams.Params {
				if l := len(param.Type); l > 0 {
					if v := namedParams.MustGetValueOrNil(param.Name); v != nil {
						var typeso Object
						if l == 1 {
							if typeso, err = vm.GetSymbolValue(param.Type[0]); err != nil {
								return
							}
						} else {
							types := make(Array, l)
							for i, symbol := range param.Type {
								if types[i], err = vm.GetSymbolValue(symbol); err != nil {
									return
								}
							}
							typeso = types
						}

						var badTypes string
						if badTypes, err = NamedParamTypeCheck(param.Name, typeso, v); err != nil {
							return
						} else if badTypes != "" {
							err = NewArgumentTypeError(
								"types of named param '"+param.Name+"'",
								badTypes,
								typeso.ToString(),
							)
							return
						}
					}
				}
			}
		}
	}

	if flags.Has(OpCallFlagVarArgs) {
		var (
			arrSize  int
			vargs    = vm.stack[basePointer+numArgs-1]
			vargsArr Array
		)
		args[0] = args[0][:numArgs-1]
		if arr, ok := vargs.(Array); ok {
			arrSize = len(arr)
			vargsArr = arr
		} else {
			if vargsArr, err = ValuesOf(vm, vargs, &NamedArgs{}); err != nil {
				return
			}
			arrSize = len(vargsArr)
		}

		if cfunc.Params.Var() {
			if numArgs < numParams {
				// f := func(a, ...b) {}
				// f(...[1]) // f(...[1, 2])
				if arrSize+numArgs < numParams {
					// f := func(a, ...b) {}
					// f(...[])
					return ErrWrongNumArguments.NewError(
						wantGEqXGotY(numParams-1, arrSize+numArgs-1),
					)
				}
				tempBuf := make(Array, 0, arrSize+numArgs)
				tempBuf = append(tempBuf,
					vm.stack[basePointer:basePointer+numArgs-1]...)
				tempBuf = append(tempBuf,
					vargsArr...)
				copy(vm.stack[basePointer:], tempBuf[:numParams-1])
				arr := tempBuf[numParams-1:]
				args = append(args, arr)
				vm.stack[basePointer+numParams-1] = append(Array{}, arr...)
			} else if numArgs > numParams {
				// f := func(a, ...b) {} // a == 1  b == [2, 3]
				// f(1, 2, ...[3])
				arr := append(Array{},
					vm.stack[basePointer+numParams-1:basePointer+numArgs-1]...)
				arr = append(arr, vargsArr...)
				args[0] = arr
				vm.stack[basePointer+numParams-1] = arr
			}
		} else {
			if arrSize+numArgs-1 != numParams {
				// f := func(a, b) {}
				// f(1, ...[2, 3, 4])
				return ErrWrongNumArguments.NewError(
					wantEqXGotY(numParams, arrSize+numArgs-1),
				)
			}
			// f := func(a, b) {}
			// f(...[1, 2])
			args[1] = vargsArr
			copy(vm.stack[basePointer+numArgs-1:], vargsArr)
		}
	} else {
		args[0] = vm.stack[basePointer : basePointer+numArgs]

		if !cfunc.Params.Var() {
			if numArgs != numParams {
				return ErrWrongNumArguments.NewError(
					wantEqXGotY(numParams, numArgs),
				)
			}
		} else {
			if numArgs < numParams-1 {
				// f := func(a, ...b) {}
				// f()
				return ErrWrongNumArguments.NewError(
					wantGEqXGotY(numParams-1, numArgs),
				)
			}
			if numArgs == numParams-1 {
				// f := func(a, ...b) {} // a==1 b==[]
				// f(1)
				vm.stack[basePointer+numArgs] = Array{}
			} else {
				// f := func(a, ...b) {} // a == 1  b == [] // a == 1  b == [2, 3]
				// f(1, 2) // f(1, 2, 3)
				args[0] = vm.stack[basePointer : basePointer+len(cfunc.Params)-1]
				arr := append(Array{}, vm.stack[basePointer+numParams-1:basePointer+numArgs]...)
				vm.stack[basePointer+numParams-1] = arr
				args[1] = arr
			}
		}
	}

	if cfunc.NamedParams.len > 0 {
		var i int
		for ; i < cfunc.NamedParams.len; i++ {
			vm.stack[basePointer+numParams+i] = Nil
		}
		// define var namedArgs
		if cfunc.NamedParams.variadic {
			vm.stack[basePointer+numParams+i-1] = &namedParams
		}
	} else {
		for i := numParams; i < numLocals; i++ {
			vm.stack[basePointer+i] = Nil
		}
	}

	if err = cfunc.ValidateParamTypes(vm, args); err != nil {
		return
	}

	// test if it's tail-call
	if cfunc == vm.curFrame.fn { // recursion
		nextOp := Opcode(vm.curInsts[vm.ip+2+1])
		if nextOp == OpReturn ||
			(nextOp == OpPop && OpReturn == Opcode(vm.curInsts[vm.ip+2+2])) {
			curBp := vm.curFrame.basePointer
			args = args.Copy().(Args)
			copy(vm.stack[curBp:curBp+len(cfunc.Params)+cfunc.NamedParams.len], vm.stack[basePointer:])
			newSp := vm.sp - numArgs - 1
			if flags.Has(OpCallFlagNamedArgs) {
				newSp--
			}
			if flags.Has(OpCallFlagVarNamedArgs) {
				newSp--
			}

			for i := vm.sp; i >= newSp; i-- {
				vm.stack[i] = nil
			}

			vm.sp = newSp
			vm.ip = -1                    // reset ip to beginning of the frame
			vm.curFrame.errHandlers = nil // reset error handlers if any set
			vm.curFrame.namedArgs = &namedParams
			vm.curFrame.args = args
			return nil
		}
	}
	frame := &(vm.frames[vm.frameIndex])
	vm.frameIndex++
	if vm.frameIndex > frameSize-1 {
		return ErrStackOverflow
	}
	frame.fn = cfunc
	frame.namedArgs = &namedParams
	frame.args = args
	frame.freeVars = cfunc.Free
	frame.errHandlers = nil
	frame.basePointer = basePointer
	vm.curFrame.ip = vm.ip + 2
	vm.curInsts = cfunc.Instructions
	vm.curFrame = frame
	vm.sp = basePointer + numLocals
	vm.ip = -1
	return nil
}

func (vm *VM) xOpCallObject(co_ Object, numArgs int, flags OpCallFlag) (err error) {
	if !Callable(co_) {
		return ErrNotCallable.NewError(co_.Type().Name())
	}

	var (
		co         = co_.(CallerObject)
		kwCount    int
		expandArgs int

		basePointer = vm.sp - numArgs
		args        Array
		namedArgs   NamedArgs
	)

	if flags.Has(OpCallFlagVarArgs) {
		expandArgs++
	}
	if flags.Has(OpCallFlagNamedArgs) {
		kwCount++
		basePointer--
	}
	if flags.Has(OpCallFlagVarNamedArgs) {
		kwCount++
		basePointer--
	}

	if flags.Has(OpCallFlagNamedArgs) || flags.Has(OpCallFlagVarNamedArgs) {
		if namedArgs, err = vm.getCalledNamedArgs(flags); err != nil {
			return
		}
	}

	args = vm.stack[basePointer : basePointer+numArgs-expandArgs]
	var vargs Array

	if expandArgs > 0 {
		if arr, ok := vm.stack[basePointer+numArgs-1].(Array); ok {
			vargs = arr
		} else if vargs, err = ValuesOf(vm, vm.stack[basePointer+numArgs-1], &NamedArgs{}); err != nil {
			return
		}
	}

	var (
		c      = Call{VM: vm, Args: Args{args, vargs}, NamedArgs: namedArgs}
		result Object
	)

	if result, err = Val(co.Call(c)); err != nil {
		return err
	}

	for i := 0; i < numArgs+kwCount; i++ {
		vm.sp--
		vm.stack[vm.sp] = nil
	}

	vm.stack[vm.sp-1] = result
	vm.ip += 2
	return nil
}

func (vm *VM) xOpUnary() error {
	tok := token.Token(vm.curInsts[vm.ip+1])
	right := vm.stack[vm.sp-1]
	var value Object

	switch tok {
	case token.Not:
		switch right.(type) {
		case Flag:
			vm.stack[vm.sp-1] = Flag(right == No)
		default:
			vm.stack[vm.sp-1] = Bool(right.IsFalsy())
		}
		vm.ip++
		return nil
	case token.Sub:
		switch o := right.(type) {
		case Int:
			value = -o
		case Float:
			value = -o
		case Char:
			value = Int(-o)
		case Uint:
			value = -o
		case Bool:
			if o {
				value = Int(-1)
			} else {
				value = Int(0)
			}
		default:
			goto invalidType
		}
	case token.Xor:
		switch o := right.(type) {
		case Int:
			value = ^o
		case Uint:
			value = ^o
		case Char:
			value = ^Int(o)
		case Bool:
			if o {
				value = ^Int(1)
			} else {
				value = ^Int(0)
			}
		case Flag:
			if o {
				value = ^Int(1)
			} else {
				value = ^Int(0)
			}
		default:
			goto invalidType
		}
	case token.Add:
		switch o := right.(type) {
		case Int, Uint, Float, Char:
			value = right
		case Bool:
			if o {
				value = Int(1)
			} else {
				value = Int(0)
			}
		case Flag:
			if o {
				value = Int(1)
			} else {
				value = Int(0)
			}
		default:
			goto invalidType
		}
	case token.Null:
		vm.stack[vm.sp-1] = Bool(right == Nil)
		vm.ip++
		return nil
	case token.NotNull:
		vm.stack[vm.sp-1] = Bool(right != Nil)
		vm.ip++
		return nil
	default:
		return ErrInvalidOperator.NewError(
			fmt.Sprintf("invalid for '%s': '%s'",
				tok.String(), right.Type().Name()))
	}

	vm.stack[vm.sp-1] = value
	vm.ip++
	return nil

invalidType:
	return ErrType.NewError(
		fmt.Sprintf("invalid type for unary '%s': '%s'",
			tok.String(), right.Type().Name()))
}

func (vm *VM) xOpSliceIndex() error {
	obj := vm.stack[vm.sp-3]
	left := vm.stack[vm.sp-2]
	right := vm.stack[vm.sp-1]
	vm.stack[vm.sp-3] = nil
	vm.stack[vm.sp-2] = nil
	vm.stack[vm.sp-1] = nil
	vm.sp -= 3

	var objlen int
	var isbytes bool

	switch obj := obj.(type) {
	case Array:
		objlen = len(obj)
	case Str:
		objlen = len(obj)
	case Bytes:
		isbytes = true
		objlen = len(obj)
	case Slicer:
		objlen = obj.Length()
	default:
		return ErrType.NewError(obj.Type().Name(), "cannot be sliced")
	}

	var low int
	switch v := left.(type) {
	case *NilType:
		low = 0
	case Int:
		low = int(v)
	case Uint:
		low = int(v)
	case Char:
		low = int(v)
	default:
		return ErrType.NewError("invalid first index type", left.Type().Name())
	}

	var high int
	switch v := right.(type) {
	case *NilType:
		high = objlen
	case Int:
		high = int(v)
	case Uint:
		high = int(v)
	case Char:
		high = int(v)
	default:
		return ErrType.NewError("invalid second index type", right.Type().Name())
	}

	if low < 0 {
		low = objlen + low
	}

	if high < 0 {
		high = objlen + high
	}
	if isbytes {
		objlen = cap(obj.(Bytes))
	}

	if high == 0 && low > 0 {
		high = objlen
	}

	if low > high {
		return ErrInvalidIndex.NewError(fmt.Sprintf("[%d:%d]", low, high))
	}

	if low < 0 || high < 0 || high > objlen {
		return ErrIndexOutOfBounds.NewError(fmt.Sprintf("[%d:%d]", low, high))
	}

	switch obj := obj.(type) {
	case Array:
		vm.stack[vm.sp] = obj[low:high]
	case Str:
		vm.stack[vm.sp] = obj[low:high]
	case Bytes:
		vm.stack[vm.sp] = obj[low:high]
	case Slicer:
		vm.stack[vm.sp] = obj.Slice(low, high)
	}

	vm.sp++
	return nil
}

func (vm *VM) newError(err *Error) *RuntimeError {
	var fileset *source.SourceFileSet
	if vm.bytecode != nil {
		fileset = vm.bytecode.FileSet
	}
	return &RuntimeError{Err: err, fileSet: fileset}
}

func (vm *VM) newErrorFromObject(object Object) *RuntimeError {
	switch v := object.(type) {
	case *RuntimeError:
		return v
	case *Error:
		return vm.newError(v)
	default:
		return vm.newError(&Error{Message: v.ToString()})
	}
}
func (vm *VM) newErrorFromError(err error) *RuntimeError {
	if v, ok := err.(Object); ok {
		return vm.newErrorFromObject(v)
	}
	return vm.newError(&Error{Message: err.Error(), Cause: err})
}

func (vm *VM) getSourcePos() source.Pos {
	if vm.curFrame == nil || vm.curFrame.fn == nil {
		return source.NoPos
	}
	return vm.curFrame.fn.SourcePos(vm.ip)
}

func (vm *VM) getCalledNamedArgs(flags OpCallFlag) (namedArgs NamedArgs, err error) {
	var (
		expand   = 0
		hasPairs = 0
	)

	if flags.Has(OpCallFlagNamedArgs) {
		hasPairs = 1
	}

	if flags.Has(OpCallFlagVarNamedArgs) {
		expand = 1
	}

	if hasPairs > 0 {
		if err = namedArgs.Add(vm.stack[vm.sp-expand-hasPairs]); err != nil {
			return
		}
	}

	if expand > 0 {
		if err = namedArgs.Add(vm.stack[vm.sp-expand]); err != nil {
			return
		}
	}

	return
}

type errHandler struct {
	sp       int
	catch    int
	finally  int
	returnTo int
}

type errHandlers struct {
	handlers []errHandler
	err      *RuntimeError
}

func (t *errHandlers) hasError() bool {
	return t != nil && t.err != nil
}

func (t *errHandlers) pop() bool {
	if t == nil || len(t.handlers) == 0 {
		return false
	}
	t.handlers = t.handlers[:len(t.handlers)-1]
	return true
}

func (t *errHandlers) last() *errHandler {
	if t == nil || len(t.handlers) == 0 {
		return nil
	}
	return &t.handlers[len(t.handlers)-1]
}

func (t *errHandlers) hasHandler() bool {
	return t != nil && len(t.handlers) > 0
}

func (t *errHandlers) findFinally(upto int) int {
	if t == nil {
		return 0
	}

start:
	index := len(t.handlers) - 1
	if index < upto || index < 0 {
		return 0
	}

	p := t.handlers[index].finally
	if p == 0 {
		t.pop()
		goto start
	}
	return p
}

func (t *errHandlers) hasReturnTo() int {
	if t.hasHandler() {
		return t.handlers[len(t.handlers)-1].returnTo
	}
	return 0
}

type frame struct {
	fn          *CompiledFunction
	freeVars    []*ObjectPtr
	ip          int
	basePointer int
	errHandlers *errHandlers
	args        Args
	namedArgs   *NamedArgs
	defers      []func()
}

func (f *frame) Defer(fn func()) {
	f.defers = append(f.defers, fn)
}

func getFrameSourcePos(frame *frame) source.Pos {
	if frame == nil || frame.fn == nil {
		return source.NoPos
	}
	return frame.fn.SourcePos(frame.ip + 1)
}

func wantEqXGotY(x, y int) string {
	buf := make([]byte, 0, 20)
	buf = append(buf, "want="...)
	buf = strconv.AppendInt(buf, int64(x), 10)
	buf = append(buf, " got="...)
	buf = strconv.AppendInt(buf, int64(y), 10)
	return string(buf)
}

func wantGEqXGotY(x, y int) string {
	buf := make([]byte, 0, 20)
	buf = append(buf, "want>="...)
	buf = strconv.AppendInt(buf, int64(x), 10)
	buf = append(buf, " got="...)
	buf = strconv.AppendInt(buf, int64(y), 10)
	return string(buf)
}

// Ported from runtime/debug.Stack
func debugStack() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, false)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

type vmPool struct {
	mu   sync.Mutex
	root *VM
	vms  map[*VM]struct{}
}

func (v *vmPool) abort(vm *VM) {
	v.mu.Lock()
	defer v.mu.Unlock()

	for vm := range v.vms {
		vm.Abort()
	}
}

func (v *vmPool) acquire(cf *CompiledFunction, usePool bool) *VM {
	var vm *VM
	if usePool {
		vm = vmSyncPool.Get().(*VM)
	} else {
		vm = &VM{bytecode: &Bytecode{}}
	}
	return v.root.pool._acquire(vm, cf)
}

func (v *vmPool) _acquire(vm *VM, cf *CompiledFunction) *VM {
	v.mu.Lock()
	defer v.mu.Unlock()

	vm.bytecode.FileSet = v.root.bytecode.FileSet
	vm.bytecode.Constants = v.root.bytecode.Constants
	vm.bytecode.NumModules = v.root.bytecode.NumModules
	vm.bytecode.Main = cf
	vm.constants = v.root.bytecode.Constants
	vm.modulesCache = v.root.modulesCache
	vm.pool = vmPool{
		root: v.root,
	}
	vm.noPanic = v.root.noPanic
	vm.SetupOpts = v.root.SetupOpts
	vm.ObjectToWriter = v.root.ObjectToWriter

	if v.vms == nil {
		v.vms = make(map[*VM]struct{})
	}
	v.vms[vm] = struct{}{}

	return vm
}

func (v *vmPool) release(vm *VM) {
	v.root.pool._release(vm)
}

func (v *vmPool) _release(vm *VM) {
	v.mu.Lock()
	delete(v.vms, vm)
	v.mu.Unlock()

	bc := vm.bytecode
	*bc = Bytecode{}
	*vm = VM{bytecode: bc}
	vmSyncPool.Put(vm)
}

func (v *vmPool) clear() {
	v.mu.Lock()
	defer v.mu.Unlock()

	for vm := range v.vms {
		delete(v.vms, vm)
	}
}

var vmSyncPool = sync.Pool{
	New: func() any {
		return &VM{
			bytecode: &Bytecode{},
		}
	},
}
