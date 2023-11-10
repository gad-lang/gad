// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// OptimizerError represents an optimizer error.
type OptimizerError struct {
	FilePos parser.SourceFilePos
	Node    ast.Node
	Err     error
}

func (e *OptimizerError) Error() string {
	return fmt.Sprintf("Optimizer Error: %s\n\tat %s", e.Err.Error(), e.FilePos)
}

func (e *OptimizerError) Unwrap() error {
	return e.Err
}

type optimizerScope struct {
	parent   *optimizerScope
	shadowed []string
}

func (s *optimizerScope) define(ident string) {
	if _, ok := BuiltinsMap[ident]; ok {
		s.shadowed = append(s.shadowed, ident)
	}
}

func (s *optimizerScope) shadowedBuiltins() []string {
	var out []string
	if len(s.shadowed) > 0 {
		out = append(out, s.shadowed...)
	}

	if s.parent != nil {
		out = append(out, s.parent.shadowedBuiltins()...)
	}
	return out
}

// SimpleOptimizer optimizes given parsed file by evaluating constants and
// expressions. It is not safe to call methods concurrently.
type SimpleOptimizer struct {
	scope            *optimizerScope
	vm               *VM
	count            int
	total            int
	maxCycle         int
	indent           int
	optimConsts      bool
	optimExpr        bool
	builtins         map[string]BuiltinType
	disabledBuiltins []string
	constants        []Object
	instructions     []byte
	moduleStore      *moduleStore
	returnStmt       node.ReturnStmt
	file             *parser.File
	errors           multipleErr
	trace            io.Writer
	exprLevel        byte
	evalBits         uint64
}

// NewOptimizer creates an Optimizer object.
func NewOptimizer(
	file *parser.File,
	base *SymbolTable,
	opts CompilerOptions,
) *SimpleOptimizer {
	var disabled []string
	if base != nil {
		disabled = base.DisabledBuiltins()
		disabled = append(disabled, base.ShadowedBuiltins()...)
	}

	var trace io.Writer
	if opts.TraceOptimizer {
		trace = opts.Trace
	}

	var builtins = BuiltinsMap
	if opts.SymbolTable != nil {
		builtins = opts.SymbolTable.builtins
	}

	return &SimpleOptimizer{
		file:             file,
		vm:               NewVM(nil).SetRecover(true),
		maxCycle:         opts.OptimizerMaxCycle,
		optimConsts:      opts.OptimizeConst,
		optimExpr:        opts.OptimizeExpr,
		disabledBuiltins: disabled,
		moduleStore:      newModuleStore(),
		trace:            trace,
		builtins:         builtins,
	}
}

func canOptimizeExpr(expr node.Expr) bool {
	if node.IsStatement(expr) {
		return false
	}

	switch expr.(type) {
	case *node.BoolLit,
		*node.IntLit,
		*node.UintLit,
		*node.FloatLit,
		*node.CharLit,
		*node.StringLit,
		*node.NilLit:
		return false
	}
	return true
}

