// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

func (c *Compiler) compileMultiParenExpr(nd *node.MultiParenExpr) error {
	args, err := nd.ToCallArgs(false)
	if err != nil {
		return c.error(err.Node, err)
	}
	return c.compileCallExpr(&node.CallExpr{
		Func:     node.EIdent(TMixedParams.Name(), nd.LParen.Pos),
		CallArgs: *args,
	})
}

func (c *Compiler) compileIfStmt(nd *node.IfStmt) error {
	// open new symbol table for the statement
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	if nd.Init != nil {
		if err := c.Compile(nd.Init); err != nil {
			return err
		}
	}

	jumpPos1 := -1
	var skipElse bool
	if v, ok := nd.Cond.(node.BoolExpr); !ok {
		op := OpJumpFalsy
		if v, ok := simplifyExpr(nd.Cond).(*node.UnaryExpr); ok && v.Token.Is(token.Null, token.NotNull) {
			if err := c.Compile(v.Expr); err != nil {
				return err
			}

			op = OpJumpNotNil
			if v.Token == token.NotNull {
				op = OpJumpNil
			}
		} else if err := c.Compile(nd.Cond); err != nil {
			return err
		}

		// first jump placeholder
		jumpPos1 = c.emit(nd, op, 0)
		if err := c.Compile(nd.Body); err != nil {
			return err
		}
	} else if v.Bool() {
		if err := c.Compile(nd.Body); err != nil {
			return err
		}
		skipElse = true
	} else {
		jumpPos1 = c.emit(nd, OpJump, 0)
	}

	if !skipElse && nd.Else != nil {
		// second jump placeholder
		jumpPos2 := c.emit(nd, OpJump, 0)
		if jumpPos1 > -1 {
			// update first jump offset
			curPos := len(c.instructions)
			c.changeOperand(jumpPos1, curPos)
		}

		if err := c.Compile(nd.Else); err != nil {
			return err
		}
		// update second jump offset
		curPos := len(c.instructions)
		c.changeOperand(jumpPos2, curPos)
	} else {
		if jumpPos1 > -1 {
			// update first jump offset
			curPos := len(c.instructions)
			c.changeOperand(jumpPos1, curPos)
		}
	}
	return nil
}

func (c *Compiler) compileTryStmt(nd *node.TryStmt) error {
	/*
		// create a single symbol table for try-catch-finally
		// any `return` statement in finally block ignores already thrown error.
		try {
			// emit: OpSetupTry (CatchPos, FinallyPos)

			// emit: OpJump (FinallyPos) // jump to finally block to skip catch block.
		} catch err {
			// emit: OpSetupCatch
			//
			// catch block is optional.
			// if err is elided  in `catch {}`, OpPop removes the error from stack.
			// catch pops the error from error handler, re-throw requires explicit
			// throw expression `throw err`.
		} finally {
			// emit: OpSetupFinally
			//
			// finally block is optional if catch block is defined but
			// instructions are always generated for finally block even if not explicitly defined
			// to cleanup symbols and re-throw error if not handled in catch block.
			//
			// emit: OpThrow 0 // this is implicit re-throw operation without putting stack trace
		}
	*/
	// fork new symbol table for the statement
	c.symbolTable = c.symbolTable.Fork(true)
	c.tryCatchIndex++
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
		c.emit(nd, OpThrow, 0) // implicit re-throw
	}()

	optry := c.emit(nd, OpSetupTry, 0, 0)
	var catchPos, finallyPos int
	if nd.Body != nil && len(nd.Body.Stmts) > 0 {
		// in order not to fork symbol table in Body, compile stmts here instead of in *BlockStmt
		for _, stmt := range nd.Body.Stmts {
			if err := c.Compile(stmt); err != nil {
				return err
			}
		}
	}

	var opjump int
	if nd.Catch != nil {
		// if there is no thrown error before catch statement, set catch ident to nil
		// otherwise jumping to finally and accessing ident in finally access previous set same index variable.
		if nd.Catch.Ident != nil {
			c.emit(nd.Catch, OpNil)
			symbol, exists := c.symbolTable.DefineLocal(nd.Catch.Ident.Name)
			if exists {
				c.emit(nd, OpSetLocal, symbol.Index)
			} else {
				c.emit(nd, OpDefineLocal, symbol.Index)
			}
		}

		opjump = c.emit(nd, OpJump, 0)
		catchPos = len(c.instructions)
		if err := c.Compile(nd.Catch); err != nil {
			return err
		}
	}

	c.tryCatchIndex--
	// always emit OpSetupFinally to cleanup
	if nd.Finally != nil {
		finallyPos = c.emit(nd.Finally, OpSetupFinally)
		if err := c.Compile(nd.Finally); err != nil {
			return err
		}
	} else {
		finallyPos = c.emit(nd, OpSetupFinally)
	}

	c.changeOperand(optry, catchPos, finallyPos)
	if nd.Catch != nil {
		// no need jumping if catch is not defined
		c.changeOperand(opjump, finallyPos)
	}
	return nil
}

// compileMatchExpr compiles a PHP8-like match. The subject is evaluated once
// into a temp local and compared (strict equality) against each arm's
// conditions; an arm matches when the subject equals any of its conditions
// (`A, B: …`), and the first matching arm wins. Expression-form arms
// (`conds: result`) leave the matched result on the stack; statement-form arms
// (`conds { body }`) run the block and yield nil. An optional `else` arm is the
// default; when nothing matches and there is no `else`, the match yields nil.
func (c *Compiler) compileMatchExpr(nd *node.MatchExpr) error {
	isStmt := nd.IsStmt()

	// validate arm shapes are consistent and locate the else arm
	var elseArm *node.MatchArm
	for _, arm := range nd.Arms {
		if arm.IsElse() {
			if elseArm != nil {
				return c.errorf(nd, "multiple else arms in match")
			}
			elseArm = arm
		}
		if isStmt && arm.Result != nil {
			return c.errorf(nd, "cannot mix `cond: result` and `cond { body }` match arms")
		}
		if !isStmt && arm.Body != nil {
			return c.errorf(nd, "cannot mix `cond: result` and `cond { body }` match arms")
		}
	}

	c.symbolTable = c.symbolTable.Fork(true)
	defer func() { c.symbolTable = c.symbolTable.Parent(false) }()

	// evaluate the subject once into a temp local
	if err := c.Compile(nd.Expr); err != nil {
		return err
	}
	subjectSym, _ := c.symbolTable.DefineLocal(":match")
	c.emit(nd, OpDefineLocal, subjectSym.Index)

	var endJumps []int
	for _, arm := range nd.Arms {
		if arm.IsElse() {
			continue // else handled after the loop
		}

		// `subject == cond_0 || subject == cond_1 || ...` — jump to the body on
		// the first matching condition, otherwise fall through to the next arm.
		var matchJumps []int
		for _, cond := range arm.Conds {
			c.emit(cond, OpGetLocal, subjectSym.Index)
			if err := c.Compile(cond); err != nil {
				return err
			}
			c.emit(cond, OpEqual)
			noMatch := c.emit(cond, OpJumpFalsy, 0)
			matchJumps = append(matchJumps, c.emit(cond, OpJump, 0))
			c.changeOperand(noMatch, len(c.instructions))
		}
		toNext := c.emit(nd, OpJump, 0)

		// body
		bodyStart := len(c.instructions)
		for _, j := range matchJumps {
			c.changeOperand(j, bodyStart)
		}
		if err := c.compileMatchArmBody(nd, arm, isStmt); err != nil {
			return err
		}
		endJumps = append(endJumps, c.emit(nd, OpJump, 0))

		c.changeOperand(toNext, len(c.instructions))
	}

	// no condition matched: the expression form yields the else value or nil;
	// the statement form runs the else block or does nothing (leaves no value).
	if elseArm != nil {
		if err := c.compileMatchArmBody(nd, elseArm, isStmt); err != nil {
			return err
		}
	} else if !isStmt {
		c.emit(nd, OpNil)
	}

	end := len(c.instructions)
	for _, j := range endJumps {
		c.changeOperand(j, end)
	}
	return nil
}

// compileMatchArmBody compiles a single match arm body. The expression form
// leaves the arm's result value on the stack; the statement form runs the arm
// block and leaves nothing (the match as a whole is value-less).
func (c *Compiler) compileMatchArmBody(nd *node.MatchExpr, arm *node.MatchArm, isStmt bool) error {
	if isStmt {
		if arm.Body != nil {
			return c.Compile(arm.Body)
		}
		return nil
	}
	return c.Compile(arm.Result)
}

// wrapComprehensionClauses wraps an innermost statement with the comprehension's
// `for`/`if` clauses (outermost clause first), producing nested ForInStmt/IfStmt.
func wrapComprehensionClauses(clauses []*node.ComprehensionClause, inner node.Stmt) node.Stmt {
	body := inner
	for i := len(clauses) - 1; i >= 0; i-- {
		cl := clauses[i]
		block := &node.BlockStmt{Stmts: node.Stmts{body}}
		if cl.For {
			key := cl.Key
			if key == nil {
				key = node.EEmptyIdent(cl.Value.Pos())
			}
			body = &node.ForInStmt{
				Key:      key,
				Value:    cl.Value,
				Iterable: cl.Iterable,
				Body:     block,
			}
		} else {
			body = &node.IfStmt{Cond: cl.Cond, Body: block}
		}
	}
	return body
}

// compileArrayComprehension compiles `[elem for x in it if cond ...]` by
// building a temp array and appending elem for each iteration that passes the
// filters, then leaving the array on the stack.
func (c *Compiler) compileArrayComprehension(nd *node.ArrayComprehension) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() { c.symbolTable = c.symbolTable.Parent(false) }()

	// :compr := []
	resultSym, _ := c.symbolTable.DefineLocal(":compr")
	c.emit(nd, OpArray, 0)
	c.emit(nd, OpDefineLocal, resultSym.Index)

	result := &node.IdentExpr{Name: ":compr"}
	// :compr = append(:compr, elem)
	inner := &node.AssignStmt{
		LHS: []node.Expr{result},
		RHS: []node.Expr{&node.CallExpr{
			Func: &node.IdentExpr{Name: BuiltinAppend.String()},
			CallArgs: node.CallArgs{Args: node.CallExprPositionalArgs{
				Values: []node.Expr{result, nd.Element},
			}},
		}},
		Token: token.Assign,
	}

	if err := c.Compile(wrapComprehensionClauses(nd.Clauses, inner)); err != nil {
		return err
	}

	c.emit(nd, OpGetLocal, resultSym.Index)
	return nil
}

// compileDictComprehension compiles
// `{k1: v1, [ke]: ve, ... for x in it if cond}` by building a dict bound to the
// special variable `_` and, for each passing iteration, assigning every element
// into it. Static keys (`name:`) use the literal name; computed keys (`[expr]:`)
// evaluate the expression. Value expressions may read/modify the in-progress
// dict via `_` (e.g. `_.z ?? 20`).
func (c *Compiler) compileDictComprehension(nd *node.DictComprehension) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() { c.symbolTable = c.symbolTable.Parent(false) }()

	// `_` refers to the dict being built
	resultSym, _ := c.symbolTable.DefineLocal("_")
	c.emit(nd, OpDict, 0)
	c.emit(nd, OpDefineLocal, resultSym.Index)

	// inner body: _[k1] = v1; _[k2] = v2; ...
	var stmts node.Stmts
	for _, el := range nd.Elements {
		stmts = append(stmts, &node.AssignStmt{
			LHS: []node.Expr{&node.IndexExpr{
				X:     &node.IdentExpr{Name: "_"},
				Index: el.BuildKeyExpr(),
			}},
			RHS:   []node.Expr{el.Value},
			Token: token.Assign,
		})
	}
	inner := &node.BlockStmt{Stmts: stmts}

	if err := c.Compile(wrapComprehensionClauses(nd.Clauses, inner)); err != nil {
		return err
	}

	c.emit(nd, OpGetLocal, resultSym.Index)
	return nil
}

