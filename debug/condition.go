package debug

import (
	"errors"

	gad "github.com/gad-lang/gad"
)

// conditionMet reports whether breakpoint bp should pause at the current
// instruction. An empty condition always matches. A condition expression is
// evaluated against the paused frame's locals (exposed as globals); the
// breakpoint pauses when the result is truthy (`!value.IsFalsy()`). A condition
// that fails to compile or run is treated as a match, so the breakpoint still
// fires and the problem surfaces rather than being silently swallowed.
func (e *Engine) conditionMet(vm *gad.VM, bp Breakpoint) bool {
	if bp.Condition == "" {
		return true
	}
	ok, err := evalCondition(vm, bp.Condition)
	if err != nil {
		return true
	}
	return ok
}

// evalCondition reports whether `cond` is truthy in the paused frame's scope.
func evalCondition(vm *gad.VM, cond string) (bool, error) {
	ret, err := evalInFrame(vm, "return ("+cond+")")
	if err != nil {
		return false, err
	}
	return ret != nil && !ret.IsFalsy(), nil
}

// EvalInFrame evaluates expr against the paused frame's locals and returns the
// result rendered with str() (or repr() when repr is set). It is valid only
// while the engine is parked at a stop; otherwise it returns an error.
func (e *Engine) EvalInFrame(expr string, repr bool) (string, error) {
	e.mu.Lock()
	vm := e.vm
	e.mu.Unlock()
	if vm == nil {
		return "", errors.New("no paused frame")
	}
	render := "str"
	if repr {
		render = "repr"
	}
	ret, err := evalInFrame(vm, "return "+render+"("+expr+")")
	if err != nil {
		return "", err
	}
	if ret == nil {
		return "", nil
	}
	return ret.ToString(), nil
}

// EvalObject evaluates expr against the paused frame's locals and returns the
// resulting object (for the tree navigator / inspection). Valid only while
// parked at a stop.
func (e *Engine) EvalObject(expr string) (gad.Object, error) {
	e.mu.Lock()
	vm := e.vm
	e.mu.Unlock()
	if vm == nil {
		return nil, errors.New("no paused frame")
	}
	return evalInFrame(vm, "return ("+expr+")")
}

// evalInFrame compiles src with the current frame's local variables bound as
// globals and runs it in a fresh VM, returning the result object. The locals are
// a snapshot, so the evaluation cannot mutate the debugged program's state.
func evalInFrame(vm *gad.VM, src string) (gad.Object, error) {
	names := vm.DebugLocalNames()
	objs := vm.DebugLocals()

	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)
	globals := gad.Dict{}
	for i, name := range names {
		if name == "" {
			continue
		}
		if _, err := st.DefineGlobal(name); err != nil {
			continue
		}
		if i < len(objs) && objs[i] != nil {
			globals[name] = objs[i]
		} else {
			globals[name] = gad.Nil
		}
	}

	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		return nil, err
	}
	return gad.NewVM(builtins.Build(), bc).RunOpts(&gad.RunOpts{Globals: globals})
}
