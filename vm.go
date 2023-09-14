// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/gad-lang/gad/parser"
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
	globals      IndexGetter
	pool         vmPool
	mu           sync.Mutex
	err          error
	noPanic      bool
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

func (vm *VM) init(opts *RunOpts) (Object, error) {
	if vm.bytecode == nil || vm.bytecode.Main == nil {
		return nil, errors.New("invalid Bytecode")
	}

	vm.err = nil
	atomic.StoreInt64(&vm.abort, 0)
	vm.initGlobals(opts.Globals)
	vm.initLocals(opts.Args.Values(), opts.NamedArgs)
	vm.initCurrentFrame(opts.Args, opts.NamedArgs)
	vm.frameIndex = 1
	vm.ip = -1
	vm.sp = vm.curFrame.fn.NumLocals

	// Resize modules cache or create it if not exists.
	// Note that REPL can set module cache before running, don't recreate it add missing indexes.
	if diff := vm.bytecode.NumModules - len(vm.modulesCache); diff > 0 {
		for i := 0; i < diff; i++ {
			vm.modulesCache = append(vm.modulesCache, nil)
		}
	}

	for run := true; run; {
		run = vm.run()
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

func (vm *VM) initGlobals(globals IndexGetter) {
	if globals == nil {
		globals = Map{}
	}
	vm.globals = globals
}

func (vm *VM) initLocals(args Array, namedArgs *NamedArgs) {
	// init all params as nil
	main := vm.bytecode.Main
	numParams := main.Params.Len
	numNamedParams := main.NamedParams.len
	locals := vm.stack[:main.NumLocals]

	// fix num reveived args < num expected args
	if diff := numParams - len(args); diff > 0 {
		if main.Params.Var {
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
		if main.Params.Var {
			locals[numParams-1] = Array{}
		}
		copy(locals, args)
		return
	}

	if main.Params.Var {
		vargs := args[numParams-1:]
		arr := make(Array, 0, len(vargs))
		locals[numParams-1] = append(arr, vargs...)
	} else if numParams > 0 {
		locals[numParams-1] = args[numParams-1]
	}

	if numParams > 0 {
		copy(locals, args[:numParams-1])
	}

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

func (vm *VM) initCurrentFrame(args Args, named *NamedArgs) {
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
}

func (vm *VM) clearCurrentFrame() {
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
			vm:   vm,
			Args: []Array{nil},
		}

		if flags.Has(OpCallFlagVarArgs) {
			if arr, ok := vm.stack[basePointer+numArgs-1].(Array); ok {
				c.Args = append(c.Args, arr)
			} else {
				return NewArgumentTypeError("last", "array",
					vm.stack[basePointer+numArgs-1].TypeName())
			}
		}

		if flags.Has(OpCallFlagNamedArgs) || flags.Has(OpCallFlagVarNamedArgs) {
			if c.NamedArgs, err = vm.getCalledNamedArgs(flags); err != nil {
				return
			}
		} else {
			c.NamedArgs = &NamedArgs{}
		}

		c.Args[0] = vm.stack[vm.sp-numArgs-kwCount : vm.sp-expandArgs-kwCount]
		ret, err := nameCaller.CallName(name.String(), c)
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
		if v, err = ig.IndexGet(name); err != nil {
			return
		}
	} else {
		return ErrNotIndexable
	}

	vm.stack[vm.sp-numArgs-1] = v
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
	if cfunc, ok := callee.(*CompiledFunction); ok {
		return vm.xOpCallCompiled(cfunc, numArgs, flags)
	}
	return vm.xOpCallObject(callee, numArgs, flags)
}

func (vm *VM) xOpCallCompiled(cfunc *CompiledFunction, numArgs int, flags OpCallFlag) (err error) {
	var (
		basePointer = vm.sp - numArgs
		numLocals   = cfunc.NumLocals
		numParams   = cfunc.Params.Len
		namedParams *NamedArgs
		args        = Args{nil}
	)

	if flags.Has(OpCallFlagNamedArgs) {
		basePointer--
	}
	if flags.Has(OpCallFlagVarNamedArgs) {
		basePointer--
	}

	if flags.Has(OpCallFlagNamedArgs) || flags.Has(OpCallFlagVarNamedArgs) {
		if namedParams, err = vm.getCalledNamedArgs(flags); err != nil {
			return
		}
		if !cfunc.NamedParams.variadic && namedParams != nil {
			if err := namedParams.CheckNamesFromSet(cfunc.NamedParamsMap); err != nil {
				return err
			}
		}
	}

	if flags.Has(OpCallFlagVarArgs) {
		var arrSize int
		if arr, ok := vm.stack[basePointer+numArgs-1].(Array); ok {
			arrSize = len(arr)
		} else {
			return NewArgumentTypeError("last", "array",
				vm.stack[basePointer+numArgs-1].TypeName())
		}
		if cfunc.Params.Var {
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
					vm.stack[basePointer+numArgs-1].(Array)...)
				copy(vm.stack[basePointer:], tempBuf[:numParams-1])
				arr := tempBuf[numParams-1:]
				args = append(args, arr)
				vm.stack[basePointer+numParams-1] = append(Array{}, arr...)
			} else if numArgs > numParams {
				// f := func(a, ...b) {} // a == 1  b == [2, 3]
				// f(1, 2, ...[3])
				arr := append(Array{},
					vm.stack[basePointer+numParams-1:basePointer+numArgs-1]...)
				arr = append(arr, vm.stack[basePointer+numArgs-1].(Array)...)
				args = append(args, arr)
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
			arr := vm.stack[basePointer+numArgs-1].(Array)
			args[0] = arr
			copy(vm.stack[basePointer+numArgs-1:], arr)
		}
	} else {
		args[0] = vm.stack[basePointer : basePointer+numArgs]

		if !cfunc.Params.Var {
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
				args[0] = vm.stack[basePointer : basePointer+cfunc.Params.Len-1]
				arr := append(Array{}, vm.stack[basePointer+numParams-1:basePointer+numArgs]...)
				vm.stack[basePointer+numParams-1] = arr
				args = append(args, arr)
			}
		}
	}

	if cfunc.NamedParams.len > 0 {
		if namedParams == nil {
			namedParams = NewNamedArgs()
		}

		var i int
		for ; i < cfunc.NamedParams.len; i++ {
			vm.stack[basePointer+numParams+i] = Nil
		}
		// define var namedArgs
		if cfunc.NamedParams.variadic {
			vm.stack[basePointer+numParams+i-1] = namedParams
		}
	} else {
		for i := numParams; i < numLocals; i++ {
			vm.stack[basePointer+i] = Nil
		}
	}

	// test if it's tail-call
	if cfunc == vm.curFrame.fn { // recursion
		nextOp := vm.curInsts[vm.ip+2+1]
		if nextOp == OpReturn ||
			(nextOp == OpPop && OpReturn == vm.curInsts[vm.ip+2+2]) {
			curBp := vm.curFrame.basePointer
			args = args.Copy().(Args)
			copy(vm.stack[curBp:curBp+cfunc.Params.Len+cfunc.NamedParams.len], vm.stack[basePointer:])
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
			vm.curFrame.namedArgs = namedParams
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
	frame.namedArgs = namedParams
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

func (vm *VM) xOpCallObject(co Object, numArgs int, flags OpCallFlag) (err error) {
	if !Callable(co) {
		return ErrNotCallable.NewError(co.TypeName())
	}

	var (
		kwCount    int
		expandArgs int

		basePointer = vm.sp - numArgs
		args        Array
		namedArgs   *NamedArgs
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
	} else {
		namedArgs = &NamedArgs{}
	}

	args = vm.stack[basePointer : basePointer+numArgs-expandArgs]
	var vargs Array

	if expandArgs > 0 {
		if arr, ok := vm.stack[basePointer+numArgs-1].(Array); ok {
			vargs = arr
		} else {
			return NewArgumentTypeError("last", "array",
				vm.stack[basePointer+numArgs-1].TypeName())
		}
	}

	result, err := co.(CallerObject).Call(NewCall(vm, WithArgsV(args, vargs...), WithNamedArgs(namedArgs)))
	if err != nil {
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
		vm.stack[vm.sp-1] = Bool(right.IsFalsy())
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
				tok.String(), right.TypeName()))
	}

	vm.stack[vm.sp-1] = value
	vm.ip++
	return nil

invalidType:
	return ErrType.NewError(
		fmt.Sprintf("invalid type for unary '%s': '%s'",
			tok.String(), right.TypeName()))
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
	case String:
		objlen = len(obj)
	case Bytes:
		isbytes = true
		objlen = len(obj)
	default:
		return ErrType.NewError(obj.TypeName(), "cannot be sliced")
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
		return ErrType.NewError("invalid first index type", left.TypeName())
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
		return ErrType.NewError("invalid second index type", right.TypeName())
	}

	if low > high {
		return ErrInvalidIndex.NewError(fmt.Sprintf("[%d:%d]", low, high))
	}
	if isbytes {
		objlen = cap(obj.(Bytes))
	}
	if low < 0 || high < 0 || high > objlen {
		return ErrIndexOutOfBounds.NewError(fmt.Sprintf("[%d:%d]", low, high))
	}

	switch obj := obj.(type) {
	case Array:
		vm.stack[vm.sp] = obj[low:high]
	case String:
		vm.stack[vm.sp] = obj[low:high]
	case Bytes:
		vm.stack[vm.sp] = obj[low:high]
	}

	vm.sp++
	return nil
}