// compileOrExpr compiles an `expr or fallback` error-fallback expression. It is
// desugared to a try/catch that evaluates Expr and, on a thrown error, evaluates
// Fallback instead with the caught error bound to the local `$err`. The result
// (either value) is left on the stack.
func (c *Compiler) compileOrExpr(nd *node.OrExpr) error {
	// fork a new symbol table so `$err` and the temp result do not leak
	c.symbolTable = c.symbolTable.Fork(true)
	c.tryCatchIndex++

	// temp local holding the resulting value of the whole expression
	tmp, _ := c.symbolTable.DefineLocal(":or")
	c.emit(nd, OpNil)
	c.emit(nd, OpDefineLocal, tmp.Index)

	optry := c.emit(nd, OpSetupTry, 0, 0)

	// try body: evaluate Expr and store its value
	if err := c.Compile(nd.Expr); err != nil {
		return err
	}
	c.emit(nd, OpSetLocal, tmp.Index)
	opjump := c.emit(nd, OpJump, 0)

	// catch body: bind $err and evaluate Fallback
	catchPos := len(c.instructions)
	c.emit(nd, OpSetupCatch)
	errSym, exists := c.symbolTable.DefineLocal("$err")
	if exists {
		c.emit(nd, OpSetLocal, errSym.Index)
	} else {
		c.emit(nd, OpDefineLocal, errSym.Index)
	}
	if err := c.Compile(nd.Fallback); err != nil {
		return err
	}
	c.emit(nd, OpSetLocal, tmp.Index)
	// If the fallback itself evaluated to an error, re-throw it (so
	// `x() or error("...")` propagates the new error); otherwise it is the
	// resulting value (so `x() or 2` / `x() or ("..." + $err)` yield a value).
	c.emit(nd, OpGetBuiltin, int(BuiltinIsError))
	c.emit(nd, OpGetLocal, tmp.Index)
	c.emit(nd, OpCall, 1, 0)
	opNotErr := c.emit(nd, OpJumpFalsy, 0)
	c.emit(nd, OpGetLocal, tmp.Index)
	c.emit(nd, OpThrow, 1)

	c.tryCatchIndex--

	// finally: cleanup + implicit re-throw (no-op when catch handled the error)
	finallyPos := c.emit(nd, OpSetupFinally)
	c.emit(nd, OpThrow, 0)

	c.changeOperand(optry, catchPos, finallyPos)
	c.changeOperand(opjump, finallyPos)
	c.changeOperand(opNotErr, finallyPos)

	c.symbolTable = c.symbolTable.Parent(false)

	// push the resulting value
	c.emit(nd, OpGetLocal, tmp.Index)
	return nil
}

