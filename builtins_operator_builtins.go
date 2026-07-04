package gad

import "github.com/gad-lang/gad/token"

// Per-operator builtin routing: operator token -> the BuiltinType of the
// gad.binOp{Op} / gad.unOp{Op} / gad.selfAssignOp{Op} builtin. The VM encodes
// operator tokens in a single instruction byte, so they index a fixed array.
var (
	binaryOpBuiltinByTok     [256]BuiltinType
	unaryOpBuiltinByTok      [256]BuiltinType
	selfAssignOpBuiltinByTok [256]BuiltinType
)

// binaryOpTokens are the binary operators exposed as gad.binOp{Op} builtins.
var binaryOpTokens = []token.Token{
	token.Add, token.Sub, token.Mul, token.Pow, token.Quo, token.Rem,
	token.And, token.Or, token.Xor, token.Shl, token.Shr, token.AndNot,
	token.LAnd, token.Equal, token.NotEqual, token.Less, token.Greater,
	token.LessEq, token.GreaterEq, token.Tilde, token.DoubleTilde,
	token.TripleTilde, token.DotDot, token.TripleLess, token.TripleGreater,
	token.DoubleMod, token.Lambda, token.Same, token.NotSame, token.Inc,
	token.Dec, token.In, token.Ain,
}

// selfAssignOpTokens are the self-assign operators exposed as
// gad.selfAssignOp{Op} builtins.
var selfAssignOpTokens = []token.Token{
	token.Add, token.Inc, token.Sub, token.Dec, token.Mul, token.Pow,
	token.Quo, token.Rem, token.And, token.Or, token.Xor, token.Shl,
	token.Shr, token.TripleLess, token.TripleGreater, token.DoubleMod,
	token.AndNot, token.LOr,
}

// unaryOpTokens are the unary operators exposed as gad.unOp{Op} builtins.
var unaryOpTokens = []token.Token{
	token.Not, token.Sub, token.Add, token.Xor, token.Inc, token.Dec,
}

// registerOperatorBuiltins creates the per-operator gad.binOp{Op} /
// gad.unOp{Op} / gad.selfAssignOp{Op} builtins and their token->builtin
// routing. Each is a method-bearing builtin whose default is the op-bound
// operator handler, so a `met gad.binOp{Op}(left, right)` overload adds a typed
// method reached by the VM's per-operator dispatch (see VM.callBinaryOp). Called
// from registerGadModule once the gad module namespace exists.
func registerOperatorBuiltins() {
	// The per-operator builtins are keyed by the static enum groups using the
	// i-th token -> i-th key after the group's Begin marker, so the token slices
	// must stay the same length as their enum ranges (both come from the same
	// operator lists via cmd/mkoptypes). Fail fast if they drift out of sync.
	if len(binaryOpTokens) != int(GroupBuiltinBinaryOperatorsEnd-GroupBuiltinBinaryOperatorsBegin-1) ||
		len(selfAssignOpTokens) != int(GroupBuiltinSelfAssignOperatorsEnd-GroupBuiltinSelfAssignOperatorsBegin-1) ||
		len(unaryOpTokens) != int(GroupBuiltinUnaryOperatorsEnd-GroupBuiltinUnaryOperatorsBegin-1) {
		panic("operator token slice length does not match its Builtin operator enum group")
	}

	mod, _ := BuiltinObjects[BuiltinModuleGad].(Dict)

	// add registers a method-bearing builtin `gad.<name>` at the static builtin
	// key bt with the given default handler. The per-operator builtins use the
	// static Builtin{Binary,Unary,SelfAssign}Operator{Op} keys (all < BuiltinEnd_)
	// so Builtins.build() clones them per VM: a `met gad.binOp{Op}(…)` overload
	// mutates an isolated method table instead of leaking into the global builtin.
	add := func(bt BuiltinType, name string, def *BuiltinFunction) {
		fn := NewBuiltinFunctionWithMethods(name, gadModuleSpec)
		fn.defaul = def
		BuiltinObjects[bt] = fn
		BuiltinsMap["gad."+name] = bt
		if mod != nil {
			mod[name] = fn
		}
	}

	// The token slices are generated in the same order as their enum groups, so
	// the i-th token maps to the i-th static key after the group's Begin marker.
	for i, tok := range binaryOpTokens {
		bt := GroupBuiltinBinaryOperatorsBegin + 1 + BuiltinType(i)
		name := "binOp" + tok.Name()
		def := NewBuiltinFunction(name, binaryOpHandler(BinaryOperatorType(tok))).
			WithModule(gadModuleSpec).WithParamsPairs("left", TAny, "right", TAny)
		add(bt, name, def)
		binaryOpBuiltinByTok[byte(tok)] = bt
	}
	for i, tok := range selfAssignOpTokens {
		bt := GroupBuiltinSelfAssignOperatorsBegin + 1 + BuiltinType(i)
		name := "selfAssignOp" + tok.Name()
		def := NewBuiltinFunction(name, selfAssignOpHandler(SelfAssignOperatorType(tok))).
			WithModule(gadModuleSpec).WithParamsPairs("left", TAny, "right", TAny)
		add(bt, name, def)
		selfAssignOpBuiltinByTok[byte(tok)] = bt
	}
	for i, tok := range unaryOpTokens {
		bt := GroupBuiltinUnaryOperatorsBegin + 1 + BuiltinType(i)
		name := "unOp" + tok.Name()
		def := NewBuiltinFunction(name, unaryOpHandler(UnaryOperatorType(tok))).
			WithModule(gadModuleSpec).WithParamsPairs("operand", TAny)
		add(bt, name, def)
		unaryOpBuiltinByTok[byte(tok)] = bt
	}
}

// callerMethoded reports whether an operator builtin carries user-defined
// `met gad.…Op{Op}(…)` overloads.
type callerMethoded interface{ HasCallerMethods() bool }

// hasOpMethods reports whether the operator builtin bt has a user overload.
// Without one, operators dispatch natively (binOpObject / selfAssignOpObject)
// instead of allocating an Args + Call and walking the method-dispatch tree.
func (vm *VM) hasOpMethods(bt BuiltinType) bool {
	m := vm.Builtins.OpMethoded(bt)
	return m != nil && m.HasCallerMethods()
}

// callBinaryOp dispatches a binary operator to its gad.binOp{Op} builtin.
func (vm *VM) callBinaryOp(tok token.Token, left, right Object) (Object, error) {
	bt := binaryOpBuiltinByTok[byte(tok)]
	if !vm.hasOpMethods(bt) {
		return BinaryOp(vm, tok, left, right)
	}
	return vm.Builtins.Call(bt, Call{VM: vm, Args: Args{Array{left, right}}})
}

// callSelfAssignOp dispatches a self-assign operator to its
// gad.selfAssignOp{Op} builtin.
func (vm *VM) callSelfAssignOp(tok token.Token, left, right Object) (Object, error) {
	bt := selfAssignOpBuiltinByTok[byte(tok)]
	if !vm.hasOpMethods(bt) {
		return selfAssignOpDispatch(vm, SelfAssignOperatorType(tok), left, right)
	}
	return vm.Builtins.Call(bt, Call{VM: vm, Args: Args{Array{left, right}}})
}

// callUnaryOp dispatches a unary operator to its gad.unOp{Op} builtin.
func (vm *VM) callUnaryOp(tok token.Token, operand Object) (Object, error) {
	return vm.Builtins.Call(unaryOpBuiltinByTok[byte(tok)],
		Call{VM: vm, Args: Args{Array{operand}}})
}
