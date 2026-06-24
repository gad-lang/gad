package debug

import (
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

// evalCondition compiles `return (cond)` with the current frame's local
// variables bound as globals and runs it in a fresh VM, returning whether the
// result is truthy.
func evalCondition(vm *gad.VM, cond string) (bool, error) {
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

	src := "return (" + cond + ")"
	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		return false, err
	}
	ret, err := gad.NewVM(builtins.Build(), bc).RunOpts(&gad.RunOpts{Globals: globals})
	if err != nil {
		return false, err
	}
	return ret != nil && !ret.IsFalsy(), nil
}
