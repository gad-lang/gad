package gad

import (
	"fmt"
	"sync/atomic"

	"github.com/gad-lang/gad/token"
)

func (vm *VM) loop() {
	var op Opcode
VMLoop:
	for atomic.LoadInt64(&vm.abort) == 0 {
		vm.ip++
		op = Opcode(vm.curInsts[vm.ip])
		switch op {
		case OpConstant:
			cidx := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			obj := vm.constants[cidx]
			vm.stack[vm.sp] = obj
			vm.sp++
			vm.ip += 2
		case OpGetLocal:
			localIdx := int(vm.curInsts[vm.ip+1])
			value := vm.stack[vm.curFrame.basePointer+localIdx]
			if v, ok := value.(*ObjectPtr); ok {
				value = *v.Value
			}
			vm.stack[vm.sp] = value
			vm.sp++
			vm.ip++
		case OpSetLocal:
			localIndex := int(vm.curInsts[vm.ip+1])
			value := vm.stack[vm.sp-1]
			index := vm.curFrame.basePointer + localIndex
			if v, ok := vm.stack[index].(*ObjectPtr); ok {
				*v.Value = value
			} else {
				vm.stack[index] = value
			}
			vm.sp--
			vm.stack[vm.sp] = nil
			vm.ip++
		case OpBinaryOp:
			tok := token.Token(vm.curInsts[vm.ip+1])
			left, right := vm.stack[vm.sp-2], vm.stack[vm.sp-1]

			value, err := Val(vm.Builtins.Call(BuiltinBinaryOp, Call{VM: vm, Args: Args{Array{BinaryOperatorTypes[tok], left, right}}}))

			if err == nil {
				vm.stack[vm.sp-2] = value
				vm.sp--
				vm.stack[vm.sp] = nil
				vm.ip++
				continue
			}
			if err == ErrInvalidOperator {
				err = ErrInvalidOperator.NewError(tok.String())
			}
			if err = vm.throwGenErr(err); err != nil {
				vm.err = err
				return
			}
		case OpAndJump:
			if vm.stack[vm.sp-1].IsFalsy() {
				pos := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
				vm.ip = pos - 1
				continue
			}
			vm.stack[vm.sp-1] = nil
			vm.sp--
			vm.ip += 2
		case OpOrJump:
			if vm.stack[vm.sp-1].IsFalsy() {
				vm.stack[vm.sp-1] = nil
				vm.sp--
				vm.ip += 2
				continue
			}
			pos := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			vm.ip = pos - 1
		case OpJumpNil:
			if vm.stack[vm.sp-1] != Nil && !vm.stack[vm.sp-1].Equal(Nil) {
				vm.ip += 2
				continue
			}
			pos := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			vm.ip = pos - 1
		case OpJumpNotNil:
			if vm.stack[vm.sp-1] == Nil || vm.stack[vm.sp-1].Equal(Nil) {
				vm.sp--
				vm.ip += 2
				continue
			}
			pos := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			vm.ip = pos - 1
		case OpIsNil:
			v := vm.stack[vm.sp-1]
			if v == Nil {
				vm.stack[vm.sp-1] = True
			} else {
				niler, _ := v.(Niler)
				vm.stack[vm.sp-1] = Bool(niler != nil && niler.IsNil())
			}
		case OpNotIsNil:
			v := vm.stack[vm.sp-1]
			if v == Nil {
				vm.stack[vm.sp-1] = False
			} else {
				niler, _ := v.(Niler)
				vm.stack[vm.sp-1] = Bool(niler == nil || !niler.IsNil())
			}
		case OpEqual:
			left, right := vm.stack[vm.sp-2], vm.stack[vm.sp-1]
			vm.stack[vm.sp-2] = Bool(left.Equal(right))
			vm.sp--
			vm.stack[vm.sp] = nil
		case OpNotEqual:
			left, right := vm.stack[vm.sp-2], vm.stack[vm.sp-1]

			switch left := left.(type) {
			case Int:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			case Str:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			case Float:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			case Bool:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			case Uint:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			case Char:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			default:
				vm.stack[vm.sp-2] = Bool(!left.Equal(right))
			}
			vm.sp--
			vm.stack[vm.sp] = nil
		case OpTrue:
			vm.stack[vm.sp] = True
			vm.sp++
		case OpFalse:
			vm.stack[vm.sp] = False
			vm.sp++
		case OpYes:
			vm.stack[vm.sp] = Yes
			vm.sp++
		case OpNo:
			vm.stack[vm.sp] = No
			vm.sp++
		case OpCall:
			err := vm.xOpCall()
			if err == nil {
				continue
			}
			if err = vm.throwGenErr(err); err != nil {
				vm.err = err
				return
			}
		case OpCallName:
			err := vm.xOpCallName()
			if err == nil {
				continue
			}
			if err = vm.throwGenErr(err); err != nil {
				vm.err = err
				return
			}
		case OpReturn:
			numRet := vm.curInsts[vm.ip+1]
			bp := vm.curFrame.basePointer
			if bp == 0 {
				bp = vm.curFrame.fn.NumLocals + 1
			}
			if numRet == 1 {
				vm.stack[bp-1] = vm.stack[vm.sp-1]
			} else {
				vm.stack[bp-1] = Nil
			}

			for i := vm.sp - 1; i >= bp; i-- {
				vm.stack[i] = nil
			}

			vm.sp = bp
			if vm.frameIndex == 1 {
				return
			}
			vm.clearCurrentFrame()
			parent := &(vm.frames[vm.frameIndex-2])
			vm.frameIndex--
			vm.ip = parent.ip
			vm.curFrame = parent
			vm.curInsts = vm.curFrame.fn.Instructions
		case OpGetBuiltin:
			builtinIndex := BuiltinType(int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8)
			vm.stack[vm.sp] = vm.Builtins.Objects[builtinIndex]
			vm.sp++
			vm.ip += 2
		case OpClosure:
			constIdx := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			fn := vm.constants[constIdx].(*CompiledFunction)
			numFree := int(vm.curInsts[vm.ip+3])
			free := make([]*ObjectPtr, numFree)
			for i := 0; i < numFree; i++ {
				switch freeVar := (vm.stack[vm.sp-numFree+i]).(type) {
				case *ObjectPtr:
					free[i] = freeVar
				default:
					temp := vm.stack[vm.sp-numFree+i]
					free[i] = &ObjectPtr{
						Value: &temp,
					}
				}
				vm.stack[vm.sp-numFree+i] = nil
			}
			vm.sp -= numFree
			newFn := &CompiledFunction{
				Instructions: fn.Instructions,
				NumLocals:    fn.NumLocals,
				SourceMap:    fn.SourceMap,
				Free:         free,
				Params:       fn.Params,
				NamedParams:  fn.NamedParams,
				sourceFile:   fn.sourceFile,
				module:       fn.module,
			}
			vm.stack[vm.sp] = newFn
			vm.sp++
			vm.ip += 3
		case OpJump:
			vm.ip = (int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8) - 1
		case OpJumpFalsy:
			vm.sp--
			obj := vm.stack[vm.sp]
			vm.stack[vm.sp] = nil

			var falsy bool
			switch obj := obj.(type) {
			case Bool:
				falsy = obj.IsFalsy()
			case Int:
				falsy = obj.IsFalsy()
			case Uint:
				falsy = obj.IsFalsy()
			case Float:
				falsy = obj.IsFalsy()
			case Str:
				falsy = obj.IsFalsy()
			default:
				falsy = obj.IsFalsy()
			}
			if falsy {
				vm.ip = (int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8) - 1
				continue
			}
			vm.ip += 2
		case OpGetGlobal:
			cidx := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			index := vm.constants[cidx]
			var ret Object
			var err error
			ret, err = Val(vm.globals.IndexGet(vm, index))

			if err != nil {
				if err := vm.throwGenErr(err); err != nil {
					vm.err = err
					return
				}
				continue
			}

			if ret == nil {
				vm.stack[vm.sp] = Nil
			} else {
				vm.stack[vm.sp] = ret
			}

			vm.ip += 2
			vm.sp++
		case OpSetGlobal:
			cidx := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			index := vm.constants[cidx]
			value := vm.stack[vm.sp-1]

			if v, ok := value.(*ObjectPtr); ok {
				value = *v.Value
			}

			if err := vm.globals.IndexSet(vm, index, value); err != nil {
				if err := vm.throwGenErr(err); err != nil {
					vm.err = err
					return
				}
				continue
			}

			vm.ip += 2
			vm.sp--
			vm.stack[vm.sp] = nil
		case OpArray:
			numItems := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			arr := make(Array, numItems)
			copy(arr, vm.stack[vm.sp-numItems:vm.sp])
			vm.sp -= numItems
			vm.stack[vm.sp] = arr

			for i := vm.sp + 1; i < vm.sp+numItems+1; i++ {
				vm.stack[i] = nil
			}

			vm.sp++
			vm.ip += 2
		case OpDict:
			numItems := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			kv := make(Dict)

			for i := vm.sp - numItems; i < vm.sp; i += 2 {
				key := vm.stack[i]
				value := vm.stack[i+1]
				kv[key.ToString()] = value
				vm.stack[i] = nil
				vm.stack[i+1] = nil
			}
			vm.sp -= numItems
			vm.stack[vm.sp] = kv
			vm.sp++
			vm.ip += 2
		case OpKeyValue:
			var (
				hasValue = int(vm.curInsts[vm.ip+1])
				kv       KeyValue
			)

			kv.K = vm.stack[vm.sp-1-hasValue]

			if hasValue == 1 {
				kv.V = vm.stack[vm.sp-1]
				vm.stack[vm.sp-1] = nil
			} else {
				kv.V = No
			}

			vm.sp -= 1 + hasValue
			vm.stack[vm.sp] = &kv

			vm.sp++
			vm.ip += 1
		case OpKeyValueArray:
			var (
				numItems = int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
				arr      = make(KeyValueArray, numItems)
			)

			for i, v := range vm.stack[vm.sp-numItems : vm.sp] {
				arr[i] = v.(*KeyValue)
			}

			vm.sp -= numItems
			vm.stack[vm.sp] = arr

			for i := vm.sp + 1; i < vm.sp+numItems+1; i++ {
				vm.stack[i] = nil
			}

			vm.sp++
			vm.ip += 2
		case OpTextWriter:
			numSel := int(vm.curInsts[vm.ip+1])
			tp := vm.sp - 1 - numSel
			value, null, abort := vm.xIndexGet(numSel, vm.stack[tp])
			if abort {
				return
			}
			if null {
				continue VMLoop
			}
			vm.stack[tp] = value
			vm.sp = tp + 1
			vm.ip++
		case OpGetIndex:
			numSel := int(vm.curInsts[vm.ip+1])
			tp := vm.sp - 1 - numSel
			value, null, abort := vm.xIndexGet(numSel, vm.stack[tp])
			if abort {
				return
			}
			if null {
				continue VMLoop
			}
			vm.stack[tp] = value
			vm.sp = tp + 1
			vm.ip++
		case OpSetIndex:
			value := vm.stack[vm.sp-3]
			target := vm.stack[vm.sp-2]
			if is, _ := target.(IndexSetter); is != nil {
				index := vm.stack[vm.sp-1]

				err := is.IndexSet(vm, index, value)

				if err != nil {
					switch err {
					case ErrNotIndexAssignable:
						err = ErrNotIndexAssignable.NewError(is.Type().Name())
					case ErrIndexOutOfBounds:
						err = ErrIndexOutOfBounds.NewError(index.ToString())
					}
					if err = vm.throwGenErr(err); err != nil {
						vm.err = err
						return
					}
					continue
				}
			} else {
				if err := vm.throwGenErr(ErrNotIndexAssignable.NewError(target.Type().Name())); err != nil {
					vm.err = err
					return
				}
				continue
			}

			vm.stack[vm.sp-3] = nil
			vm.stack[vm.sp-2] = nil
			vm.stack[vm.sp-1] = nil
			vm.sp -= 3
		case OpSliceIndex:
			err := vm.xOpSliceIndex()
			if err == nil {
				continue
			}
			if err = vm.throwGenErr(err); err != nil {
				vm.err = err
				return
			}
		case OpGetFree:
			freeIndex := int(vm.curInsts[vm.ip+1])
			vm.stack[vm.sp] = *vm.curFrame.freeVars[freeIndex].Value
			vm.sp++
			vm.ip++
		case OpSetFree:
			freeIndex := int(vm.curInsts[vm.ip+1])
			*vm.curFrame.freeVars[freeIndex].Value = vm.stack[vm.sp-1]
			vm.sp--
			vm.stack[vm.sp] = nil
			vm.ip++
		case OpGetLocalPtr:
			localIndex := int(vm.curInsts[vm.ip+1])
			var freeVar *ObjectPtr
			value := vm.stack[vm.curFrame.basePointer+localIndex]

			if obj, ok := value.(*ObjectPtr); ok {
				freeVar = obj
			} else {
				freeVar = &ObjectPtr{Value: &value}
				vm.stack[vm.curFrame.basePointer+localIndex] = freeVar
			}

			vm.stack[vm.sp] = freeVar
			vm.sp++
			vm.ip++
		case OpGetFreePtr:
			freeIndex := int(vm.curInsts[vm.ip+1])
			value := vm.curFrame.freeVars[freeIndex]
			vm.stack[vm.sp] = value
			vm.sp++
			vm.ip++
		case OpDefineLocal:
			localIndex := int(vm.curInsts[vm.ip+1])
			vm.stack[vm.curFrame.basePointer+localIndex] = vm.stack[vm.sp-1]
			vm.sp--
			vm.stack[vm.sp] = nil
			vm.ip++
		case OpNil:
			vm.stack[vm.sp] = Nil
			vm.sp++
		case OpStdIn:
			vm.stack[vm.sp] = vm.StdIn
			vm.sp++
		case OpStdOut:
			vm.stack[vm.sp] = vm.StdOut
			vm.sp++
		case OpStdErr:
			vm.stack[vm.sp] = vm.StdErr
			vm.sp++
		case OpDotName:
			vm.stack[vm.sp] = Str(vm.curFrame.fn.module.Name)
			vm.sp++
		case OpDotFile:
			vm.stack[vm.sp] = Str(vm.curFrame.fn.module.File)
			vm.sp++
		case OpIsModule:
			vm.stack[vm.sp] = Bool(vm.curFrame.fn.module.Name != MainName)
			vm.sp++
		case OpCallee:
			vm.stack[vm.sp] = vm.curFrame.fn
			vm.sp++
		case OpArgs:
			vm.stack[vm.sp] = vm.curFrame.args
			vm.sp++
		case OpNamedArgs:
			vm.stack[vm.sp] = vm.curFrame.namedArgs
			vm.sp++
		case OpPop:
			vm.sp--
			vm.stack[vm.sp] = nil
		case OpIterInit:
			dst := vm.stack[vm.sp-1]

			if dst == Nil {
				vm.stack[vm.sp-1] = NewStateIteratorObject(vm, &nilIteratorObject{})
				continue
			}

			if _, it, err := ToIterator(vm, dst, &NamedArgs{}); err != nil {
				if err = vm.throwGenErr(err); err != nil {
					vm.err = err
					return
				}
			} else if it != nil {
				ito := NewStateIteratorObject(vm, it)
				vm.stack[vm.sp-1] = ito
				continue
			}

			var err error = ErrNotIterable.NewError(dst.Type().Name())
			if err = vm.throwGenErr(err); err != nil {
				vm.err = err
				return
			}
		case OpIterNext:
			iterator := vm.stack[vm.sp-1].(*StateIteratorObject)
			hasMore, err := iterator.Read()
			if err != nil {
				if err = vm.throwGenErr(err); err != nil {
					vm.err = err
					return
				}
			}
			vm.stack[vm.sp-1] = Bool(hasMore)
		case OpIterNextElse:
			iterator := vm.stack[vm.sp-1].(*StateIteratorObject)
			truePos := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			falsePos := int(vm.curInsts[vm.ip+4]) | int(vm.curInsts[vm.ip+3])<<8
			vm.ip += 4

			hasMore, err := iterator.Read()
			if err != nil {
				if err = vm.throwGenErr(err); err != nil {
					vm.err = err
					return
				}
			}

			vm.stack[vm.sp-1] = Bool(hasMore)

			if hasMore {
				vm.ip = truePos - 1
			} else {
				vm.ip = falsePos - 1
			}
		case OpIterKey:
			iterator := vm.stack[vm.sp-1]
			val := iterator.(*StateIteratorObject).State.Entry.K
			vm.stack[vm.sp-1] = val
		case OpIterValue:
			iterator := vm.stack[vm.sp-1]
			val := iterator.(*StateIteratorObject).State.Entry.V
			vm.stack[vm.sp-1] = val
		case OpLoadModule:
			cidx := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			midx := int(vm.curInsts[vm.ip+4]) | int(vm.curInsts[vm.ip+3])<<8
			value := vm.modulesCache[midx]

			if value == nil {
				// module cache is empty, load the object from constants
				vm.stack[vm.sp] = vm.constants[cidx]
				vm.sp++
				// load module by putting true for subsequent OpJumpFalsy
				// if module is a compiledFunction it will be called and result will be stored in module cache
				// if module is not a compiledFunction, copy of object will be stored in module cache
				vm.stack[vm.sp] = True
				vm.sp++
			} else {
				vm.stack[vm.sp] = value
				vm.sp++
				// no need to load the module, put false for subsequent OpJumpFalsy
				vm.stack[vm.sp] = False
				vm.sp++
			}

			vm.ip += 4
		case OpStoreModule:
			midx := int(vm.curInsts[vm.ip+2]) | int(vm.curInsts[vm.ip+1])<<8
			value := vm.stack[vm.sp-1]

			if v, ok := value.(Copier); ok {
				// store deep copy of the module if supported
				value = v.Copy()
				vm.stack[vm.sp-1] = value
			}

			vm.modulesCache[midx] = value
			vm.ip += 2
		case OpSetupTry:
			vm.xOpSetupTry()
		case OpSetupCatch:
			vm.xOpSetupCatch()
		case OpSetupFinally:
			vm.xOpSetupFinally()
		case OpThrow:
			err := vm.xOpThrow()
			if err != nil {
				vm.err = err
				return
			}
		case OpFinalizer:
			upto := int(vm.curInsts[vm.ip+1])

			pos := vm.curFrame.errHandlers.findFinally(upto)
			if pos <= 0 {
				vm.ip++
				continue
			}
			// go to finally if set
			handler := vm.curFrame.errHandlers.last()
			// save current ip to come back to same position
			handler.returnTo = vm.ip
			// save current sp to come back to same position
			handler.sp = vm.sp
			// remove current error if any
			vm.curFrame.errHandlers.err = nil
			// set ip to finally's position
			vm.ip = pos - 1
		case OpUnary:
			err := vm.xOpUnary()
			if err == nil {
				continue
			}
			if err = vm.throwGenErr(err); err != nil {
				vm.err = err
				return
			}
		case OpNoOp:
		default:
			vm.err = fmt.Errorf("unknown opcode %d", vm.curInsts[vm.ip])
			return
		}
	}
	vm.err = ErrVMAborted
}
