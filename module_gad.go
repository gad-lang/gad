package gad

// gadModuleSpec is the spec for the global `gad` namespace.
var gadModuleSpec = NewModuleSpecFromName("gad")

// GadModule returns the `gad` builtin namespace (the operator functions, the
// parse/eval meta-functions and the StmtsObject/StmtObject AST value types).
func GadModule() Dict {
	return Dict{
		"binOp":            BuiltinObjects[BuiltinBinaryOperator],
		"selfAssignOp":     BuiltinObjects[BuiltinSelfAssignOperator],
		"unOp":             BuiltinObjects[BuiltinUnaryOperator],
		"enter":            BuiltinObjects[BuiltinEnter],
		"exit":             BuiltinObjects[BuiltinExit],
		"methodFromArgs":   BuiltinObjects[BuiltinMethodFromArgs],
		"parse":            gadParseFn,
		"parseFile":        gadParseFileFn,
		"eval":             gadEvalFn,
		"invoker":          gadInvokerFn,
		"quote":            gadQuoteFn,
		"unquote":          gadUnquoteFn,
		"StmtsObject":      StmtsType,
		"StmtObject":       StmtType,
		"SourceType":       SourceTypeEnum,
		"SourceFileObject": SourceFileType,
		"Env":              EnvType,
		// Non-literal meta/structural type objects (no source literal of their
		// own), exposed for reflection and `::` type checks — e.g. `x :: gad.Class`.
		"Class":           TClass,
		"Enum":            TEnum,
		"Interface":       TInterface,
		"MethodInterface": TMethodInterface,
		"Module":          TModule,
		"ModuleSpec":      TStaticModule,
		"Symbol":          TSymbol,
	}
}

// registerGadModule registers `gad` as a global namespace whose members
// `binOp` / `selfAssignOp` resolve to the existing operator builtins. The
// qualified names map to the same builtin enums used by the VM's operator
// dispatch, so `gad.binOp(...)` and `met gad.binOp(...)` share identity with
// it.
func registerGadModule() {
	name := gadModuleSpec.Name
	// Build gad.parse / gad.parseFile / gad.eval now (init time), when the
	// builtin type values they reference are all initialized.
	buildGadNamespaceFuncs()
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

	// Meta-functions and AST value types exposed in the `gad` namespace.
	BuiltinObjects[BuiltinParse] = gadParseFn
	BuiltinObjects[BuiltinParseFile] = gadParseFileFn
	BuiltinObjects[BuiltinEval] = gadEvalFn
	BuiltinsMap[name+".parse"] = BuiltinParse
	BuiltinsMap[name+".parseFile"] = BuiltinParseFile
	BuiltinsMap[name+".eval"] = BuiltinEval
	BuiltinObjects[BuiltinSourceType] = SourceTypeEnum
	BuiltinsMap[name+".StmtsObject"] = BuiltinStmts
	BuiltinsMap[name+".StmtObject"] = BuiltinStmt
	BuiltinsMap[name+".SourceType"] = BuiltinSourceType
	BuiltinsMap[name+".SourceFileObject"] = BuiltinSourceFile
	BuiltinsMap[name+".Env"] = BuiltinEnvType

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