func (c *Compiler) compileCatchStmt(nd *node.CatchStmt) error {
	c.emit(nd, OpSetupCatch)
	if nd.Ident != nil {
		symbol, exists := c.symbolTable.DefineLocal(nd.Ident.Name)
		if exists {
			c.emit(nd, OpSetLocal, symbol.Index)
		} else {
			c.emit(nd, OpDefineLocal, symbol.Index)
		}
	} else {
		c.emit(nd, OpPop)
	}

	if nd.Body == nil {
		return nil
	}

	// in order not to fork symbol table in Body, compile stmts here instead of in *BlockStmt
	for _, stmt := range nd.Body.Stmts {
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileFinallyStmt(nd *node.FinallyStmt) error {
	if nd.Body == nil {
		return nil
	}

	// in order not to fork symbol table in Body, compile stmts here instead of in *BlockStmt
	for _, stmt := range nd.Body.Stmts {
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileThrowStmt(nd *node.ThrowStmt) error {
	if nd.Expr != nil {
		if err := c.Compile(nd.Expr); err != nil {
			return err
		}
	}
	c.emit(nd, OpThrow, 1)
	return nil
}

func (c *Compiler) compileThrowExpr(nd *node.ThrowExpr) error {
	if nd.Expr != nil {
		if err := c.Compile(nd.Expr); err != nil {
			return err
		}
	}
	c.emit(nd, OpThrow, 1)
	return nil
}

func (c *Compiler) compileDeclStmt(nd *node.DeclStmt) error {
	decl := nd.Decl.(*node.GenDecl)
	if len(decl.Specs) == 0 {
		return c.errorf(nd, "empty declaration not allowed")
	}

	switch decl.Tok {
	case token.Param:
		return c.compileDeclParam(decl)
	case token.Global:
		return c.compileDeclGlobal(decl)
	case token.Var, token.Const:
		return c.compileDeclValue(decl)
	}
	return nil
}

func (c *Compiler) compileDeclParam(nd *node.GenDecl) error {
	if c.symbolTable.parent != nil {
		return c.errorf(nd, "param not allowed in this scope")
	}

	var (
		positionalSpecs, namedSpecs = nd.Params()
		positional                  = make([]*Param, len(positionalSpecs))
		named                       = make([]*NamedParam, len(namedSpecs))
	)

	for i, spec := range positionalSpecs {
		var (
			p = &Param{
				Name: spec.Ident.Ident.Name,
				Var:  spec.Var,
			}
		)

		symbol, err := c.requireSymbol(nd, p.Name)
		if err != nil {
			return err
		}

		p.Symbol = &symbol.SymbolInfo

		symbols := make([]*SymbolInfo, len(spec.Ident.Type))
		for i2, t := range spec.Ident.Type {
			symbol, err := c.requireSymbol(nd, t.Ident().Name)
			if err != nil {
				return err
			}
			symbols[i2] = &symbol.SymbolInfo
		}
		p.TypesSymbols = symbols

		if spec.Var {
			if i != len(positionalSpecs)-1 {
				return c.errorf(nd,
					"only last param accept variadic")
			}
			c.variadic = true
		}

		positional[i] = p
	}

	var namedSpecCount = len(namedSpecs)

	for i, spec := range namedSpecs {
		np := &NamedParam{
			Name:         spec.Ident.Ident.Name,
			Var:          spec.Var,
			TypesSymbols: make(ParamType, 0),
		}

		symbol, err := c.requireSymbol(nd, np.Name)
		if err != nil {
			return err
		}
		np.Symbol = &symbol.SymbolInfo

		if spec.Var {
			if i != len(namedSpecs)-1 {
				return c.errorf(nd,
					"only last named param accept variadic")
			}

			namedSpecCount--
			c.varNamedParams = true
		} else {
			if spec.Value == nil {
				spec.Value = node.Flag(false, spec.Pos())
			}
			np.Value = spec.Value.String()
			np.TypesSymbols = make([]*SymbolInfo, len(spec.Ident.Type))
			for i2, t := range spec.Ident.Type {
				symbol, err := c.requireSymbol(nd, t.Ident().Name)
				if err != nil {
					return err
				}
				np.TypesSymbols[i2] = &symbol.SymbolInfo
			}
		}

		named[i] = np
	}

	if err := c.symbolTable.defineParams(NewParams(positional...), NewNamedParams(named...)); err != nil {
		return c.error(nd, err)
	}

	stmts := c.helperBuildKwargsStmts(namedSpecCount, func(index int) (name *node.IdentExpr, value node.Expr) {
		spec := namedSpecs[index]
		return spec.Ident.Ident, spec.Value
	})

	return c.Compile(&node.BlockStmt{Stmts: stmts})
}

func (c *Compiler) compileDeclGlobal(nd *node.GenDecl) error {
	if c.symbolTable.parent != nil {
		return c.errorf(nd, "global not allowed in this scope")
	}

	for _, sp := range nd.Specs {
		spec := sp.(*node.ParamSpec)
		symbol, err := c.symbolTable.DefineGlobal(spec.Ident.Ident.Name)
		if err != nil {
			return c.error(nd, err)
		}

		idx := c.addConstant(Str(spec.Ident.Ident.Name))
		symbol.Index = idx
	}
	return nil
}

func (c *Compiler) compileDeclValue(nd *node.GenDecl) error {
	var (
		isConst  bool
		lastExpr node.Expr
	)
	if nd.Tok == token.Const {
		isConst = true
		defer func() { c.iotaVal = -1 }()
	}

	for _, sp := range nd.Specs {
		spec := sp.(*node.ValueSpec)
		if isConst {
			if spec.Data != nil {
				if v, ok := spec.Data.(int); ok {
					c.iotaVal = v
				} else {
					return c.errorf(nd, "invalid iota value")
				}
			}
		}
		for i, ident := range spec.Idents {
			leftExpr := []node.Expr{ident}
			var v node.Expr
			if i < len(spec.Values) {
				v = spec.Values[i]
			}

			if v == nil {
				if isConst && lastExpr != nil {
					v = lastExpr
				} else {
					v = &node.NilLit{TokenPos: ident.Pos()}
				}
			} else {
				lastExpr = v
			}

			assign := &node.AssignStmt{
				Token:    token.Define,
				LHS:      leftExpr,
				RHS:      []node.Expr{v},
				TokenPos: ident.Pos(),
			}

			if err := c.atDo(assign, func() error {
				return c.compileAssignStmt(assign, assign.LHS, assign.RHS, nd.Tok, assign.Token)
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Compiler) checkAssignment(
	nd ast.Node,
	lhs []node.Expr,
	rhs []node.Expr,
	op token.Token,
) (bool, error) {
	_, numRHS := len(lhs), len(rhs)
	if numRHS > 1 {
		return false, c.errorf(nd,
			"multiple expressions on the right side not supported")
	}

	var selector bool
Loop:
	for _, expr := range lhs {
		switch expr.(type) {
		case *node.SelectorExpr, *node.IndexExpr:
			selector = true
			break Loop
		}
	}

	if selector {
		if op == token.Define {
			// using selector on new variable does not make sense
			return false, c.errorf(nd, "operator ':=' not allowed with selector")
		}
	}

	return true, nil
}

func (c *Compiler) compileAssignStmt(
	nd ast.Node,
	lhs []node.Expr,
	rhs []node.Expr,
	keyword token.Token,
	op token.Token,
) error {
	compile, err := c.checkAssignment(nd, lhs, rhs, op)
	if err != nil || !compile {
		return err
	}

	// dict destructuring: `(;a, _b:b, r=2, **other) = dict`
	if len(lhs) == 1 {
		if kva, ok := lhs[0].(*node.KeyValueArrayLit); ok {
			return c.compileDictDestructuring(nd, kva, rhs, keyword, op)
		}
		// MixedParams destructuring: `(a, b, **pos_rest; c, p:d, **named_rest) = mp`
		if mp, ok := lhs[0].(*node.MultiParenExpr); ok {
			return c.compileMixedParamsDestructuring(nd, mp, rhs, keyword, op)
		}
	}

	var isArrDestruct bool
	var tempArrSymbol *Symbol
	// +=, -=, *=, /=
	if op != token.Assign && op != token.Define {
		if err := c.Compile(lhs[0]); err != nil {
			return err
		}
	} else if len(lhs) > 1 {
		isArrDestruct = true
		// ignore redefinition of :array symbol, it can be used multiple times
		// within a block.
		tempArrSymbol, _ = c.symbolTable.DefineLocal(":array")
		// ignore disabled builtins of symbol table for BuiltinMakeArray because
		// it is required to handle destructuring assignment.
		c.emit(nd, OpGetBuiltin, int(BuiltinMakeArray))
		c.emit(nd, OpConstant, c.addConstant(Int(len(lhs))))
	}

	if op == token.AddAssign {
		switch lhs[0].(type) {
		case *node.ModuleLit:
			// compile RHSs
			for _, expr := range rhs {
				if err := c.Compile(expr); err != nil {
					return err
				}
			}
			c.emit(nd, OpSelfAssign, int(token.Unassign(op)))
			return nil
		}
	}

	if op == token.Assign {
		switch lhs[0].(type) {
		case *node.StdInLit, *node.StdOutLit, *node.StdErrLit:
			var fd int64
			switch lhs[0].(type) {
			case *node.StdOutLit:
				fd = 1
			case *node.StdErrLit:
				fd = 2
			}
			return c.compileCallExpr(&node.CallExpr{
				Func: &node.IdentExpr{Name: BuiltinStdIO.String()},
				CallArgs: node.CallArgs{Args: node.CallExprPositionalArgs{Values: []node.Expr{
					&node.IntLit{Value: fd},
					rhs[0],
				}}},
			})
		}
	}

	if op == token.NullichAssign || op == token.LOrAssign {
		op2 := OpJumpNotNil
		if op == token.LOrAssign {
			op2 = OpOrJump
		}
		jumpPos := c.emit(nd, op2, 0)
		// compile RHSs
		for _, expr := range rhs {
			if err := c.Compile(expr); err != nil {
				return err
			}
		}
		if err := c.compileDefineAssign(nd, lhs[0], keyword, token.Assign, false); err != nil {
			return err
		}
		c.changeOperand(jumpPos, len(c.instructions))
		return nil
	}

	// compile RHSs
	for _, expr := range rhs {
		if err := c.Compile(expr); err != nil {
			return err
		}
	}

	if isArrDestruct {
		return c.compileDestructuring(nd, lhs, tempArrSymbol, keyword, op)
	}

	if op > token.GroupSelfAssignOperatorBegin && op < token.GroupSelfAssignOperatorEnd {
		c.emit(nd, OpSelfAssign, int(token.Unassign(op)))
	}

	return c.compileDefineAssign(nd, lhs[0], keyword, op, false)
}

func (c *Compiler) compileDestructuring(
	nd ast.Node,
	lhs []node.Expr,
	tempArrSymbol *Symbol,
	keyword token.Token,
	op token.Token,
) error {
	c.emit(nd, OpCall, 2, 0)
	c.emit(nd, OpDefineLocal, tempArrSymbol.Index)
	numLHS := len(lhs)
	var found int

	for lhsIndex, expr := range lhs {
		if op == token.Define {
			if term, ok := expr.(*node.IdentExpr); ok {
				if _, ok = c.symbolTable.find(term.Name); ok {
					found++
				}
			}
			if found == numLHS {
				return c.errorf(nd, "no new variable on left side")
			}
		}

		c.emit(nd, OpGetLocal, tempArrSymbol.Index)
		c.emit(nd, OpConstant, c.addConstant(Int(lhsIndex)))
		c.emit(nd, OpGetIndex, 1)
		err := c.compileDefineAssign(nd, expr, keyword, op, keyword != token.Const)
		if err != nil {
			return err
		}
	}

	if !c.symbolTable.InBlock() {
		// blocks set nil to variables defined in it after block
		c.emit(nd, OpNil)
		c.emit(nd, OpSetLocal, tempArrSymbol.Index)
	}
	return nil
}

// destructureKeyName extracts the identifier or string-literal name used as a
// dict key / target variable in a dict-destructuring element.
func destructureKeyName(e node.Expr) (string, bool) {
	switch t := e.(type) {
	case *node.IdentExpr:
		return t.Name, true
	case *node.TypedIdentExpr:
		return t.Ident.Name, true
	case *node.StrLit:
		return t.Value(), true
	}
	return "", false
}

// compileDictDestructuring compiles `(;a, _b:b, r=2, **other) = dict`.
//
//	a        -> a    = dict["a"]            (nil when absent)
//	_b:b     -> _b   = dict["b"]            (rename)
//	r=2      -> r    = dict["r"] ?? 2       (fallback default)
//	**other  -> other = remaining keys      (optional, must be last)
func (c *Compiler) compileDictDestructuring(
	nd ast.Node,
	kva *node.KeyValueArrayLit,
	rhs []node.Expr,
	keyword token.Token,
	op token.Token,
) error {
	if op != token.Assign && op != token.Define {
		return c.errorf(nd, "operator %q not allowed with dict destructuring",
			op.String())
	}

	var (
		pairs   []*node.KeyValuePairLit
		restVar node.Expr
	)
	for _, el := range kva.Elements {
		switch e := el.(type) {
		case *node.KeyValuePairLit:
			if restVar != nil {
				return c.errorf(nd, "** rest target must be the last element")
			}
			pairs = append(pairs, e)
		case *node.NamedArgVarLit:
			if restVar != nil {
				return c.errorf(nd, "only one ** rest target is allowed")
			}
			restVar = e.Value
		default:
			return c.errorf(nd, "invalid dict destructuring target %T", el)
		}
	}

	// evaluate the source dict once into a temp local
	if err := c.Compile(rhs[0]); err != nil {
		return err
	}
	dictSym, _ := c.symbolTable.DefineLocal(":dict")
	c.emit(nd, OpDefineLocal, dictSym.Index)

	hasRest := restVar != nil
	if hasRest {
		// copy so consumed keys can be removed for **other without mutating the
		// source dict
		c.emit(nd, OpGetBuiltin, int(BuiltinCopy))
		c.emit(nd, OpGetLocal, dictSym.Index)
		c.emit(nd, OpCall, 1, 0)
		c.emit(nd, OpSetLocal, dictSym.Index)
	}

	// `:=` defines new locals for all targets; `=` assigns to all targets
	// (which must already be defined).
	allowRedefine := keyword != token.Const
	for _, pair := range pairs {
		dictKey, ok := destructureKeyName(pair.Key)
		if !ok {
			return c.errorf(nd, "invalid dict destructuring key %T", pair.Key)
		}
		if pair.Colon {
			// rename: the value names the source key
			dictKey, ok = destructureKeyName(pair.Value)
			if !ok {
				return c.errorf(nd, "invalid dict destructuring source key %T",
					pair.Value)
			}
		}

		// push dict[dictKey]
		c.emit(nd, OpGetLocal, dictSym.Index)
		c.emit(nd, OpConstant, c.addConstant(Str(dictKey)))
		c.emit(nd, OpGetIndex, 1)

		// fallback default: `name=expr` uses expr when the key is absent (nil)
		if !pair.Colon && pair.Value != nil {
			jp := c.emit(nd, OpJumpNotNil, 0)
			if err := c.Compile(pair.Value); err != nil {
				return err
			}
			c.changeOperand(jp, len(c.instructions))
		}

		if err := c.compileDefineAssign(nd, pair.Key, keyword, op, allowRedefine); err != nil {
			return err
		}

		if hasRest {
			// remove the consumed key from the copy
			c.emit(nd, OpGetBuiltin, int(BuiltinDelete))
			c.emit(nd, OpGetLocal, dictSym.Index)
			c.emit(nd, OpConstant, c.addConstant(Str(dictKey)))
			c.emit(nd, OpCall, 2, 0)
			c.emit(nd, OpPop)
		}
	}

	if hasRest {
		c.emit(nd, OpGetLocal, dictSym.Index)
		if err := c.compileDefineAssign(nd, restVar, keyword, op, allowRedefine); err != nil {
			return err
		}
	}

	if !c.symbolTable.InBlock() {
		c.emit(nd, OpNil)
		c.emit(nd, OpSetLocal, dictSym.Index)
	}
	return nil
}

// compileMixedParamsDestructuring compiles
// `(a, b, **pos_rest; c, p:d, r=2, **named_rest) = mp` against a MixedParams
// value. The positional side reads mp.positional (with an optional `**rest`
// slice); the named side reuses dict destructuring against dict(mp.named).
func (c *Compiler) compileMixedParamsDestructuring(
	nd ast.Node,
	mp *node.MultiParenExpr,
	rhs []node.Expr,
	keyword token.Token,
	op token.Token,
) error {
	if op != token.Assign && op != token.Define {
		return c.errorf(nd, "operator %q not allowed with destructuring", op.String())
	}

	// evaluate the source MixedParams once into a temp local
	if err := c.Compile(rhs[0]); err != nil {
		return err
	}
	mpSym, _ := c.symbolTable.DefineLocal("$__mp")
	c.emit(nd, OpDefineLocal, mpSym.Index)

	mpIdent := func() node.Expr { return &node.IdentExpr{Name: "$__mp"} }
	positional := func() node.Expr { return node.ESelector(mpIdent(), node.Str("positional", 0)) }

	allowRedefine := keyword != token.Const

	// positional targets: a = mp.positional[i]; **rest = mp.positional[i:]
	var restSeen bool
	for i, el := range mp.PositionalElements {
		if av, ok := el.(*node.NamedArgVarLit); ok {
			if restSeen {
				return c.errorf(nd, "only one ** rest target is allowed in the positional section")
			}
			restSeen = true
			slice := node.ESlice(positional(), &node.IntLit{Value: int64(i)}, nil, 0, 0)
			if err := c.compileDefineAssignValue(nd, av.Value, slice, keyword, op, allowRedefine); err != nil {
				return err
			}
			continue
		}
		if restSeen {
			return c.errorf(nd, "** rest target must be the last positional element")
		}
		idx := node.EIndex(positional(), &node.IntLit{Value: int64(i)}, 0, 0)
		if err := c.compileDefineAssignValue(nd, el, idx, keyword, op, allowRedefine); err != nil {
			return err
		}
	}

	// named targets: reuse dict destructuring against dict(mp.named)
	if len(mp.NamedElements) > 0 {
		named := node.ESelector(mpIdent(), node.Str("named", 0))
		dictCall := &node.CallExpr{
			Func: &node.IdentExpr{Name: BuiltinDict.String()},
			CallArgs: node.CallArgs{Args: node.CallExprPositionalArgs{
				Values: []node.Expr{named},
			}},
		}
		kva := &node.KeyValueArrayLit{Elements: mp.NamedElements}
		if err := c.compileDictDestructuring(nd, kva, []node.Expr{dictCall}, keyword, op); err != nil {
			return err
		}
	}

	if !c.symbolTable.InBlock() {
		c.emit(nd, OpNil)
		c.emit(nd, OpSetLocal, mpSym.Index)
	}
	return nil
}

// compileDefineAssignValue compiles `target OP value`, where value is an
// expression that is evaluated and then bound to target.
func (c *Compiler) compileDefineAssignValue(
	nd ast.Node,
	target node.Expr,
	value node.Expr,
	keyword token.Token,
	op token.Token,
	allowRedefine bool,
) error {
	if err := c.Compile(value); err != nil {
		return err
	}
	return c.compileDefineAssign(nd, target, keyword, op, allowRedefine)
}

func (c *Compiler) compileDefine(
	nd ast.Node,
	ident string,
	allowRedefine bool,
	keyword token.Token,
) error {
	symbol, exists := c.symbolTable.DefineLocal(ident)
	if !allowRedefine && exists && ident != "_" {
		return c.errorf(nd, "%q redeclared in this block", ident)
	}

	if symbol.Constant {
		return c.errorf(nd, "assignment to constant variable %q", ident)
	}
	if c.iotaVal > -1 && ident == "iota" && keyword == token.Const {
		return c.errorf(nd, "assignment to iota")
	}

	c.emit(nd, OpDefineLocal, symbol.Index)
	symbol.Assigned = true
	symbol.Constant = keyword == token.Const && ident != "_"
	return nil
}

func (c *Compiler) compileAssign(
	nd ast.Node,
	symbol *Symbol,
	ident string,
) error {
	if symbol.Constant {
		return c.errorf(nd, "assignment to constant variable %q", ident)
	}

	switch symbol.Scope {
	case ScopeLocal:
		c.emit(nd, OpSetLocal, symbol.Index)
		symbol.Assigned = true
	case ScopeFree:
		c.emit(nd, OpSetFree, symbol.Index)
		symbol.Assigned = true
		s := symbol
		for s != nil {
			if s.Original != nil && s.Original.Scope == ScopeLocal {
				s.Original.Assigned = true
			}
			s = s.Original
		}
	case ScopeGlobal:
		c.emit(nd, OpSetGlobal, symbol.Index)
		symbol.Assigned = true
	default:
		return c.errorf(nd, "unresolved reference %q", ident)
	}
	return nil
}

func (c *Compiler) compileDefineAssign(
	nd ast.Node,
	lhs node.Expr,
	keyword token.Token,
	op token.Token,
	allowRedefine bool,
) error {
	ident, identExpr, selectors := resolveAssignLHS(lhs)

	numSel := len(selectors)
	if numSel == 0 && op == token.Define {
		return c.compileDefine(nd, ident, allowRedefine, keyword)
	} else if _, ok := identExpr.(*node.ModuleLit); ok {
		c.emit(nd, OpModule)
	} else {
		symbol, err := c.requireSymbol(nd, ident)
		if err != nil {
			return err
		}

		if numSel == 0 {
			return c.compileAssign(nd, symbol, ident)
		}

		// get indexes until last one and set the value to the last index
		switch symbol.Scope {
		case ScopeLocal:
			c.emit(nd, OpGetLocal, symbol.Index)
		case ScopeFree:
			c.emit(nd, OpGetFree, symbol.Index)
		case ScopeGlobal:
			c.emit(nd, OpGetGlobal, symbol.Index)
		default:
			return c.errorf(nd, "unexpected scope %q for symbol %q",
				symbol.Scope, ident)
		}
	}

	if numSel > 1 {
		for i := 0; i < numSel-1; i++ {
			if err := c.Compile(selectors[i]); err != nil {
				return err
			}
		}
		c.emit(nd, OpGetIndex, numSel-1)
	}

	if err := c.Compile(selectors[numSel-1]); err != nil {
		return err
	}

	c.emit(nd, OpSetIndex)
	return nil
}

func resolveAssignLHS(expr node.Expr) (name string, nameExpr node.Expr, selectors []node.Expr) {
	switch term := expr.(type) {
	case *node.SelectorExpr:
		name, nameExpr, selectors = resolveAssignLHS(term.X)
		selectors = append(selectors, term.Sel)
	case *node.IndexExpr:
		name, nameExpr, selectors = resolveAssignLHS(term.X)
		selectors = append(selectors, term.Index)
	case *node.IdentExpr:
		name = term.Name
		nameExpr = term
	case *node.ModuleLit:
		name = term.String()
		nameExpr = term
	}
	return
}

func (c *Compiler) compileBranchStmt(nd *node.BranchStmt) error {
	switch nd.Token {
	case token.Break:
		curLoop := c.currentLoop()
		if curLoop == nil {
			return c.errorf(nd, "break not allowed outside loop")
		}

		var pos int
		if curLoop.lastTryCatchIndex == c.tryCatchIndex {
			pos = c.emit(nd, OpJump, 0)
		} else {
			c.emit(nd, OpFinalizer, curLoop.lastTryCatchIndex+1)
			pos = c.emit(nd, OpJump, 0)
		}
		curLoop.breaks = append(curLoop.breaks, pos)
	case token.Continue:
		curLoop := c.currentLoop()
		if curLoop == nil {
			return c.errorf(nd, "continue not allowed outside loop")
		}

		var pos int
		if curLoop.lastTryCatchIndex == c.tryCatchIndex {
			pos = c.emit(nd, OpJump, 0)
		} else {
			c.emit(nd, OpFinalizer, curLoop.lastTryCatchIndex+1)
			pos = c.emit(nd, OpJump, 0)
		}
		curLoop.continues = append(curLoop.continues, pos)
	default:
		return c.errorf(nd, "invalid branch statement: %s", nd.Token.String())
	}
	return nil
}

func (c *Compiler) compileBlockStmt(nd *node.BlockStmt) error {
	if len(nd.Stmts) == 0 {
		return nil
	}

	// desugar block-scoped `deferb` by wrapping the block in the deferb runner
	if stmtsHaveDeferb(nd.Stmts) {
		wrapped, err := c.wrapDeferbBlock(nd.Stmts)
		if err != nil {
			return err
		}
		nd.Stmts = wrapped
	}

	c.symbolTable = c.symbolTable.Fork(true)
	if err := c.compileStmts(nd.Stmts...); err != nil {
		return err
	}

	c.symbolTable = c.symbolTable.Parent(false)
	return nil
}

func (c *Compiler) compileReturn(nd *node.Return) error {
	if nd.Result == nil {
		if c.tryCatchIndex > -1 {
			c.emit(nd, OpFinalizer, 0)
		}
		c.emit(nd, OpReturn, 0)
		return nil
	}

	if nd.Assign {
		switch t := nd.Result.(type) {
		case *node.IdentExpr:
			symbol, err := c.requireSymbol(nd, t.Name)
			if err != nil {
				return err
			}
			c.emit(nd, OpSetReturn, symbol.Index)
		case *node.ModuleLit:
			c.emit(nd, OpSetReturnModule)
		default:
			return c.errorf(nd, "return of assign require Ident|ModuleLit")
		}
	} else {
		if err := c.Compile(nd.Result); err != nil {
			return err
		}
		if c.tryCatchIndex > -1 {
			c.emit(nd, OpFinalizer, 0)
		}

		c.emit(nd, OpReturn, 1)
	}
	return nil
}

func (c *Compiler) compileForStmt(stmt *node.ForStmt) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// init statement
	if stmt.Init != nil {
		if err := c.Compile(stmt.Init); err != nil {
			return err
		}
	}

	// pre-condition position
	preCondPos := len(c.instructions)

	// condition expression
	postCondPos := -1
	if stmt.Cond != nil {
		if err := c.Compile(stmt.Cond); err != nil {
			return err
		}
		// condition jump position
		postCondPos = c.emit(stmt, OpJumpFalsy, 0)
	}

	// enter loop
	loop := c.enterLoop()

	// body statement
	if err := c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.instructions)

	// post statement
	if stmt.Post != nil {
		if err := c.Compile(stmt.Post); err != nil {
			return err
		}
	}

	// back to condition
	c.emit(stmt, OpJump, preCondPos)

	// post-statement position
	postStmtPos := len(c.instructions)
	if postCondPos >= 0 {
		c.changeOperand(postCondPos, postStmtPos)
	}

	// update all break/continue jump positions
	for _, pos := range loop.breaks {
		c.changeOperand(pos, postStmtPos)
	}

	for _, pos := range loop.continues {
		c.changeOperand(pos, postBodyPos)
	}
	return nil
}

func (c *Compiler) compileForInStmt(stmt *node.ForInStmt) error {
	c.symbolTable = c.symbolTable.Fork(true)
	defer func() {
		c.symbolTable = c.symbolTable.Parent(false)
	}()

	// for-in statement is compiled like following:
	//   when ARG is iterable:
	//     :it := iterator(ARG)
	//     for :it.next()  {
	//       k, v := :it.get()  // set locals
	//
	//       ... body ...
	//     }
	//     :it.endLoop()
	//
	//   when ARG is iterator:
	//     :it := ARG
	//     for :it.next()  {
	//       k, v := :it.get()  // set locals
	//
	//       ... body ...
	//     }
	//
	// ":it" is a local variable but it will not conflict with other user variables
	// because character ":" is not allowed in the variable names.

	// init
	//   :it = iterator(iterable)
	itSymbol, exists := c.symbolTable.DefineLocal(":it")
	if exists {
		return c.errorf(stmt, ":it redeclared in this block")
	}

	if err := c.Compile(stmt.Iterable); err != nil {
		return err
	}

	c.emit(stmt, OpIterInit)
	c.emit(stmt, OpDefineLocal, itSymbol.Index)

	var (
		iterNextElsePos,
		truePos,
		falsePos int
	)

	if stmt.Else != nil {
		c.emit(stmt, OpGetLocal, itSymbol.Index)
		iterNextElsePos = c.emit(stmt, OpIterNextElse, 0, 0)
	}

	// pre-condition position
	preCondPos := len(c.instructions)

	// condition
	//  :it.Next()
	c.emit(stmt, OpGetLocal, itSymbol.Index)
	c.emit(stmt, OpIterNext)

	// condition jump position
	postCondPos := c.emit(stmt, OpJumpFalsy, 0)

	if stmt.Else != nil {
		truePos = len(c.instructions)
		defer func() {
			c.changeOperand(iterNextElsePos, truePos, falsePos)
		}()
	}

	// enter loop
	loop := c.enterLoop()

	// assign key variable
	if stmt.Key.Name != "_" {
		keySymbol, exists := c.symbolTable.DefineLocal(stmt.Key.Name)
		if exists {
			return c.errorf(stmt, "%q redeclared in this block", stmt.Key.Name)
		}
		c.emit(stmt, OpGetLocal, itSymbol.Index)
		c.emit(stmt, OpIterKey)
		keySymbol.Assigned = true
		c.emit(stmt, OpDefineLocal, keySymbol.Index)
	}

	// assign value variable
	if stmt.Value.Name != "_" {
		valueSymbol, exists := c.symbolTable.DefineLocal(stmt.Value.Name)
		if exists {
			return c.errorf(stmt, "%q redeclared in this block", stmt.Value.Name)
		}
		c.emit(stmt, OpGetLocal, itSymbol.Index)
		c.emit(stmt, OpIterValue)
		valueSymbol.Assigned = true
		c.emit(stmt, OpDefineLocal, valueSymbol.Index)
	}

	// body statement
	if err := c.Compile(stmt.Body); err != nil {
		c.leaveLoop()
		return err
	}

	c.leaveLoop()

	// post-body position
	postBodyPos := len(c.instructions)

	// back to condition
	c.emit(stmt, OpJump, preCondPos)

	// else stmt
	if stmt.Else != nil {
		falsePos = len(c.instructions)
		if err := c.Compile(stmt.Else); err != nil {
			return err
		}
	}

	// post-statement position
	postStmtPos := len(c.instructions)
	c.changeOperand(postCondPos, postStmtPos)

	// update all break/continue jump positions
	for _, pos := range loop.breaks {
		c.changeOperand(pos, postStmtPos)
	}

	for _, pos := range loop.continues {
		c.changeOperand(pos, postBodyPos)
	}
	return nil
}

func (c *Compiler) compileFuncStmt(nd *node.FuncStmt) (err error) {
	if nd.Func.Type.NameExpr == nil {
		return c.errorf(nd, "func stmt require ident")
	}

	if err = c.Compile(&node.CallExpr{
		Func: node.EIdent(TFunc.Name(), nd.Pos()),
		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{
				Values: []node.Expr{nd.Func},
			},
		},
	}); err != nil {
		return
	}

	return c.compileDefineAssign(nd, nd.Func.Type.NameExpr, token.Const, token.Define, false)
}

func (c *Compiler) compileFuncExpr(nd *node.FuncExpr) error {
	body := nd.Body
	if body == nil {
		if nd.BodyExpr == nil {
			return c.errorf(nd, "func does not have body or body expression")
		}
		body = &node.BlockStmt{
			Stmts: []node.Stmt{
				&node.ReturnStmt{
					Return: node.Return{
						ReturnPos: nd.BodyExpr.Pos(),
						Result:    nd.BodyExpr,
					},
				},
			},
		}
	}

	return c.compileFunc(nd, nd.Type, body)
}

func (c *Compiler) compilePtr(nd *node.Ptr) (err error) {
	return c.errorf(nd, "compile %T is not implemented", nd)
}

func (c *Compiler) compileComputedExpr(nd *node.ComputedExpr) (err error) {
	stmts := nd.Stmts
	switch t := stmts[len(stmts)-1].(type) {
	case *node.IncDecStmt:
		stmts = append(stmts, &node.ReturnStmt{Return: node.Return{Result: t.Expr}})
	case *node.ExprStmt:
		stmts = append(stmts[:len(stmts)-1], &node.ReturnStmt{Return: node.Return{Result: t.Expr}})
	}
	if err = c.Compile(&node.FuncExpr{
		Body: &node.BlockStmt{Stmts: stmts},
	}); err != nil {
		return
	}
	c.emit(nd, OpComputedValue)
	return
}

func (c *Compiler) compileMethodExpr(nd *node.MethodExpr) error {
	var (
		nameExpr node.Expr
		methods  node.Exprs
	)

	switch t := nd.Expr.(type) {
	case *node.FuncExpr:
		nameExpr = t.Type.NameExpr
		oldType := *t.Type

		defer func() {
			*t.Type = oldType
		}()

		t.Type.NameExpr = nil
		methods = append(methods, t)
	case *node.FuncWithMethodsExpr:
		nameExpr = t.NameExpr
		methods = t.Funcs()
	}

	return c.compileAddMethodsExpr(nd, nameExpr, methods...)
}

func (c *Compiler) compileAddMethodsExpr(nd node.Node, nameExpr node.Expr, methods ...node.Expr) (err error) {
	defer c.pushSelector()()
	expr, selectors := resolveSelectorExprs(nameExpr)

	// `met module.NAME(...)` on a builtin namespace member (e.g. core.binOp)
	// resolves to the single qualified builtin so the method is added to the
	// same object the VM dispatches against, rather than to the namespace dict
	// member (which build() does not keep identical to the enum object).
	if ident, ok := expr.(*node.IdentExpr); ok && len(selectors) == 1 {
		if sel, _ := selectors[0].(*node.StrLit); sel != nil {
			if base, ok := c.symbolTable.Resolve(ident.Name); ok && base.Scope == ScopeBuiltin {
				if sym, ok := c.symbolTable.Resolve(ident.Name + "." + sel.Value()); ok &&
					sym.Scope == ScopeBuiltin {
					c.emit(nd, OpGetBuiltin, sym.Index)
					for _, method := range methods {
						if err = c.Compile(method); err != nil {
							return err
						}
					}
					c.emit(nd, OpAddMethod, 0, len(methods))
					return nil
				}
			}
		}
	}

	if err = c.Compile(expr); err != nil {
		return err
	}

	for _, selector := range selectors {
		if err = c.Compile(selector); err != nil {
			return err
		}
	}

	for _, method := range methods {
		if err = c.Compile(method); err != nil {
			return err
		}
	}

	c.emit(nd, OpAddMethod, len(selectors), len(methods))
	return nil
}

func (c *Compiler) compileClosureLit(nd *node.ClosureExpr) error {
	var stmts []node.Stmt
	if b, ok := nd.Body.(*node.BlockExpr); ok {
		stmts = b.Stmts
		if l := len(stmts); l > 0 {
			switch t := stmts[l-1].(type) {
			case *node.ExprStmt:
				stmts[l-1] = &node.ReturnStmt{
					Return: node.Return{
						ReturnPos: t.Pos(),
						Result:    t.Expr,
					},
				}
			}
		}
	} else {
		stmts = append(stmts, &node.ReturnStmt{Return: node.Return{Result: nd.Body}})
	}
	return c.compileFunc(nd, &node.FuncType{FuncHeader: node.FuncHeader{Params: nd.Params, Return: nd.Return}}, &node.BlockStmt{Stmts: stmts})
}

func (c *Compiler) compileFunc(nd ast.Node, typ *node.FuncType, body *node.BlockStmt) (err error) {
	var (
		params      []*Param
		namedParams []*NamedParam
		returnTypes []*ReturnVar
		st          = c.symbolTable.Fork(false)
	)

	if typ != nil {
		for _, ident := range typ.Params.Args.Values {
			p := &Param{}
			if p.Name, p.TypesSymbols, err = c.nameSymbolsOfTypedIdent(nd, ident); err != nil {
				return
			}
			params = append(params, p)
		}

		if typ.Params.Args.Var != nil {
			p := &Param{Var: true}
			if p.Name, p.TypesSymbols, err = c.nameSymbolsOfTypedIdent(nd, typ.Params.Args.Var); err != nil {
				return
			}
			params = append(params, p)
		}

		for i, name := range typ.Params.NamedArgs.Names {
			p := &NamedParam{}
			if p.Name, p.TypesSymbols, err = c.nameSymbolsOfTypedIdent(nd, name); err != nil {
				return
			}

			if v := typ.Params.NamedArgs.Values[i]; v != nil {
				p.Value = v.String()
			}

			namedParams = append(namedParams, p)
		}

		if typ.Params.NamedArgs.Var != nil {
			p := &NamedParam{
				Name:         typ.Params.NamedArgs.Var.Name,
				Var:          true,
				TypesSymbols: make(ParamType, 0),
			}
			namedParams = append(namedParams, p)
		}

		if err = st.DefineParams(NewParams(params...), NewNamedParams(namedParams...)); err != nil {
			return
		}

		if count := len(typ.Params.NamedArgs.Values); count > 0 {
			body.Stmts = append(c.helperBuildKwargsStmts(count, func(index int) (name *node.IdentExpr, value node.Expr) {
				ident := typ.Params.NamedArgs.Names[index].Ident
				return ident, typ.Params.NamedArgs.Values[index]
			}), body.Stmts...)
		}

		if returnTypes, err = c.returnTypesOf(nd, typ.Return); err != nil {
			return
		}
	}

	fork := c.fork(c.file, c.moduleMap, st)

	if typ != nil {
		fork.variadic = typ.Params.Args.Var != nil
	}

	// desugar `defer` by wrapping the body in the defer runner; mark the
	// original body claimed so the inner $__body thunk is not re-wrapped
	if !body.DeferClaimed && stmtsHaveDefer(body.Stmts) {
		body.DeferClaimed = true
		if body, err = c.wrapDeferBody(body); err != nil {
			return err
		}
	}

	if err := fork.Compile(body); err != nil {
		return err
	}
	freeSymbols := fork.symbolTable.FreeSymbols()
	for _, s := range freeSymbols {
		switch s.Scope {
		case ScopeLocal:
			c.emit(nd, OpGetLocalPtr, s.Index)
		case ScopeFree:
			c.emit(nd, OpGetFreePtr, s.Index)
		}
	}

	bc := fork.Bytecode()
	bc.Main.module = c.module
	bc.Main.ReturnVars = returnTypes

	if typ != nil {
		if typ.NameExpr != nil {
			bc.Main.FuncName = typ.Name()
		}
	}

	if bc.Main.NumLocals > 256 {
		return c.error(nd, ErrSymbolLimit)
	}

	c.constants = bc.Constants

	if len(freeSymbols) > 0 {
		bc.Main.AllowMethods = false
	}

	index := c.addConstant(bc.Main)

	if bc.Main.FuncName == "" {
		bc.Main.FuncName = c.newAnonymousFuncName()
	}

	if len(freeSymbols) > 0 {
		c.emit(nd, OpClosure, index, len(freeSymbols))
	} else {
		c.emit(nd, OpConstant, index)
	}
	return nil
}

func (c *Compiler) compileFuncWithMethodsStmt(nd *node.FuncWithMethodsStmt) error {
	var (
		args = make(node.Exprs, len(nd.Methods)+1)
		name *node.IdentExpr
	)

	if nd.NameExpr != nil {
		name, _ = nd.NameExpr.(*node.IdentExpr)
		if name == nil {
			return c.errorf(nd, "require NameExpr as *Ident")
		}
		args[0] = node.Str(name.String(), name.NamePos)
	} else {
		args[0] = node.Str("", 0)
	}

	if len(nd.Methods) == 0 {
		return c.errorf(nd, "funcWithMethods does not have methods")
	}

	for i, m := range nd.Methods {
		args[i+1] = m.Func()
	}

	call := &node.CallExpr{
		Func: node.EIdent(BuiltinFunc.String(), nd.Pos()),

		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{
				Values: args,
			},
		},
	}

	return c.Compile(&node.DeclStmt{
		Decl: &node.GenDecl{
			Tok: token.Const,
			Specs: []node.Spec{
				&node.ValueSpec{
					Idents: []*node.IdentExpr{name},
					Values: []node.Expr{call},
				},
			},
		},
	})
}

func (c *Compiler) compileFuncWithMethodsExpr(nd *node.FuncWithMethodsExpr) error {
	if len(nd.Methods) == 0 {
		return c.errorf(nd, "funcWithMethods does not have methods")
	}

	args := make(node.Exprs, len(nd.Methods)+1)

	if nd.NameExpr != nil {
		name, _ := nd.NameExpr.(*node.IdentExpr)
		if name == nil {
			return c.errorf(nd, "require NameExpr as *Ident")
		}
		args[0] = node.Str(name.String(), name.NamePos)
	} else {
		args[0] = node.Str("", 0)
	}

	for i, m := range nd.Methods {
		args[i+1] = m.Func()
	}

	call := &node.CallExpr{
		Func: node.EIdent(BuiltinFunc.String(), nd.Pos()),

		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{
				Values: args,
			},
		},
	}

	return c.Compile(call)
}

// propCallExpr builds the `Prop(name, methods...)` constructor call that a
// `prop` statement or expression compiles to. The accessor methods are lowered
// to function literals, exactly like func-with-methods.
func (c *Compiler) propCallExpr(nd *node.PropExpr) (*node.CallExpr, error) {
	if len(nd.Methods) == 0 {
		return nil, c.errorf(nd, "prop does not have methods")
	}

	args := make(node.Exprs, len(nd.Methods)+1)

	if nd.NameExpr != nil {
		name, _ := nd.NameExpr.(*node.IdentExpr)
		if name == nil {
			return nil, c.errorf(nd, "require NameExpr as *Ident")
		}
		args[0] = node.Str(name.String(), name.NamePos)
	} else {
		args[0] = node.Str("", 0)
	}

	for i, m := range nd.Methods {
		args[i+1] = m.Func()
	}

	return &node.CallExpr{
		Func: node.EIdent(BuiltinProp.String(), nd.Pos()),
		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{
				Values: args,
			},
		},
	}, nil
}

func (c *Compiler) compilePropStmt(nd *node.PropStmt) error {
	// An anonymous prop statement has nothing to bind to: evaluate it as an
	// expression statement.
	if nd.NameExpr == nil {
		return c.compilePropExpr(&nd.PropExpr)
	}

	name, _ := nd.NameExpr.(*node.IdentExpr)
	if name == nil {
		return c.errorf(nd, "require NameExpr as *Ident")
	}

	call, err := c.propCallExpr(&nd.PropExpr)
	if err != nil {
		return err
	}

	return c.Compile(&node.DeclStmt{
		Decl: &node.GenDecl{
			Tok: token.Const,
			Specs: []node.Spec{
				&node.ValueSpec{
					Idents: []*node.IdentExpr{name},
					Values: []node.Expr{call},
				},
			},
		},
	})
}

func (c *Compiler) compilePropExpr(nd *node.PropExpr) error {
	call, err := c.propCallExpr(nd)
	if err != nil {
		return err
	}
	return c.Compile(call)
}

// methodInterfaceCallExpr builds the MethodInterface(name, headers...) call a
// `meti` expression lowers to.
func (c *Compiler) methodInterfaceCallExpr(nd *node.MethodInterfaceExpr) *node.CallExpr {
	var name string
	if id := nd.NameIdent(); id != nil {
		name = id.Name
	}
	args := make(node.Exprs, 0, len(nd.Headers)+1)
	args = append(args, node.Str(name, nd.Pos()))
	for _, h := range nd.Headers {
		args = append(args, h)
	}
	return &node.CallExpr{
		Func:     node.EIdent(BuiltinMethodInterface.String(), nd.Pos()),
		CallArgs: node.CallArgs{Args: node.CallExprPositionalArgs{Values: args}},
	}
}

func (c *Compiler) compileMethodInterfaceExpr(nd *node.MethodInterfaceExpr) error {
	return c.Compile(c.methodInterfaceCallExpr(nd))
}

func (c *Compiler) compileMethodInterfaceStmt(nd *node.MethodInterfaceStmt) error {
	// an anonymous `meti { … }` statement is just an expression statement
	if nd.NameExpr == nil {
		return c.compileMethodInterfaceExpr(&nd.MethodInterfaceExpr)
	}
	name, _ := nd.NameExpr.(*node.IdentExpr)
	if name == nil {
		return c.errorf(nd, "require NameExpr as *Ident")
	}
	// `meti Name { … }` -> `const Name = MethodInterface("Name", …)`
	return c.Compile(&node.DeclStmt{
		Decl: &node.GenDecl{
			Tok: token.Const,
			Specs: []node.Spec{
				&node.ValueSpec{
					Idents: []*node.IdentExpr{name},
					Values: []node.Expr{c.methodInterfaceCallExpr(&nd.MethodInterfaceExpr)},
				},
			},
		},
	})
}

// compileFuncHeaderExpr lowers a `<(params) <return>>` header value to a
// FunctionHeader(name, params, namedParams, return) constructor call, where
// each parameter/return is a typedIdent.
func (c *Compiler) compileFuncHeaderExpr(nd *node.FuncHeaderExpr) error {
	typedIdents := func(idents ...*node.TypedIdentExpr) node.Exprs {
		out := make(node.Exprs, 0, len(idents))
		for _, ti := range idents {
			if ti != nil {
				out = append(out, ti)
			}
		}
		return out
	}

	pos, end := nd.Pos(), nd.End()
	params := typedIdents(nd.Params.Args.Values...)
	if nd.Params.Args.Var != nil {
		params = append(params, nd.Params.Args.Var)
	}
	named := typedIdents(nd.Params.NamedArgs.Names...)
	ret := typedIdents(nd.Return...)

	return c.Compile(&node.CallExpr{
		Func: node.EIdent(BuiltinFunctionHeader.String(), pos),
		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{
				Values: []node.Expr{
					node.Str(nd.Name(), pos),
					node.Array(pos, end, params...),
					node.Array(pos, end, named...),
					node.Array(pos, end, ret...),
				},
			},
		},
	})
}

func (c *Compiler) compileLogical(nd *node.BinaryExpr) error {
	// left side term
	if err := c.Compile(nd.LHS); err != nil {
		return err
	}

	// jump position
	var jumpPos int
	switch nd.Token {
	case token.LAnd:
		jumpPos = c.emit(nd, OpAndJump, 0)
	case token.Nullich:
		jumpPos = c.emit(nd, OpJumpNotNil, 0)
	default:
		jumpPos = c.emit(nd, OpOrJump, 0)
	}

	// right side term
	if err := c.Compile(nd.RHS); err != nil {
		return err
	}
	c.changeOperand(jumpPos, len(c.instructions))
	return nil
}

// asRangeExpr unwraps parentheses and returns the underlying `..` BinaryExpr,
// or nil when e is not a range expression.
func asRangeExpr(e node.Expr) *node.BinaryExpr {
	for {
		switch t := e.(type) {
		case *node.ParenExpr:
			e = t.Expr
		case *node.BinaryExpr:
			if t.Token == token.DotDot {
				return t
			}
			return nil
		default:
			return nil
		}
	}
}

// rangeCallExpr builds the `Range(from, to[; step=step])` call that a `..`
// expression compiles to.
func rangeCallExpr(from, to, step node.Expr) *node.CallExpr {
	call := &node.CallExpr{Func: &node.IdentExpr{Name: "Range"}}
	call.CallArgs.Args.Values = []node.Expr{from, to}
	if step != nil {
		call.CallArgs.NamedArgs.AppendS("step", step)
	}
	return call
}

func (c *Compiler) compileBinaryExpr(nd *node.BinaryExpr) error {
	// `from .. to` is sugar for `Range(from, to)`; `(from .. to) / step` (and the
	// equivalent `from .. to / step`) is `Range(from, to; step=step)`.
	if nd.Token == token.DotDot {
		return c.Compile(rangeCallExpr(nd.LHS, nd.RHS, nil))
	}
	if nd.Token == token.Quo {
		if rng := asRangeExpr(nd.LHS); rng != nil {
			return c.Compile(rangeCallExpr(rng.LHS, rng.RHS, nd.RHS))
		}
	}

	if nd.Token == token.Pipe {
		var call node.CallExpr
		switch t := nd.RHS.(type) {
		case *node.CallExpr:
			call = *t
		default:
			call = node.CallExpr{
				Func: t,
			}
		}
		call.CallArgs.Args.Values = append([]node.Expr{nd.LHS}, call.CallArgs.Args.Values...)
		return c.Compile(&call)
	}

	if err := c.Compile(nd.LHS); err != nil {
		return err
	}

	if err := c.Compile(nd.RHS); err != nil {
		return err
	}

	switch nd.Token {
	case token.Equal:
		c.emit(nd, OpEqual)
	case token.NotEqual:
		c.emit(nd, OpNotEqual)
	case token.In:
		c.emit(nd, OpIn)
	default:
		if !nd.Token.IsBinaryOperator() {
			return c.errorf(nd, "invalid binary operator: %s",
				nd.Token.String())
		}
		c.emit(nd, OpBinary, int(nd.Token))
	}
	return nil
}

func (c *Compiler) compileUnaryExpr(nd *node.UnaryExpr) error {
	// prefix `++x` / `--x` mutate the operand and yield its new value
	if nd.Token == token.Inc || nd.Token == token.Dec {
		return c.compilePrefixIncDec(nd)
	}
	if isMain, _ := nd.Expr.(*node.IsMainLit); isMain != nil && nd.Token == token.Not {
		c.emit(nd, OpNotIsMain)
	} else if err := c.Compile(nd.Expr); err != nil {
		return err
	} else {
		switch nd.Token {
		case token.Not, token.Sub, token.Xor, token.Add:
			c.emit(nd, OpUnary, int(nd.Token))
		case token.Null:
			c.emit(nd, OpIsNil)
		case token.NotNull:
			c.emit(nd, OpNotIsNil)
		default:
			return c.errorf(nd,
				"invalid unary operator: %s", nd.Token.String())
		}
	}
	return nil
}

// compilePrefixIncDec compiles `++ident` / `--ident`: apply the unary
// increment/decrement operator to the variable's current value, store the
// result back and leave the new value on the stack.
func (c *Compiler) compilePrefixIncDec(nd *node.UnaryExpr) error {
	ident, _ := nd.Expr.(*node.IdentExpr)
	if ident == nil {
		return c.errorf(nd, "operator %q requires a variable operand", nd.Token.String())
	}
	// load current value, apply the operator
	if err := c.Compile(ident); err != nil {
		return err
	}
	c.emit(nd, OpUnary, int(nd.Token))
	// store the new value back into the variable (consumes it)
	if err := c.compileDefineAssign(nd, ident, token.Var, token.Assign, false); err != nil {
		return err
	}
	// yield the new value
	return c.Compile(ident)
}

func (c *Compiler) compileSelectorExpr(nd *node.SelectorExpr) error {
	defer c.pushSelector()()
	expr, selectors := resolveSelectorExprs(nd)

	// Builtin module member access: `module.NAME`, where `module` is a builtin
	// module namespace and `module.NAME` is a registered qualified builtin,
	// compiles to a single OpGetBuiltin instead of loading the namespace dict
	// and indexing it. A shadowing local/global `module` disables this.
	if ident, ok := expr.(*node.IdentExpr); ok && len(selectors) == 1 {
		if sel, _ := selectors[0].(*node.StrLit); sel != nil {
			if base, ok := c.symbolTable.Resolve(ident.Name); ok && base.Scope == ScopeBuiltin {
				if sym, ok := c.symbolTable.Resolve(ident.Name + "." + sel.Value()); ok &&
					sym.Scope == ScopeBuiltin {
					c.emit(nd, OpGetBuiltin, sym.Index)
					return nil
				}
			}
		}
	}

	if err := c.Compile(expr); err != nil {
		return err
	}
	for _, selector := range selectors {
		if err := c.Compile(selector); err != nil {
			return err
		}
	}
	c.emit(nd, OpGetIndex, len(selectors))
	return nil
}

func (c *Compiler) pushSelector() func() {
	var (
		increases bool
		stackLen  = len(c.stack)
	)
	switch c.stack[stackLen-2].(type) {
	case *node.SelectorExpr, *node.NullishSelectorExpr:
	default:
		increases = true
		c.selectorStack = append(c.selectorStack, nil)
	}
	i := len(c.selectorStack) - 1
	j := len(c.selectorStack[i])
	c.selectorStack[i] = append(c.selectorStack[i], nil)
	return func() {
		for _, f := range c.selectorStack[i][j] {
			f()
		}
		c.selectorStack[i] = c.selectorStack[i][:j]
		if increases {
			c.selectorStack = c.selectorStack[:i]
		}
	}
}

func (c *Compiler) selectorHandler(f func()) {
	l := len(c.selectorStack) - 1
	c.selectorStack[l][0] = append(c.selectorStack[l][0], f)
}

func (c *Compiler) compileNullishSelectorExpr(nd *node.NullishSelectorExpr) error {
	defer c.pushSelector()()

	expr, selectors := resolveSelectorExprs(nd)

	var jumpPos int

	if err := c.Compile(expr); err != nil {
		return err
	}

	for _, selector := range selectors[0 : len(selectors)-1] {
		if err := c.Compile(selector); err != nil {
			return err
		}
	}

	jumpPos = c.emit(nd, OpJumpNil, 0)
	c.selectorHandler(func() {
		c.changeOperand(jumpPos, len(c.instructions))
	})

	if err := c.Compile(selectors[len(selectors)-1]); err != nil {
		return err
	}
	c.emit(nd, OpGetIndex, len(selectors))
	return nil
}

func resolveSelectorExprs(nd node.Expr) (expr node.Expr, selectors []node.Expr) {
	expr = nd
	switch v := nd.(type) {
	case *node.SelectorExpr:
		expr, selectors = resolveIndexExprs(v.X)
		selectors = append(selectors, v.Sel)
	case *node.NullishSelectorExpr:
		expr, selectors = resolveIndexExprs(v.Expr)
		selectors = append(selectors, v.Sel)
	}
	return
}

func (c *Compiler) compileIndexExpr(nd *node.IndexExpr) error {
	expr, indexes := resolveIndexExprs(nd)
	if err := c.Compile(expr); err != nil {
		return err
	}
	for _, index := range indexes {
		if err := c.Compile(index); err != nil {
			return err
		}
	}
	c.emit(nd, OpGetIndex, len(indexes))
	return nil
}

func resolveIndexExprs(nd node.Expr) (expr node.Expr, indexes []node.Expr) {
	expr = nd
	if v, ok := nd.(*node.IndexExpr); ok {
		expr, indexes = resolveIndexExprs(v.X)
		indexes = append(indexes, v.Index)
	}
	return
}

func (c *Compiler) compileSliceExpr(nd *node.SliceExpr) error {
	if err := c.Compile(nd.Expr); err != nil {
		return err
	}

	if nd.Low != nil {
		if err := c.Compile(nd.Low); err != nil {
			return err
		}
	} else {
		c.emit(nd, OpNil)
	}

	if nd.High != nil {
		if err := c.Compile(nd.High); err != nil {
			return err
		}
	} else {
		c.emit(nd, OpNil)
	}

	c.emit(nd, OpSliceIndex)
	return nil
}

func (c *Compiler) compileCallExpr(nd *node.CallExpr) error {
	var (
		selExpr    *node.SelectorExpr
		isSelector bool
		op         = OpCall
	)

	if nd.Func != nil {
		selExpr, isSelector = nd.Func.(*node.SelectorExpr)
	}
	if isSelector {
		if err := c.Compile(selExpr.X); err != nil {
			return err
		}
		op = OpCallName
	} else {
		if err := c.Compile(nd.Func); err != nil {
			return err
		}
	}

	return c.compileCallArgs(nd.CallPos(), op, &nd.CallArgs, selExpr)
}

func (c *Compiler) compileCallArgs(pos source.Pos, op Opcode, nd *node.CallArgs, selExpr *node.SelectorExpr) error {
	var (
		flags   OpCallFlag
		numArgs = len(nd.Args.Values)
	)

	for _, arg := range nd.Args.Values {
		if err := c.Compile(arg); err != nil {
			return err
		}
	}

	if nd.Args.Var != nil {
		numArgs++
		flags |= OpCallFlagVarArgs
		if err := c.Compile(nd.Args.Var.Value); err != nil {
			return err
		}
	}

	if numKwargs := len(nd.NamedArgs.Names); numKwargs > 0 {
		flags |= OpCallFlagNamedArgs
		namedArgs := &node.KeyValueArrayLit{Elements: make([]node.Expr, numKwargs)}

		for i, name := range nd.NamedArgs.Names {
			value := nd.NamedArgs.Values[i]
			if name.Var {
				namedArgs.Elements[i] = &node.NamedArgVarLit{Value: name.Exp}
			} else {
				if value == nil {
					// is flag
					value = &node.FlagLit{Value: true}
				}
				if name.Exp != nil {
					namedArgs.Elements[i] = &node.KeyValueLit{Key: name.Expr(), Value: value}
				} else {
					namedArgs.Elements[i] = &node.KeyValuePairLit{Key: name.Expr(), Value: value}
				}
			}
		}

		if err := c.compileKeyValueArrayLit(namedArgs); err != nil {
			return err
		}
	}

	if selExpr != nil {
		if err := c.Compile(selExpr.Sel); err != nil {
			return err
		}
	}

	c.emitPos(pos, nd, op, numArgs, int(flags))
	return nil
}

func (c *Compiler) compileFile(nd *parser.File) (err error) {
	return c.compileFileStmts(nd.Stmts)
}

func (c *Compiler) compileFileStmts(stmts node.Stmts) (err error) {
	var paramsNames []string

	for _, stmt := range stmts {
		switch stmt := stmt.(type) {
		case *node.DeclStmt:
			if g, _ := stmt.Decl.(*node.GenDecl); g != nil {
				if g.Tok == token.Param {
					// puts exports dict after param decl
					positional, named := g.Params()
					for _, spec := range positional {
						paramsNames = append(paramsNames, spec.Ident.Ident.Name)
					}
					for _, spec := range named {
						paramsNames = append(paramsNames, spec.Ident.Ident.Name)
					}
					break
				}
			}
		}
	}

	if len(paramsNames) > 0 {
		if _, err = c.symbolTable.defineParamsVar(paramsNames); err != nil {
			return
		}
	}

	return c.compileStmts(stmts...)
}

func (c *Compiler) defineModule(module *ModuleSpec) *storeItem {
	return c.addModule(1, module)
}

func (c *Compiler) CompileModule(nd *ModuleStmt) (err error) {
	c.defineModule(nd.Module)
	c.module = nd.Module
	return c.compileFileStmts(nd.Stmts)
}

func (c *Compiler) compileImportExpr(nd *node.ImportExpr) (err error) {
	moduleName, args := nd.Build()
	if moduleName == "" {
		return c.errorf(nd, "empty module name")
	}

	var (
		p   = c
		pth []int
	)

	for p != nil {
		pth = append(pth, p.module.Index)
		p = p.parent
	}

	importer := c.moduleMap.Get(moduleName)
	if importer == nil {
		return c.errorf(nd, "module '%s' not found", moduleName)
	}

	extImp, isExt := importer.(ExtImporter)
	if isExt {
		if name, err := extImp.Name(); err != nil {
			return c.errorf(nd, "resolve name of module '%s': %v", moduleName, err.Error())
		} else if len(name) > 0 {
			moduleName = name
		}
	}

	if err = c.checkCyclicImports(nd, moduleName); err != nil {
		return
	}

	pth = pth[:len(pth)-1]
	slices.Reverse(pth)

	moduleStoreEntry, exists := c.getModule(moduleName)
	if !exists {
		spec := &ModuleSpec{
			ModuleInfo: ModuleInfo{
				Name: moduleName,
			},
			Path: pth,
		}
		mod, url, err := importer.Import(c.opts.Context, spec)
		if err != nil {
			return c.error(nd, err)
		}
		switch v := mod.(type) {
		case []byte:
			var moduleMap *ModuleMap
			if isExt {
				moduleMap = c.moduleMap.Fork(moduleName)
			} else {
				moduleMap = c.BaseModuleMap()
			}

			spec.URL = url

			err = c.compileModule(nd, importer, spec, moduleMap, v)
			if err != nil {
				return err
			}

			moduleStoreEntry = c.moduleStore.items[spec.Index]
		case ModuleInitFunc:
			spec.URL = url
			spec.InitGoFunc = v.Caller(spec)
			moduleStoreEntry = c.addModule(1, spec)
		case BuiltinCompileModuleFunc:
			var (
				st = NewSymbolTable(c.symbolTable.builtins).
					DisableBuiltin(c.symbolTable.DisabledBuiltins()...)
				fork = c.fork(c.file, c.moduleMap, st)
				bc   *Bytecode
			)

			spec.URL = url
			moduleStoreEntry = c.addModule(1, spec)
			fork.module = spec
			fork.file = nil

			if bc, err = v(&BuiltinCompileModuleContext{
				Compiler: fork,
				FileSet:  c.file.Set(),
				Spec:     spec,
			}); err != nil {
				return c.error(nd, err)
			}

			c.constants = bc.Constants
			spec.InitCompiledFunc = bc.Main
		default:
			return c.errorf(nd, "invalid import value type: %T", v)
		}
	}

	switch moduleStoreEntry.typ {
	case 1:
		// load module
		// if module is already stored, load from VM.modulesCache otherwise call compiled function
		// and store copy of result to VM.modulesCache.
		c.emit(nd, OpLoadModule, moduleStoreEntry.storeIndex)
		jumpPos := c.emit(nd, OpJumpFalsy, 0)

		if err := c.compileCallArgs(nd.CallPos(), OpInitModule, &args, nil); err != nil {
			return c.errorf(nd, "invalid init module args: %v", err)
		}

		c.changeOperand(jumpPos, len(c.instructions))
	case 2:
		// load module
		// if module is already stored, load from VM.modulesCache otherwise copy object
		// and store it to VM.modulesCache.
		c.emit(nd, OpLoadModule, moduleStoreEntry.storeIndex)
		jumpPos := c.emit(nd, OpJumpFalsy, 0)
		c.changeOperand(jumpPos, len(c.instructions))
	default:
		return c.errorf(nd, "invalid module type: %v", moduleStoreEntry.typ)
	}
	return nil
}

func (c *Compiler) compileEmbedExpr(nd *node.EmbedExpr) (err error) {
	pth := nd.Path()
	if pth == "" {
		return c.errorf(nd, "empty path")
	}

	importer := c.embeddedMap.Get(pth)
	if importer == nil {
		return c.errorf(nd, "path '%s' not found", pth)
	}

	var (
		name,
		absPath string
	)

	if extImp, _ := importer.(EmbeddedExtImporter); extImp != nil {
		var tempName string
		if tempName, absPath, err = extImp.Paths(); err == nil {
			name = tempName
		} else {
			// if not exists, try import using current name
			if !os.IsNotExist(err) {
				return c.errorf(nd, "resolve name of embed '%s': %v", pth, err.Error())
			}
			err = nil
			name = pth
		}
	} else {
		name = pth
	}

	constantIndex, _, exists := c.getEmbed(name)
	if !exists {
		opts := &EmbeddedImportOptions{
			Sources:    nd.Sources(),
			Includes:   nd.Includes(),
			Excludes:   nd.Excludes(),
			IncludesRe: nd.IncludesRe(),
			ExcludesRe: nd.ExcludesRe(),
			Tree:       nd.Tree(),
		}

		if configFile := nd.ConfigFile(); configFile != "" {
			if err = c.applyEmbedConfig(configFile, absPath, opts); err != nil {
				return c.error(nd, err)
			}
		}

		data, err := importer.Import(c.opts.Context, name, absPath, opts)
		if err != nil {
			return c.error(nd, err)
		}
		constantIndex = c.addEmbed(data)
	}

	c.emit(nd, OpConstant, constantIndex)
	return nil
}

type embedConfig struct {
	Sources    []string `yaml:"sources"`
	Includes   []string `yaml:"includes"`
	Excludes   []string `yaml:"excludes"`
	IncludesRe []string `yaml:"includes_re"`
	ExcludesRe []string `yaml:"excludes_re"`
	Tree       bool     `yaml:"tree"`
}

func (c *Compiler) applyEmbedConfig(configFile, absPath string, opts *EmbeddedImportOptions) error {
	var pth string
	if filepath.IsAbs(configFile) {
		pth = configFile
	} else if absPath != "" && filepath.IsAbs(absPath) {
		pth = filepath.Join(filepath.Dir(absPath), configFile)
	} else {
		pth = configFile
	}
	data, err := os.ReadFile(pth)
	if err != nil {
		return err
	}

	var cfg embedConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if len(opts.Sources) == 0 && len(cfg.Sources) > 0 {
		opts.Sources = cfg.Sources
	}
	if len(opts.Includes) == 0 && len(cfg.Includes) > 0 {
		opts.Includes = cfg.Includes
	}
	if len(opts.Excludes) == 0 && len(cfg.Excludes) > 0 {
		opts.Excludes = cfg.Excludes
	}
	if len(opts.IncludesRe) == 0 && len(cfg.IncludesRe) > 0 {
		opts.IncludesRe = cfg.IncludesRe
	}
	if len(opts.ExcludesRe) == 0 && len(cfg.ExcludesRe) > 0 {
		opts.ExcludesRe = cfg.ExcludesRe
	}
	if !opts.Tree && cfg.Tree {
		opts.Tree = cfg.Tree
	}

	return nil
}

func (c *Compiler) compileCondExpr(nd *node.CondExpr) error {
	if v, ok := nd.Cond.(node.BoolExpr); ok {
		if v.Bool() {
			return c.Compile(nd.True)
		}
		return c.Compile(nd.False)
	}

	op := OpJumpFalsy
	if v, ok := simplifyExpr(nd.Cond).(*node.UnaryExpr); ok && v.Token.Is(token.Null, token.NotNull) {
		if err := c.Compile(v.Expr); err != nil {
			return err
		}

		op = OpJumpNotNil
		if v.Token == token.NotNull {
			op = OpJumpNil
		}
	} else if err := c.Compile(nd.Cond); err != nil {
		return err
	}

	// first jump placeholder
	jumpPos1 := c.emit(nd, op, 0)
	if err := c.Compile(nd.True); err != nil {
		return err
	}

	// second jump placeholder
	jumpPos2 := c.emit(nd, OpJump, 0)

	// update first jump offset
	curPos := len(c.instructions)
	c.changeOperand(jumpPos1, curPos)
	if err := c.Compile(nd.False); err != nil {
		return err
	}
	// update second jump offset
	curPos = len(c.instructions)
	c.changeOperand(jumpPos2, curPos)
	return nil
}

func (c *Compiler) compileTemplateLit(nd *node.TemplateLit) error {
	var tmplValue string
	switch t := nd.Value.(type) {
	case *node.StrLit:
		tmplValue = t.Value()
	case *node.RawStrLit:
		tmplValue = t.Value()
	case *node.RawHeredocLit:
		// Parse the untrimmed body so interpolation positions map to source;
		// Build re-applies the heredoc indentation stripping.
		tmplValue = t.RawContent()
	case *node.HeredocLit:
		// Parse the untrimmed, un-escaped body so interpolation positions map to
		// source; Build re-applies indentation stripping and escape processing.
		tmplValue = t.RawContent()
	case *node.SymbolLit:
		tmplValue = t.Value()
	default:
		return c.errorf(nd, "expected string for template literal")
	}

	file, err := parser.ParseTemplateString(tmplValue, nd.StringValuePos())
	if err != nil {
		return c.errorf(nd, "template parse error: %w", err)
	}

	expr, err := nd.Build(file.Stmts)
	if err != nil {
		return c.errorf(nd, "template build error: %w", err)
	}

	return c.Compile(expr)
}

func (c *Compiler) compileIdent(nd *node.IdentExpr) error {
	symbol, ok := c.symbolTable.Resolve(nd.Name)
	if !ok {
		if c.iotaVal < 0 || nd.Name != "iota" {
			return c.errorf(nd, "unresolved reference %q", nd.Name)
		}
		c.emit(nd, OpConstant, c.addConstant(Int(c.iotaVal)))
		return nil
	}

	switch symbol.Scope {
	case ScopeGlobal:
		c.emit(nd, OpGetGlobal, symbol.Index)
	case ScopeLocal:
		c.emit(nd, OpGetLocal, symbol.Index)
	case ScopeBuiltin:
		c.emit(nd, OpGetBuiltin, symbol.Index)
	case ScopeFree:
		c.emit(nd, OpGetFree, symbol.Index)
	}
	return nil
}

func (c *Compiler) compileArrayLit(nd *node.ArrayExpr) error {
	var hasSpread bool
	for _, elem := range nd.Elements {
		if _, ok := elem.(*node.ArgVarLit); ok {
			hasSpread = true
			break
		}
	}

	if !hasSpread {
		for _, elem := range nd.Elements {
			if err := c.Compile(elem); err != nil {
				return err
			}
		}
		c.emit(nd, OpArray, len(nd.Elements))
		return nil
	}

	// `[1, 2, *a, 4, *b]` merges by concatenation: runs of plain elements are
	// built with OpArray and joined to spread operands with `+`.
	var (
		run     []node.Expr
		emitted bool
	)
	flush := func() error {
		if len(run) == 0 {
			return nil
		}
		for _, e := range run {
			if err := c.Compile(e); err != nil {
				return err
			}
		}
		c.emit(nd, OpArray, len(run))
		if emitted {
			c.emit(nd, OpBinary, int(token.Add))
		}
		emitted = true
		run = run[:0]
		return nil
	}
	for _, elem := range nd.Elements {
		if av, ok := elem.(*node.ArgVarLit); ok {
			if err := flush(); err != nil {
				return err
			}
			if !emitted {
				// start from an empty array so the first spread is copied,
				// never aliased
				c.emit(nd, OpArray, 0)
				emitted = true
			}
			if err := c.Compile(av.Value); err != nil {
				return err
			}
			c.emit(nd, OpBinary, int(token.Add))
		} else {
			run = append(run, elem)
		}
	}
	if err := flush(); err != nil {
		return err
	}
	if !emitted {
		c.emit(nd, OpArray, 0)
	}
	return nil
}

func (c *Compiler) compileDictLit(nd *node.DictExpr) error {
	var hasSpread bool
	for _, elt := range nd.Elements {
		if elt.Spread != nil {
			hasSpread = true
			break
		}
	}

	if !hasSpread {
		for _, elt := range nd.Elements {
			// key
			if err := c.Compile(elt.BuildKeyExpr()); err != nil {
				return err
			}
			// value
			if err := c.Compile(elt.Value); err != nil {
				return err
			}
		}
		c.emit(nd, OpDict, len(nd.Elements)*2)
		return nil
	}

	// `{a:1, *b, c:2, *d}` merges by concatenation: runs of plain key/value
	// elements are built with OpDict and joined to spread operands with `+`
	// (later keys win).
	var (
		run     []*node.DictElementLit
		emitted bool
	)
	flush := func() error {
		if len(run) == 0 {
			return nil
		}
		for _, elt := range run {
			if err := c.Compile(elt.BuildKeyExpr()); err != nil {
				return err
			}
			if err := c.Compile(elt.Value); err != nil {
				return err
			}
		}
		c.emit(nd, OpDict, len(run)*2)
		if emitted {
			c.emit(nd, OpBinary, int(token.Add))
		}
		emitted = true
		run = run[:0]
		return nil
	}
	for _, elt := range nd.Elements {
		if elt.Spread != nil {
			if err := flush(); err != nil {
				return err
			}
			if !emitted {
				// start from an empty dict so the first spread is copied,
				// never aliased
				c.emit(nd, OpDict, 0)
				emitted = true
			}
			if err := c.Compile(elt.Spread); err != nil {
				return err
			}
			c.emit(nd, OpBinary, int(token.Add))
		} else {
			run = append(run, elt)
		}
	}
	if err := flush(); err != nil {
		return err
	}
	if !emitted {
		c.emit(nd, OpDict, 0)
	}
	return nil
}

func (c *Compiler) compileKeyValuePairLit(elt *node.KeyValuePairLit) (err error) {
	// key
	switch t := elt.Key.(type) {
	case *node.IdentExpr:
		c.emit(elt, OpConstant, c.addConstant(Str(t.Name)))
	default:
		if err = c.Compile(elt.Key); err != nil {
			return
		}
	}

	if elt.Value == nil {
		c.emit(elt, OpYes)
		c.emit(elt, OpKeyValue, 1) // 1 => with value
	} else if flag, _ := elt.Value.(*node.FlagLit); flag != nil {
		if flag.Value {
			c.emit(elt, OpYes)
			c.emit(elt, OpKeyValue, 1) // 1 => with value
		} else {
			c.emit(elt, OpKeyValue, 0) // 0 => without value
		}
	} else {
		if err = c.Compile(elt.Value); err != nil {
			return err
		}
		c.emit(elt, OpKeyValue, 1) // 1 => with value
	}
	return
}

func (c *Compiler) compileKeyValueLit(elt *node.KeyValueLit) (err error) {
	// key
	switch t := elt.Key.(type) {
	case *node.IdentExpr:
		c.emit(elt, OpConstant, c.addConstant(Str(t.Name)))
	default:
		if err = c.Compile(elt.Key); err != nil {
			return
		}
	}

	if flag, _ := elt.Value.(*node.FlagLit); flag != nil {
		if flag.Value {
			c.emit(elt, OpYes)
			c.emit(elt, OpKeyValue, 1) // 1 => with value
		} else {
			c.emit(elt, OpKeyValue, 0) // 0 => without value
		}
	} else {
		// value
		if elt.Value == nil {
			c.emit(elt, OpYes)
		} else if err = c.Compile(elt.Value); err != nil {
			return err
		}
		c.emit(elt, OpKeyValue, 1) // 1 => with value
	}
	return
}

func (c *Compiler) compileKeyValueArrayLit(nd *node.KeyValueArrayLit) (err error) {
	length := len(nd.Elements)
elems:
	for _, elt := range nd.Elements {
		switch t := elt.(type) {
		case *node.KeyValuePairLit:
			if flag, _ := t.Value.(*node.FlagLit); flag != nil {
				if !flag.Value {
					length--
					continue elems
				}
			}
			if err = c.compileKeyValuePairLit(t); err != nil {
				return
			}
		case *node.KeyValueLit:
			if flag, _ := t.Value.(*node.FlagLit); flag != nil {
				if !flag.Value {
					length--
					continue elems
				}
			}
			if err = c.compileKeyValueLit(t); err != nil {
				return
			}
		case *node.NamedArgVarLit:
			if err = c.Compile(t); err != nil {
				return
			}
		default:
			return c.error(t, node.NewExpectedError(t, &node.KeyValuePairLit{}, &node.KeyValueLit{}, &node.NamedArgVarLit{}))
		}
	}

	c.emit(nd, OpKeyValueArray, length)
	return nil
}

func (c *Compiler) compileTypedIdentExpr(nd *node.TypedIdentExpr) error {
	types := make(node.Exprs, len(nd.Type))
	for i, expr := range nd.Type {
		types[i] = expr.Expr
	}
	return c.compileCallExpr(&node.CallExpr{
		Func: node.EIdent(BuiltinTypedIdent.String(), nd.Pos()),
		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{
				Values: []node.Expr{
					node.Str(nd.Ident.Name, nd.Ident.NamePos),
					node.Array(nd.Pos(), nd.End(), types...),
				},
			},
		},
	})
}

