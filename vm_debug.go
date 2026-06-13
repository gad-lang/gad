package gad

//go:generate go run ./cmd/update-delve gen

import (
	"github.com/gad-lang/gad/parser/source"
)

// DebugStepper is invoked before each instruction executes when the VM runs in
// debug mode (see VM.SetDebugger). Implementations typically block inside Step
// to pause execution (breakpoints, single-stepping) and inspect state through
// the VM's Debug* accessors.
//
// The debug execution loop (loopDebug) is generated from the production loop by
// cmd/update-delve, so the two never drift; the production loop has no
// per-instruction hook and is unaffected.
type DebugStepper interface {
	// Step is called with the VM positioned at the instruction about to run.
	Step(vm *VM)
}

// SetDebugger attaches d, switching the VM to its debug execution loop on the
// next run. Pass nil to detach. Must be set before Run/RunOpts.
func (vm *VM) SetDebugger(d DebugStepper) { vm.dbg = d }

// Debugger returns the attached DebugStepper, or nil.
func (vm *VM) Debugger() DebugStepper { return vm.dbg }

// DebugIP returns the index of the instruction about to execute.
func (vm *VM) DebugIP() int { return vm.ip }

// DebugOpcode returns the opcode about to execute.
func (vm *VM) DebugOpcode() Opcode {
	if vm.ip < 0 || vm.ip >= len(vm.curInsts) {
		return OpNoOp
	}
	return Opcode(vm.curInsts[vm.ip])
}

// DebugSourcePos returns the source position (file/line/column) of the
// instruction about to execute, or a zero FilePos when unavailable.
func (vm *VM) DebugSourcePos() source.FilePos {
	pos := vm.getSourcePos()
	if vm.bytecode == nil || vm.bytecode.FileSet == nil || !pos.IsValid() {
		return source.FilePos{}
	}
	return vm.bytecode.FileSet.Position(pos)
}

// DebugFrame describes one active call frame for the debugger's stack view.
type DebugFrame struct {
	FuncName string
	Pos      source.FilePos
}

// DebugFrames returns the active call frames from outermost to innermost (the
// current frame is last).
func (vm *VM) DebugFrames() []DebugFrame {
	if vm.bytecode == nil {
		return nil
	}
	fs := vm.bytecode.FileSet
	out := make([]DebugFrame, 0, vm.frameIndex+1)
	for i := 0; i <= vm.frameIndex && i < len(vm.frames); i++ {
		f := &vm.frames[i]
		if f.fn == nil {
			continue
		}
		ip := f.ip
		if i == vm.frameIndex {
			ip = vm.ip
		}
		var pos source.FilePos
		if fs != nil {
			if p := f.fn.SourcePos(ip); p.IsValid() {
				pos = fs.Position(p)
			}
		}
		out = append(out, DebugFrame{FuncName: f.fn.Name(), Pos: pos})
	}
	return out
}

// DebugLocals returns the local variable values of the current frame
// (dereferencing captured pointers).
func (vm *VM) DebugLocals() []Object {
	if vm.curFrame == nil || vm.curFrame.fn == nil {
		return nil
	}
	base := vm.curFrame.basePointer
	n := vm.curFrame.fn.NumLocals
	out := make([]Object, 0, n)
	for i := 0; i < n; i++ {
		idx := base + i
		if idx < 0 || idx >= len(vm.stack) {
			break
		}
		v := vm.stack[idx]
		if p, ok := v.(*ObjectPtr); ok && p.Value != nil {
			v = *p.Value
		}
		out = append(out, v)
	}
	return out
}

// DebugAbort requests the running VM to stop (equivalent to Abort).
func (vm *VM) DebugAbort() { vm.Abort() }