func canOptimizeInsts(constants []Object, insts []byte) bool {
	if len(insts) == 0 {
		return false
	}

	// using array here instead of map or slice is faster to look up opcode
	allowedOps := [...]bool{
		OpConstant: true, OpNull: true, OpBinaryOp: true, OpUnary: true,
		OpNoOp: true, OpAndJump: true, OpOrJump: true, OpArray: true,
		OpReturn: true, OpEqual: true, OpNotEqual: true, OpPop: true,
		OpGetBuiltin: true, OpCall: true, OpSetLocal: true, OpDefineLocal: true,
		OpTrue: true, OpFalse: true, OpJumpNil: true, OpJumpNotNil: true,
		OpCallee: true, OpArgs: true, OpNamedArgs: true,
		OpStdIn: true, OpStdOut: true, OpStdErr: true,
		OpTextWriter: true,
	}

	allowedBuiltins := [...]bool{
		BuiltinContains: true, BuiltinBool: true, BuiltinInt: true,
		BuiltinUint: true, BuiltinChar: true, BuiltinFloat: true,
		BuiltinString: true, BuiltinChars: true, BuiltinLen: true,
		BuiltinTypeName: true, BuiltinBytes: true, BuiltinError: true,
		BuiltinWrite: true, BuiltinPrint: true, BuiltinSprintf: true,
		BuiltinIsError: true, BuiltinIsInt: true, BuiltinIsUint: true,
		BuiltinIsFloat: true, BuiltinIsChar: true, BuiltinIsBool: true,
		BuiltinIsString: true, BuiltinIsBytes: true, BuiltinIsMap: true,
		BuiltinIsArray: true, BuiltinIsNil: true, BuiltinIsIterable: true,
		BuiltinDecimal: true, BuiltinItems: true, BuiltinValues: true,
		BuiltinKeys: true, BuiltinKeyValue: true, BuiltinKeyValueArray: true,
		BuiltinBuffer: true, BuiltinVMPushWriter: true, BuiltinVMPopWriter: true,
		^byte(0): false,
	}

	canOptimize := true

	IterateInstructions(insts,
		func(_ int, opcode Opcode, operands []int, _ int) bool {
			if !allowedOps[opcode] {
				canOptimize = false
				return false
			}

			if opcode == OpConstant &&
				!isObjectConstant(constants[operands[0]]) {
				canOptimize = false
				return false
			}

			if opcode == OpGetBuiltin &&
				!allowedBuiltins[operands[0]] {
				canOptimize = false
				return false
			}
			return true
		},
	)
	return canOptimize
}

func (so *SimpleOptimizer) evalExpr(expr node.Expr) (node.Expr, bool) {
	if !so.optimExpr {
		return nil, false
	}

	if len(so.errors) > 0 {
		// do not evaluate erroneous line again
		prevPos := so.errors[len(so.errors)-1].(*OptimizerError).FilePos
		if so.file.InputFile.Set().Position(expr.Pos()).Line == prevPos.Line {
			return nil, false
		}
	}

	if so.trace != nil {
		so.printTraceMsgf("eval: %s", expr)
	}

	if !so.canEval() || !canOptimizeExpr(expr) {
		if so.trace != nil {
			so.printTraceMsgf("cannot optimize expression")
		}
		return nil, false
	}

	x, ok := so.slowEvalExpr(expr)
	if !ok {
		so.setNoEval()
		if so.trace != nil {
			so.printTraceMsgf("cannot optimize code")
		}
	} else {
		so.count++
	}
	return x, ok
}