func (c *Compiler) compileNamedArgVarLit(nd *node.NamedArgVarLit) (err error) {
	if err = c.Compile(nd.Value); err != nil {
		return
	}
	c.emit(nd, OpNamedParamsVar)
	return
}

func (c *Compiler) compileNamedParamValue(nd *namedParamValue) (err error) {
	if err = c.Compile(&nd.StrLit); err != nil {
		return
	}
	c.emit(nd, OpNamedParamValue)
	return nil
}
func (c *Compiler) compileExportStmt(nd *node.ExportStmt) (err error) {
	var (
		key   = nd.KeyExpr
		value = nd.ValueExpr
	)
	if key == nil {
		switch t := nd.ValueExpr.(type) {
		case *node.DictExpr, *node.ParenExpr:
			if err = c.Compile(t); err != nil {
				return
			}

			c.emit(nd, OpExtendModule)
			c.emit(nd, OpPop)
			return nil
		case *node.FuncWithMethodsExpr:
			if t.NameExpr == nil {
				return c.errorf(t, "*ExportStmt of value as %T require NameExpr field", t)
			}
			var ok bool
			if key, ok = t.NameExpr.(*node.IdentExpr); !ok {
				return c.errorf(t, "*ExportStmt of value as %T require NameExpr field as *Ident", t)
			}
		case *node.FuncExpr:
			if t.Type.NameExpr == nil {
				return c.errorf(t, "*ExportStmt of value as %T require NameExpr field", t)
			}
			var ok bool
			if key, ok = t.Type.NameExpr.(*node.IdentExpr); !ok {
				return c.errorf(t, "*ExportStmt of value as %T require NameExpr field as *Ident", t)
			}
		default:
			return c.errorf(t, "*ExportStmt of value must be *DictExpr | *ParenExpr | *FuncWithMethodsExpr | *FuncExpr")
		}
	}

	if ident, _ := key.(*node.IdentExpr); ident != nil {
		key = node.Str(ident.Name, ident.NamePos)
		if value == nil {
			value = ident
		}
	}

	if value == nil {
		return c.errorf(nd, "*ExportStmt require value")
	}

	ass := &node.AssignStmt{
		TokenPos: nd.Pos(),
		Token:    token.Assign,
		LHS:      []node.Expr{node.EIndex(node.LModule(nd.Pos()), key, nd.TokenPos, key.End())},
		RHS:      []node.Expr{value},
	}
	return c.Compile(ass)
}

