// Package debug provides a breakpoint/stepping debugger engine for the Gad VM.
//
// An Engine implements gad.DebugStepper: attach it with vm.SetDebugger(engine)
// and run the VM in its own goroutine. The engine pauses execution at
// breakpoints and step boundaries, publishing StopEvents on Stops(); a
// controller (a DAP server, a CLI or the web UI) consumes those events,
// inspects state via the engine's accessors, and resumes with Continue /
// StepInto / StepOver / StepOut.
//
// The engine drives the generated debug loop (vm_loop_debug.go); the production
// VM loop is unaffected.
package debug

import (
	"sync"
	"sync/atomic"

	gad "github.com/gad-lang/gad"
)

// StopReason explains why execution paused.
type StopReason string

const (
	StopEntry      StopReason = "entry"
	StopBreakpoint StopReason = "breakpoint"
	StopStep       StopReason = "step"
	StopPause      StopReason = "pause"
)

// StopEvent is published when execution pauses.
type StopEvent struct {
	Reason StopReason
	File   string
	Line   int
	Column int
}

// command is a resume directive sent from the controller to the paused VM.
type command int

const (
	cmdContinue command = iota
	cmdStepInto
	cmdStepOver
	cmdStepOut
)

// Frame is a snapshot of a call frame for the stack view.
type Frame struct {
	FuncName string
	File     string
	Line     int
	Column   int
	// Locals holds this frame's local variables.
	Locals []Variable
}

// Variable is a named local value snapshot.
type Variable struct {
	Name  string
	Type  string
	Value string
}

// Engine is a gad.DebugStepper implementing breakpoints and stepping.
type Engine struct {
	mu          sync.Mutex
	breakpoints map[int]struct{} // source lines (1-based)
	cmd         command          // active resume directive
	refDepth    int              // frame depth captured at the last stop
	stopOnEntry bool

	pause atomic.Bool

	stops  chan StopEvent
	resume chan command

	// Step-goroutine-only state.
	started  bool
	lastLine int

	// vm is captured on each Step so the controller can inspect state while the
	// VM is parked. Guarded by mu.
	vm *gad.VM
}

// New creates an Engine. When stopOnEntry is true, execution pauses before the
// first instruction.
func New(stopOnEntry bool) *Engine {
	return &Engine{
		breakpoints: map[int]struct{}{},
		cmd:         cmdContinue,
		stopOnEntry: stopOnEntry,
		stops:       make(chan StopEvent),
		resume:      make(chan command),
	}
}

// Stops returns the channel of StopEvents. The VM is parked while an event is
// pending; resume it with Continue/StepInto/StepOver/StepOut.
func (e *Engine) Stops() <-chan StopEvent { return e.stops }

// SetBreakpoints replaces the breakpoint set with the given source lines and
// returns the lines that were accepted (all, here — lines are not validated
// against the source map).
func (e *Engine) SetBreakpoints(lines []int) []int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.breakpoints = make(map[int]struct{}, len(lines))
	for _, l := range lines {
		e.breakpoints[l] = struct{}{}
	}
	return lines
}

// Continue resumes until the next breakpoint or pause.
func (e *Engine) Continue() { e.resume <- cmdContinue }

// StepInto resumes until the next source line (entering calls).
func (e *Engine) StepInto() { e.resume <- cmdStepInto }

// StepOver resumes until the next source line at the current depth or shallower.
func (e *Engine) StepOver() { e.resume <- cmdStepOver }

// StepOut resumes until control returns to a shallower frame.
func (e *Engine) StepOut() { e.resume <- cmdStepOut }

// Pause requests a stop at the next instruction.
func (e *Engine) Pause() { e.pause.Store(true) }

// Step implements gad.DebugStepper. It is called before each instruction in the
// debug loop and blocks while the VM is paused.
func (e *Engine) Step(vm *gad.VM) {
	pos := vm.DebugSourcePos()
	line := pos.Line
	depth := len(vm.DebugFrames())

	e.mu.Lock()
	e.vm = vm
	cmd := e.cmd
	refDepth := e.refDepth
	_, isBp := e.breakpoints[line]
	entry := e.stopOnEntry && !e.started
	e.mu.Unlock()

	sameSpot := line == e.lastLine
	stop := false
	reason := StopStep

	switch {
	case e.pause.Load():
		stop, reason = true, StopPause
	case entry:
		stop, reason = true, StopEntry
	case isBp && line > 0 && !sameSpot:
		stop, reason = true, StopBreakpoint
	default:
		switch cmd {
		case cmdContinue:
			// Only breakpoints / pause stop, handled above.
		case cmdStepInto:
			stop = line > 0 && !sameSpot
		case cmdStepOver:
			stop = line > 0 && !sameSpot && depth <= refDepth
		case cmdStepOut:
			stop = line > 0 && depth < refDepth
		}
	}

	if line > 0 {
		e.lastLine = line
	}
	if !stop {
		return
	}

	// Park: record position, clear one-shot flags, hand the event to the
	// controller and block until it resumes us.
	e.mu.Lock()
	e.started = true
	e.pause.Store(false)
	e.mu.Unlock()

	e.stops <- StopEvent{Reason: reason, File: pos.FileName(), Line: line, Column: pos.Column}
	c := <-e.resume

	e.mu.Lock()
	e.cmd = c
	e.refDepth = depth
	e.mu.Unlock()
}

// Frames returns the current call stack (innermost last). Valid while parked.
func (e *Engine) Frames() []Frame {
	e.mu.Lock()
	vm := e.vm
	e.mu.Unlock()
	if vm == nil {
		return nil
	}
	df := vm.DebugFrames()
	out := make([]Frame, len(df))
	for i, f := range df {
		out[i] = Frame{
			FuncName: f.FuncName,
			File:     f.Pos.FileName(),
			Line:     f.Pos.Line,
			Column:   f.Pos.Column,
			Locals:   variablesOf(f.Locals, f.LocalNames),
		}
	}
	return out
}

// variablesOf builds named Variable snapshots from raw local values and names.
func variablesOf(objs []gad.Object, names []string) []Variable {
	out := make([]Variable, len(objs))
	for i, o := range objs {
		out[i] = Variable{Name: localName(names, i), Type: objectType(o), Value: objectString(o)}
	}
	return out
}

// Locals returns the current frame's local variables. Valid while parked.
func (e *Engine) Locals() []Variable {
	e.mu.Lock()
	vm := e.vm
	e.mu.Unlock()
	if vm == nil {
		return nil
	}
	objs := vm.DebugLocals()
	names := vm.DebugLocalNames()
	out := make([]Variable, len(objs))
	for i, o := range objs {
		out[i] = Variable{
			Name:  localName(names, i),
			Type:  objectType(o),
			Value: objectString(o),
		}
	}
	return out
}

// localName returns the debug name for slot i, falling back to "local<i>".
func localName(names []string, i int) string {
	if i < len(names) && names[i] != "" {
		return names[i]
	}
	return "local" + itoa(i)
}

func objectType(o gad.Object) string {
	if o == nil || o == gad.Nil {
		return "nil"
	}
	return o.Type().Name()
}

func objectString(o gad.Object) string {
	if o == nil {
		return "nil"
	}
	return o.ToString()
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
