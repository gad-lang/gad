package gad_test

import (
	"sync"
	"testing"

	gad "github.com/gad-lang/gad"
)

// recordingStepper records the source line of each executed instruction.
type recordingStepper struct {
	mu    sync.Mutex
	lines []int
	steps int
}

func (r *recordingStepper) Step(vm *gad.VM) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps++
	if pos := vm.DebugSourcePos(); pos.Line > 0 {
		if len(r.lines) == 0 || r.lines[len(r.lines)-1] != pos.Line {
			r.lines = append(r.lines, pos.Line)
		}
	}
}

func compileForDebug(t *testing.T, src string) *gad.Bytecode {
	t.Helper()
	b := gad.NewBuiltins()
	st := gad.NewSymbolTable(b.NameSet)
	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return bc
}

func TestVMDebuggerSteps(t *testing.T) {
	const src = "a := 1\nb := 2\nreturn a + b\n"
	bc := compileForDebug(t, src)

	stepper := &recordingStepper{}
	vm := gad.NewVM(gad.NewBuiltins().Build(), bc)
	vm.SetDebugger(stepper)

	ret, err := vm.RunOpts(&gad.RunOpts{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := ret.(gad.Int); got != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
	if stepper.steps == 0 {
		t.Fatal("debugger Step was never called")
	}
	if len(stepper.lines) == 0 {
		t.Fatal("no source lines recorded")
	}
	// The program touches lines 1..3.
	for _, want := range []int{1, 2, 3} {
		found := false
		for _, l := range stepper.lines {
			if l == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected to step over line %d; got lines %v", want, stepper.lines)
		}
	}
}

func TestVMDebuggerMatchesNormalRun(t *testing.T) {
	const src = `
total := 0
for i := 0; i < 5; i++ {
	total += i
}
return total
`
	want := gad.Int(10)

	// Normal run.
	bc1 := compileForDebug(t, src)
	ret1, err := gad.NewVM(gad.NewBuiltins().Build(), bc1).RunOpts(&gad.RunOpts{})
	if err != nil || ret1.(gad.Int) != want {
		t.Fatalf("normal run: ret=%v err=%v", ret1, err)
	}

	// Debug run must produce the same result.
	bc2 := compileForDebug(t, src)
	vm := gad.NewVM(gad.NewBuiltins().Build(), bc2)
	vm.SetDebugger(&recordingStepper{})
	ret2, err := vm.RunOpts(&gad.RunOpts{})
	if err != nil || ret2.(gad.Int) != want {
		t.Fatalf("debug run: ret=%v err=%v", ret2, err)
	}
}

func TestVMDebuggerFramesAndLocals(t *testing.T) {
	const src = `
f := func(x) {
	y := x * 2
	return y
}
return f(21)
`
	bc := compileForDebug(t, src)

	var maxDepth int
	sawLocal := false
	stepper := stepperFunc(func(vm *gad.VM) {
		if frames := vm.DebugFrames(); len(frames) > maxDepth {
			maxDepth = len(frames)
		}
		for _, l := range vm.DebugLocals() {
			if i, ok := l.(gad.Int); ok && i == 42 {
				sawLocal = true
			}
		}
	})

	vm := gad.NewVM(gad.NewBuiltins().Build(), bc)
	vm.SetDebugger(stepper)
	ret, err := vm.RunOpts(&gad.RunOpts{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ret.(gad.Int) != 42 {
		t.Fatalf("expected 42, got %v", ret)
	}
	if maxDepth < 2 {
		t.Fatalf("expected to enter the function frame (depth >= 2), got %d", maxDepth)
	}
	if !sawLocal {
		t.Fatal("expected to observe local y == 42 inside the function")
	}
}

type stepperFunc func(vm *gad.VM)

func (f stepperFunc) Step(vm *gad.VM) { f(vm) }