func (c *Compiler) compileToRawExpr(nd *node.ToRaw) (err error) {
	e := nd.Expr
try:
	switch et := e.(type) {
	case *node.StrLit:
		c.emit(nd, OpConstant, c.addConstant(RawStr(et.Value())))
	case *node.RawStrLit:
		c.emit(nd, OpConstant, c.addConstant(RawStr(et.Value())))
	case *node.ParenExpr:
		e = et.Expr
		goto try
	default:
		if err = c.Compile(et); err != nil {
			return
		}
		c.emit(nd, OpToRawStr)
	}
	return
}

func (c *Compiler) helperBuildKwargsStmts(count int, get func(index int) (name *node.IdentExpr, value node.Expr)) (stmts []node.Stmt) {
	for i := 0; i < count; i++ {
		name, value := get(i)
		if value == nil {
			value = &node.NilLit{}
		} else if cv, _ := value.(*node.ComputedExpr); cv != nil {
			value = &node.CallExpr{Func: cv}
		}
		stmts = append(stmts, &node.AssignStmt{
			TokenPos: name.Pos(),
			Token:    token.NullichAssign,
			LHS:      []node.Expr{name},
			RHS:      []node.Expr{value},
		})
	}
	return
}

// returnTypesOf builds the compiled return-type list from the function type's
// return idents. A bare entry ("<int>") yields an anonymous return whose type
// is the ident itself; a typed entry ("<x int|bool>") yields a named return
// whose types come from the type list. Type names are resolved in the enclosing
// scope, mirroring parameter type resolution.
func (c *Compiler) returnTypesOf(nd ast.Node, rets []*node.TypedIdentExpr) (types []*ReturnVar, err error) {
	if len(rets) == 0 {
		return nil, nil
	}

	types = make([]*ReturnVar, len(rets))
	for i, ti := range rets {
		var (
			rt        = &ReturnVar{}
			typeNames []string
		)

		if len(ti.Type) == 0 {
			typeNames = []string{ti.Ident.Name}
		} else {
			rt.Name = ti.Ident.Name
			typeNames = make([]string, len(ti.Type))
			for j, t := range ti.Type {
				typeNames[j] = t.Ident().Name
			}
		}

		rt.TypesSymbols = make(ParamType, len(typeNames))
		for j, name := range typeNames {
			var symbol *Symbol
			if symbol, err = c.requireSymbol(nd, name); err != nil {
				return
			}
			rt.TypesSymbols[j] = &symbol.SymbolInfo
		}

		types[i] = rt
	}

	return
}

func (c *Compiler) nameSymbolsOfTypedIdent(nd ast.Node, ti *node.TypedIdentExpr) (name string, symbols []*SymbolInfo, err error) {
	name = ti.Ident.Name

	if len(ti.Type) == 0 {
		ti.Type = append(ti.Type, node.EType(node.EIdent("any", ti.Pos())))
	}

	symbols = make([]*SymbolInfo, len(ti.Type))

	for i2, t := range ti.Type {
		var symbol *Symbol
		if symbol, err = c.requireSymbol(nd, t.Ident().Name); err != nil {
			return
		}
		symbols[i2] = &symbol.SymbolInfo
	}

	return
}

func simplifyExpr(e node.Expr) node.Expr {
do:
	switch t := e.(type) {
	case *node.ParenExpr:
		switch t2 := t.Expr.(type) {
		case *node.ParenExpr, *node.UnaryExpr:
			e = t2
			goto do
		}
	}
	return e
}