func (so *SimpleOptimizer) slowEvalExpr(expr node.Expr) (node.Expr, bool) {
	st := NewSymbolTable(so.builtins).
		EnableParams(false).
		DisableBuiltin(so.disabledBuiltins...).
		DisableBuiltin(so.scope.shadowedBuiltins()...)

	compiler := NewCompiler(
		so.file.InputFile,
		CompilerOptions{
			SymbolTable: st,
			moduleStore: so.moduleStore.reset(),
			Constants:   so.constants[:0],
			Trace:       so.trace,
		},
	)
	compiler.instructions = so.instructions[:0]
	compiler.indent = so.indent

	so.returnStmt.Result = expr

	if err := compiler.Compile(&so.returnStmt); err != nil {
		return nil, false
	}

	bytecode := compiler.Bytecode()

	// obtain constants and instructions slices to reuse
	so.constants = bytecode.Constants
	so.instructions = bytecode.Main.Instructions

	if !canOptimizeInsts(bytecode.Constants, bytecode.Main.Instructions) {
		if so.trace != nil {
			so.printTraceMsgf("cannot optimize instructions")
		}
		return nil, false
	}

	obj, err := so.vm.SetBytecode(bytecode).Clear().Run(nil)
	if err != nil {
		if so.trace != nil {
			so.printTraceMsgf("eval error: %s", err)
		}
		if !errors.Is(err, ErrVMAborted) {
			so.errors = append(so.errors, so.error(expr, err))
		}
		obj = nil
	}

	switch v := obj.(type) {
	case String:
		l := strconv.Quote(string(v))
		expr = &node.StringLit{
			Value:    string(v),
			Literal:  l,
			ValuePos: expr.Pos(),
		}
	case *NilType:
		expr = &node.NilLit{TokenPos: expr.Pos()}
	case Bool:
		l := strconv.FormatBool(bool(v))
		expr = &node.BoolLit{
			Value:    bool(v),
			Literal:  l,
			ValuePos: expr.Pos(),
		}
	case Int:
		l := strconv.FormatInt(int64(v), 10)
		expr = &node.IntLit{
			Value:    int64(v),
			Literal:  l,
			ValuePos: expr.Pos(),
		}
	case Uint:
		l := strconv.FormatUint(uint64(v), 10)
		expr = &node.UintLit{
			Value:    uint64(v),
			Literal:  l,
			ValuePos: expr.Pos(),
		}
	case Float:
		l := strconv.FormatFloat(float64(v), 'f', -1, 64)
		expr = &node.FloatLit{
			Value:    float64(v),
			Literal:  l,
			ValuePos: expr.Pos(),
		}
	case Char:
		l := strconv.QuoteRune(rune(v))
		expr = &node.CharLit{
			Value:    rune(v),
			Literal:  l,
			ValuePos: expr.Pos(),
		}
	default:
		return nil, false
	}
	return expr, true
}

func (so *SimpleOptimizer) canEval() bool {
	// if left bits are set, we should not eval, pointless
	return so.evalBits>>so.exprLevel == 0
}

func (so *SimpleOptimizer) setNoEval() {
	// set level bit to 1, we got an eval error
	so.evalBits |= 1 << (so.exprLevel - 1)
}

func (so *SimpleOptimizer) enterExprLevel() {
	// clear bits on the left
	shift := 64 - so.exprLevel
	so.evalBits = so.evalBits << shift >> shift
	so.exprLevel++
	// if opt.trace != nil {
	// 	opt.printTraceMsgf(fmt.Sprintf("level:%d %064b", opt.exprLevel, opt.evalBits))
	// }
}

func (so *SimpleOptimizer) leaveExprLevel() {
	// if opt.trace != nil {
	// 	opt.printTraceMsgf(fmt.Sprintf("level:%d %064b", opt.exprLevel, opt.evalBits))
	// }
	so.exprLevel--
}

// Optimize optimizes ast tree by simple constant folding and evaluating simple expressions.
func (so *SimpleOptimizer) Optimize() error {
	so.errors = nil

	defer so.vm.Abort()

	if so.trace != nil {
		so.printTraceMsgf("Enter Optimizer")
	}

	for i := 1; i <= so.maxCycle; i++ {
		so.count = 0
		so.exprLevel = 0
		if so.trace != nil {
			so.printTraceMsgf("%d. pass", i)
		}
		so.enterScope()
		so.optimize(so.file)
		so.leaveScope()

		if so.count == 0 {
			break
		}

		if len(so.errors) > 2 { // bailout
			break
		}
		so.total += so.count
	}

	if so.trace != nil {
		if so.total > 0 {
			so.printTraceMsgf("Total: %d", so.total)
		} else {
			so.printTraceMsgf("No Optimization")
		}
		so.printTraceMsgf("File: %s", so.file)
		so.printTraceMsgf("Exit Optimizer")
		so.printTraceMsgf("----------------------")
	}

	if so.errors == nil {
		return nil
	}
	return so.errors
}