func (vm *VM) newError(err *Error) *RuntimeError {
	var fileset *parser.SourceFileSet
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
		return vm.newError(&Error{Message: v.String()})
	}
}
func (vm *VM) newErrorFromError(err error) *RuntimeError {
	if v, ok := err.(Object); ok {
		return vm.newErrorFromObject(v)
	}
	return vm.newError(&Error{Message: err.Error(), Cause: err})
}

func (vm *VM) getSourcePos() parser.Pos {
	if vm.curFrame == nil || vm.curFrame.fn == nil {
		return parser.NoPos
	}
	return vm.curFrame.fn.SourcePos(vm.ip)
}

func (vm *VM) getCalledNamedArgs(flags OpCallFlag) (namedArgs *NamedArgs, err error) {
	var (
		expand   = 0
		hasPairs = 0
	)

	namedArgs = &NamedArgs{}

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
}

func getFrameSourcePos(frame *frame) parser.Pos {
	if frame == nil || frame.fn == nil {
		return parser.NoPos
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
// pool. So it is possible to reuse a VM instance for multiple Invoke calls.
// This is useful when you want to execute multiple functions in a single VM.
// For example, you can use Acquire and Release to execute multiple functions
// in a single VM instance.
// Note that you should call Release after Acquire, if you want to reuse the VM.
// If you don't want to use the pool, you can just call Invoke method.
// It is unsafe to hold a reference to the VM after Release is called.
// Using VM pool is about three times faster than creating a new VM for each
// Invoke call.
type Invoker struct {
	vm         *VM
	child      *VM
	callee     Object
	isCompiled bool
	dorelease  bool
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
	if inv.isCompiled {
		return inv.child.RunOpts(&RunOpts{Globals: inv.vm.globals, Args: args, NamedArgs: namedArgs})
	}
	return inv.invokeObject(inv.callee, args)
}

func (inv *Invoker) invokeObject(co Object, args Args) (Object, error) {
	callee, _ := co.(CallerObject)
	if callee == nil {
		return Nil, ErrNotCallable.NewError(co.TypeName())
	}
	return callee.Call(Call{
		vm:   inv.vm,
		Args: args,
	})
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
