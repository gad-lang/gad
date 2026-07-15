package gad

// gadModuleSpec is the spec for the global `gad` namespace.
var gadModuleSpec = NewModuleSpecFromName("gad")

// GadModule returns the `gad` builtin namespace (the operator functions).
func GadModule() Dict {
	return Dict{
		"binOp":          BuiltinObjects[BuiltinBinaryOperator],
		"selfAssignOp":   BuiltinObjects[BuiltinSelfAssignOperator],
		"unOp":           BuiltinObjects[BuiltinUnaryOperator],
		"enter":          BuiltinObjects[BuiltinEnter],
		"exit":           BuiltinObjects[BuiltinExit],
		"methodFromArgs": BuiltinObjects[BuiltinMethodFromArgs],
	}
}

// registerGadModule registers `gad` as a global namespace whose members
// `binOp` / `selfAssignOp` resolve to the existing operator builtins. The
// qualified names map to the same builtin enums used by the VM's operator
// dispatch, so `gad.binOp(...)` and `met gad.binOp(...)` share identity with
// it.
func registerGadModule() {
	name := gadModuleSpec.Name
	setOperatorModule(BuiltinObjects[BuiltinBinaryOperator])
	setOperatorModule(BuiltinObjects[BuiltinSelfAssignOperator])
	setOperatorModule(BuiltinObjects[BuiltinUnaryOperator])
	setOperatorModule(BuiltinObjects[BuiltinEnter])
	setOperatorModule(BuiltinObjects[BuiltinExit])

	BuiltinsMap[name] = BuiltinModuleGad
	BuiltinObjects[BuiltinModuleGad] = GadModule()
	BuiltinsMap[name+".binOp"] = BuiltinBinaryOperator
	BuiltinsMap[name+".selfAssignOp"] = BuiltinSelfAssignOperator
	BuiltinsMap[name+".unOp"] = BuiltinUnaryOperator
	BuiltinsMap[name+".enter"] = BuiltinEnter
	BuiltinsMap[name+".exit"] = BuiltinExit
	BuiltinsMap[name+".methodFromArgs"] = BuiltinMethodFromArgs

	// Per-operator builtins: gad.binOp{Op} / gad.unOp{Op} / gad.selfAssignOp{Op}.
	registerOperatorBuiltins()
}

// setOperatorModule ties an operator builtin to the core module spec.
func setOperatorModule(o Object) {
	switch m := o.(type) {
	case *BuiltinFunctionWithMethods:
		m.Module = gadModuleSpec
	case *BuiltinFunction:
		m.Module = gadModuleSpec
	}
}