func (so *SimpleOptimizer) binaryopInts(
	op token.Token,
	left, right *node.IntLit,
) (node.Expr, bool) {

	var val int64
	switch op {
	case token.Add:
		val = left.Value + right.Value
	case token.Sub:
		val = left.Value - right.Value
	case token.Mul:
		val = left.Value * right.Value
	case token.Quo:
		if right.Value == 0 {
			return nil, false
		}
		val = left.Value / right.Value
	case token.Rem:
		val = left.Value % right.Value
	case token.And:
		val = left.Value & right.Value
	case token.Or:
		val = left.Value | right.Value
	case token.Shl:
		val = left.Value << right.Value
	case token.Shr:
		val = left.Value >> right.Value
	case token.AndNot:
		val = left.Value &^ right.Value
	default:
		return nil, false
	}
	l := strconv.FormatInt(val, 10)
	return &node.IntLit{Value: val, Literal: l, ValuePos: left.ValuePos}, true
}

func (so *SimpleOptimizer) binaryopFloats(
	op token.Token,
	left, right *node.FloatLit,
) (node.Expr, bool) {

	var val float64
	switch op {
	case token.Add:
		val = left.Value + right.Value
	case token.Sub:
		val = left.Value - right.Value
	case token.Mul:
		val = left.Value * right.Value
	case token.Quo:
		if right.Value == 0 {
			return nil, false
		}
		val = left.Value / right.Value
	default:
		return nil, false
	}

	return &node.FloatLit{
		Value:    val,
		Literal:  strconv.FormatFloat(val, 'f', -1, 64),
		ValuePos: left.ValuePos,
	}, true
}

func (so *SimpleOptimizer) binaryop(
	op token.Token,
	left, right node.Expr,
) (node.Expr, bool) {

	if !so.optimConsts {
		return nil, false
	}

	switch left := left.(type) {
	case *node.IntLit:
		if right, ok := right.(*node.IntLit); ok {
			return so.binaryopInts(op, left, right)
		}
	case *node.FloatLit:
		if right, ok := right.(*node.FloatLit); ok {
			return so.binaryopFloats(op, left, right)
		}
	case *node.StringLit:
		right, ok := right.(*node.StringLit)
		if ok && op == token.Add {
			v := left.Value + right.Value
			return &node.StringLit{
				Value:    v,
				Literal:  strconv.Quote(v),
				ValuePos: left.ValuePos,
			}, true
		}
	}
	return nil, false
}

func (so *SimpleOptimizer) unaryop(
	op token.Token,
	expr node.Expr,
) (node.Expr, bool) {

	if !so.optimConsts {
		return nil, false
	}

	switch expr := expr.(type) {
	case *node.IntLit:
		switch op {
		case token.Not:
			v := expr.Value == 0
			return &node.BoolLit{
				Value:    v,
				Literal:  strconv.FormatBool(v),
				ValuePos: expr.ValuePos,
			}, true
		case token.Sub:
			v := -expr.Value
			l := strconv.FormatInt(v, 10)
			return &node.IntLit{
				Value:    v,
				Literal:  l,
				ValuePos: expr.ValuePos,
			}, true
		case token.Xor:
			v := ^expr.Value
			l := strconv.FormatInt(v, 10)
			return &node.IntLit{
				Value:    v,
				Literal:  l,
				ValuePos: expr.ValuePos,
			}, true
		}
	case *node.UintLit:
		switch op {
		case token.Not:
			v := expr.Value == 0
			return &node.BoolLit{
				Value:    v,
				Literal:  strconv.FormatBool(v),
				ValuePos: expr.ValuePos,
			}, true
		case token.Sub:
			v := -expr.Value
			l := strconv.FormatUint(v, 10)
			return &node.UintLit{
				Value:    v,
				Literal:  l,
				ValuePos: expr.ValuePos,
			}, true
		case token.Xor:
			v := ^expr.Value
			l := strconv.FormatUint(v, 10)
			return &node.UintLit{
				Value:    v,
				Literal:  l,
				ValuePos: expr.ValuePos,
			}, true
		}
	case *node.FloatLit:
		switch op {
		case token.Sub:
			v := -expr.Value
			l := strconv.FormatFloat(v, 'f', -1, 64)
			return &node.FloatLit{
				Value:    v,
				Literal:  l,
				ValuePos: expr.ValuePos,
			}, true
		}
	}
	return nil, false
}

func (so *SimpleOptimizer) optimize(nd ast.Node) (node.Expr, bool) {
	if so.trace != nil {
		if nd != nil {
			defer untraceoptim(traceoptim(so, fmt.Sprintf("%s (%s)",
				nd.String(), reflect.TypeOf(nd).Elem().Name())))
		} else {
			defer untraceoptim(traceoptim(so, "<nil>"))
		}
	}

	if !node.IsStatement(nd) {
		so.enterExprLevel()
		defer so.leaveExprLevel()
	}

	var (
		expr node.Expr
		ok   bool
	)

	switch nd := nd.(type) {
	case *parser.File:
		for _, stmt := range nd.Stmts {
			_, _ = so.optimize(stmt)
		}
	case *node.ExprStmt:
		if nd.Expr != nil {
			if expr, ok = so.optimize(nd.Expr); ok {
				nd.Expr = expr
			}
			if expr, ok = so.evalExpr(nd.Expr); ok {
				nd.Expr = expr
			}
		}
	case *node.ParenExpr:
		if nd.Expr != nil {
			return so.optimize(nd.Expr)
		}
	case *node.BinaryExpr:
		if expr, ok = so.optimize(nd.LHS); ok {
			nd.LHS = expr
		}
		if expr, ok = so.optimize(nd.RHS); ok {
			nd.RHS = expr
		}
		if expr, ok = so.binaryop(nd.Token, nd.LHS, nd.RHS); ok {
			so.count++
			return expr, ok
		}
		return so.evalExpr(nd)
	case *node.UnaryExpr:
		if expr, ok = so.optimize(nd.Expr); ok {
			nd.Expr = expr
		}
		if expr, ok = so.unaryop(nd.Token, nd.Expr); ok {
			so.count++
			return expr, ok
		}
		return so.evalExpr(nd)
	case *node.IfStmt:
		if nd.Init != nil {
			_, _ = so.optimize(nd.Init)
		}
		if expr, ok = so.optimize(nd.Cond); ok {
			nd.Cond = expr
		}
		if expr, ok = so.evalExpr(nd.Cond); ok {
			nd.Cond = expr
		}
		if falsy, ok := isLitFalsy(nd.Cond); ok {
			// convert expression to BoolLit so that Compiler skips if block
			nd.Cond = &node.BoolLit{
				Value:    !falsy,
				Literal:  strconv.FormatBool(!falsy),
				ValuePos: nd.Cond.Pos(),
			}
		}
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
		if nd.Else != nil {
			_, _ = so.optimize(nd.Else)
		}
	case *node.TryStmt:
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
		if nd.Catch != nil {
			_, _ = so.optimize(nd.Catch)
		}
		if nd.Finally != nil {
			_, _ = so.optimize(nd.Finally)
		}
	case *node.CatchStmt:
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
	case *node.FinallyStmt:
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
	case *node.ThrowStmt:
		if nd.Expr != nil {
			if expr, ok = so.optimize(nd.Expr); ok {
				nd.Expr = expr
			}
			if expr, ok = so.evalExpr(nd.Expr); ok {
				nd.Expr = expr
			}
		}
	case *node.ForStmt:
		if nd.Init != nil {
			_, _ = so.optimize(nd.Init)
		}
		if nd.Cond != nil {
			if expr, ok = so.optimize(nd.Cond); ok {
				nd.Cond = expr
			}
		}
		if nd.Post != nil {
			_, _ = so.optimize(nd.Post)
		}
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
	case *node.ForInStmt:
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
		if nd.Else != nil {
			_, _ = so.optimize(nd.Else)
		}
	case *node.BlockStmt:
		for _, stmt := range nd.Stmts {
			_, _ = so.optimize(stmt)
		}
	case *node.AssignStmt:
		for _, lhs := range nd.LHS {
			if ident, ok := lhs.(*node.Ident); ok {
				so.scope.define(ident.Name)
			}
		}
		for i, rhs := range nd.RHS {
			if expr, ok = so.optimize(rhs); ok {
				nd.RHS[i] = expr
			}
		}
		for i, rhs := range nd.RHS {
			if expr, ok = so.evalExpr(rhs); ok {
				nd.RHS[i] = expr
			}
		}
	case *node.DeclStmt:
		decl := nd.Decl.(*node.GenDecl)
		switch decl.Tok {
		case token.Param, token.Global:
			for _, sp := range decl.Specs {
				spec := sp.(*node.ParamSpec)
				so.scope.define(spec.Ident.Name)
			}
		case token.Var, token.Const:
			for _, sp := range decl.Specs {
				spec := sp.(*node.ValueSpec)
				for i := range spec.Idents {
					so.scope.define(spec.Idents[i].Name)
					if i < len(spec.Values) && spec.Values[i] != nil {
						v := spec.Values[i]
						if expr, ok = so.optimize(v); ok {
							spec.Values[i] = expr
							v = expr
						}
						if expr, ok = so.evalExpr(v); ok {
							spec.Values[i] = expr
						}
					}
				}
			}
		}
	case *node.ArrayLit:
		for i := range nd.Elements {
			if expr, ok = so.optimize(nd.Elements[i]); ok {
				nd.Elements[i] = expr
			}
			if expr, ok = so.evalExpr(nd.Elements[i]); ok {
				nd.Elements[i] = expr
			}
		}
	case *node.MapLit:
		for i := range nd.Elements {
			if expr, ok = so.optimize(nd.Elements[i].Value); ok {
				nd.Elements[i].Value = expr
			}
			if expr, ok = so.evalExpr(nd.Elements[i].Value); ok {
				nd.Elements[i].Value = expr
			}
		}
	case *node.IndexExpr:
		if expr, ok = so.optimize(nd.Index); ok {
			nd.Index = expr
		}
		if expr, ok = so.evalExpr(nd.Index); ok {
			nd.Index = expr
		}
	case *node.SliceExpr:
		if nd.Low != nil {
			if expr, ok = so.optimize(nd.Low); ok {
				nd.Low = expr
			}
			if expr, ok = so.evalExpr(nd.Low); ok {
				nd.Low = expr
			}
		}
		if nd.High != nil {
			if expr, ok = so.optimize(nd.High); ok {
				nd.High = expr
			}
			if expr, ok = so.evalExpr(nd.High); ok {
				nd.High = expr
			}
		}
	case *node.FuncLit:
		so.enterScope()
		defer so.leaveScope()
		for _, ident := range nd.Type.Params.Args.Values {
			so.scope.define(ident.Name)
		}
		for _, ident := range nd.Type.Params.NamedArgs.Names {
			so.scope.define(ident.Name)
		}
		if nd.Body != nil {
			_, _ = so.optimize(nd.Body)
		}
	case *node.ReturnStmt:
		if nd.Result != nil {
			if expr, ok = so.optimize(nd.Result); ok {
				nd.Result = expr
			}
			if expr, ok = so.evalExpr(nd.Result); ok {
				nd.Result = expr
			}
		}
	case *node.CallExpr:
		if nd.Func != nil {
			_, _ = so.optimize(nd.Func)
		}
		for i := range nd.Args.Values {
			if expr, ok = so.optimize(nd.Args.Values[i]); ok {
				nd.Args.Values[i] = expr
			}
			if expr, ok = so.evalExpr(nd.Args.Values[i]); ok {
				nd.Args.Values[i] = expr
			}
		}
		for i := range nd.NamedArgs.Values {
			if expr, ok = so.optimize(nd.NamedArgs.Values[i]); ok {
				nd.NamedArgs.Values[i] = expr
			}
			if expr, ok = so.evalExpr(nd.NamedArgs.Values[i]); ok {
				nd.NamedArgs.Values[i] = expr
			}
		}
	case *node.CondExpr:
		if expr, ok = so.optimize(nd.Cond); ok {
			nd.Cond = expr
		}
		if expr, ok = so.evalExpr(nd.Cond); ok {
			nd.Cond = expr
		}
		if falsy, ok := isLitFalsy(nd.Cond); ok {
			// convert expression to BoolLit so that Compiler skips expressions
			nd.Cond = &node.BoolLit{
				Value:    !falsy,
				Literal:  strconv.FormatBool(!falsy),
				ValuePos: nd.Cond.Pos(),
			}
		}

		if expr, ok = so.optimize(nd.True); ok {
			nd.True = expr
		}
		if expr, ok = so.evalExpr(nd.True); ok {
			nd.True = expr
		}
		if expr, ok = so.optimize(nd.False); ok {
			nd.False = expr
		}
		if expr, ok = so.evalExpr(nd.False); ok {
			nd.False = expr
		}
	}
	return nil, false
}

func (so *SimpleOptimizer) enterScope() {
	so.scope = &optimizerScope{parent: so.scope}
}

func (so *SimpleOptimizer) leaveScope() {
	so.scope = so.scope.parent
}

// Total returns total number of evaluated constants and expressions.
func (so *SimpleOptimizer) Total() int {
	return so.total
}

func (so *SimpleOptimizer) error(nd ast.Node, err error) error {
	pos := so.file.InputFile.Set().Position(nd.Pos())
	return &OptimizerError{
		FilePos: pos,
		Node:    nd,
		Err:     err,
	}
}

func (so *SimpleOptimizer) printTraceMsgf(format string, args ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	i := 2 * so.indent
	for i > n {
		_, _ = fmt.Fprint(so.trace, dots)
		i -= n
	}

	_, _ = fmt.Fprint(so.trace, dots[0:i], "<")
	_, _ = fmt.Fprintf(so.trace, format, args...)
	_, _ = fmt.Fprintln(so.trace, ">")
}

func traceoptim(so *SimpleOptimizer, msg string) *SimpleOptimizer {
	printTrace(so.indent, so.trace, msg, "{")
	so.indent++
	return so
}

func untraceoptim(so *SimpleOptimizer) {
	so.indent--
	printTrace(so.indent, so.trace, "}")
}

func isObjectConstant(obj Object) bool {
	switch obj.(type) {
	case Bool, Int, Uint, Float, Char, String, *NilType:
		return true
	}
	return false
}

func isLitFalsy(expr node.Expr) (bool, bool) {
	if expr == nil {
		return false, false
	}

	switch v := expr.(type) {
	case *node.BoolLit:
		return !v.Value, true
	case *node.IntLit:
		return Int(v.Value).IsFalsy(), true
	case *node.UintLit:
		return Uint(v.Value).IsFalsy(), true
	case *node.FloatLit:
		return Float(v.Value).IsFalsy(), true
	case *node.StringLit:
		return String(v.Value).IsFalsy(), true
	case *node.CharLit:
		return Char(v.Value).IsFalsy(), true
	case *node.NilLit:
		return Nil.IsFalsy(), true
	}
	return false, false
}

type multipleErr []error

func (m multipleErr) Errors() []error {
	return m
}

func (m multipleErr) Error() string {
	if len(m) == 0 {
		return ""
	}
	return m[0].Error()
}

func (m multipleErr) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		if len(m) == 0 {
			return
		}
		if len(m) > 1 {
			_, _ = fmt.Fprint(s, "multiple errors:\n ")
		}
		switch {
		case s.Flag('+'):
			_, _ = fmt.Fprint(s, m[0].Error())
			for _, err := range m[1:] {
				_, _ = fmt.Fprint(s, "\n ")
				_, _ = fmt.Fprint(s, err.Error())
			}
		case s.Flag('#'):
			_, _ = fmt.Fprintf(s, "%#v", []error(m))
		default:
			_, _ = fmt.Fprint(s, m.Error())
		}
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", m.Error())
	}
}
