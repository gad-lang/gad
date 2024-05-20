// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

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
			c.emit(nd.Catch, OpNull)
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
		names     = make([]string, 0, len(nd.Specs))
		types     []ParamType
		namedSpec []node.Spec
	)

	for i, sp := range nd.Specs {
		if np, _ := sp.(*node.NamedParamSpec); np != nil {
			namedSpec = nd.Specs[i:]
			break
		} else {
			spec := sp.(*node.ParamSpec)
			names = append(names, spec.Ident.Ident.Name)
			if len(spec.Ident.Type) > 0 {
				symbols := make([]*SymbolInfo, len(spec.Ident.Type))
				for i2, name := range spec.Ident.Type {
					symbol, ok := c.symbolTable.Resolve(name.Name)
					if !ok {
						return c.errorf(nd, "unresolved reference %q", name)
					}
					symbols[i2] = &symbol.SymbolInfo
				}
				types[i] = symbols
			}

			if spec.Variadic {
				if c.variadic {
					return c.errorf(nd,
						"multiple variadic param declaration")
				}
				c.variadic = true
			}
		}
	}

	if err := c.symbolTable.SetParams(c.variadic, names, types); err != nil {
		return c.error(nd, err)
	}

	namedSpecCount := len(namedSpec)

	if namedSpecCount == 0 {
		return nil
	}

	named := make([]*NamedParam, len(namedSpec))

	for i, sp := range namedSpec {
		spec := sp.(*node.NamedParamSpec)
		if spec.Value == nil {
			namedSpecCount--
			if c.varNamedParams {
				return c.errorf(nd,
					"multiple variadic named param declaration")
			}
			named[i] = &NamedParam{Name: spec.Ident.Ident.Name}
			c.varNamedParams = true
		} else {
			np := NewNamedParam(spec.Ident.Ident.Name, spec.Value.String())
			np.Type = make([]*SymbolInfo, len(spec.Ident.Type))
			for i2, name := range spec.Ident.Type {
				symbol, ok := c.symbolTable.Resolve(name.Name)
				if !ok {
					return c.errorf(nd, "unresolved reference %q", name)
				}
				np.Type[i2] = &symbol.SymbolInfo
			}
			named[i] = np
		}
	}

	if err := c.symbolTable.SetNamedParams(named...); err != nil {
		return c.error(nd, err)
	}

	stmts := c.helperBuildKwargsIfUndefinedStmts(namedSpecCount, func(index int) (name *node.Ident, types []*SymbolInfo, value node.Expr) {
		spec := namedSpec[index].(*node.NamedParamSpec)
		return spec.Ident.Ident, named[index].Type, spec.Value
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
			if v, ok := spec.Data.(int); ok {
				c.iotaVal = v
			} else {
				return c.errorf(nd, "invalid iota value")
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

			rightExpr := []node.Expr{v}
			err := c.compileAssignStmt(nd, leftExpr, rightExpr, nd.Tok, token.Define)
			if err != nil {
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
				Func: &node.Ident{Name: BuiltinStdIO.String()},
				CallArgs: node.CallArgs{Args: node.CallExprArgs{Values: []node.Expr{
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

	if op != token.Assign && op != token.Define {
		c.compileCompoundAssignment(nd, op)
	}
	return c.compileDefineAssign(nd, lhs[0], keyword, op, false)
}

func (c *Compiler) compileCompoundAssignment(
	nd ast.Node,
	op token.Token,
) {
	switch op {
	case token.AddAssign:
		c.emit(nd, OpBinaryOp, int(token.Add))
	case token.SubAssign:
		c.emit(nd, OpBinaryOp, int(token.Sub))
	case token.MulAssign:
		c.emit(nd, OpBinaryOp, int(token.Mul))
	case token.QuoAssign:
		c.emit(nd, OpBinaryOp, int(token.Quo))
	case token.RemAssign:
		c.emit(nd, OpBinaryOp, int(token.Rem))
	case token.AndAssign:
		c.emit(nd, OpBinaryOp, int(token.And))
	case token.OrAssign:
		c.emit(nd, OpBinaryOp, int(token.Or))
	case token.AndNotAssign:
		c.emit(nd, OpBinaryOp, int(token.AndNot))
	case token.XorAssign:
		c.emit(nd, OpBinaryOp, int(token.Xor))
	case token.ShlAssign:
		c.emit(nd, OpBinaryOp, int(token.Shl))
	case token.ShrAssign:
		c.emit(nd, OpBinaryOp, int(token.Shr))
	}
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
			if term, ok := expr.(*node.Ident); ok {
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
		c.emit(nd, OpNull)
		c.emit(nd, OpSetLocal, tempArrSymbol.Index)
	}
	return nil
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
	ident, selectors := resolveAssignLHS(lhs)
	numSel := len(selectors)
	if numSel == 0 && op == token.Define {
		return c.compileDefine(nd, ident, allowRedefine, keyword)
	}

	symbol, ok := c.symbolTable.Resolve(ident)
	if !ok {
		return c.errorf(nd, "unresolved reference %q", ident)
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

func resolveAssignLHS(expr node.Expr) (name string, selectors []node.Expr) {
	switch term := expr.(type) {
	case *node.SelectorExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Sel)
	case *node.IndexExpr:
		name, selectors = resolveAssignLHS(term.Expr)
		selectors = append(selectors, term.Index)
	case *node.Ident:
		name = term.Name
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

	c.symbolTable = c.symbolTable.Fork(true)
	if err := c.compileStmts(nd.Stmts...); err != nil {
		return err
	}

	c.symbolTable = c.symbolTable.Parent(false)
	return nil
}

func (c *Compiler) compileReturnStmt(nd *node.ReturnStmt) error {
	if nd.Result == nil {
		if c.tryCatchIndex > -1 {
			c.emit(nd, OpFinalizer, 0)
		}
		c.emit(nd, OpReturn, 0)
		return nil
	}

	if err := c.Compile(nd.Result); err != nil {
		return err
	}

	if c.tryCatchIndex > -1 {
		c.emit(nd, OpFinalizer, 0)
	}

	c.emit(nd, OpReturn, 1)
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
	//
	//   for :it := iterator(iterable); :it.next();  {
	//     k, v := :it.get()  // set locals
	//
	//     ... body ...
	//   }
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

func (c *Compiler) compileFuncLit(nd *node.FuncLit) error {
	if ident := nd.Type.Ident; ident != nil && nd.Type.Token == token.Func {
		nodeIndex := len(c.stack) - 1
		// prevent recursion on compileAssignStmt
		if nodeIndex < 1 || c.stack[nodeIndex-1] != c.stack[nodeIndex] {
			var (
				c2        = c
				addMethod bool
			)
		loop:
			for c2 != nil {
				for _, o := range c2.constants {
					switch ot := o.(type) {
					case *CallerObjectWithMethods:
						if ot.CallerObject.(*CompiledFunction).Name == ident.Name {
							addMethod = true
							break loop
						}
					}
				}
				c2 = c2.parent
			}

			if !addMethod {
				_, addMethod = c.symbolTable.builtins.Map[ident.Name]
			}

			if addMethod {
				nd.Type.AllowMethods = false
				nd.Type.Ident = nil
				return c.compileCallExpr(&node.CallExpr{
					Func: &node.Ident{Name: BuiltinAddCallMethod.String()},
					CallArgs: node.CallArgs{
						Args: node.CallExprArgs{
							Values: []node.Expr{
								ident,
								nd,
							},
						},
					},
				})
			}

			ass := &node.AssignStmt{
				LHS:   []node.Expr{ident},
				RHS:   []node.Expr{nd},
				Token: token.Define,
			}
			err := c.compileAssignStmt(ass,
				ass.LHS, ass.RHS, token.Const, ass.Token)
			if err != nil && strings.Contains(err.Error(), fmt.Sprintf("%q redeclared in this block", ident.Name)) {
				nd := *nd
				nd.Type.AllowMethods = false
				nd.Type.Ident = nil
				return c.compileCallExpr(&node.CallExpr{
					Func: &node.Ident{Name: BuiltinAddCallMethod.String()},
					CallArgs: node.CallArgs{
						Args: node.CallExprArgs{
							Values: []node.Expr{
								ident,
								&nd,
							},
						},
					},
				})
			}
			return err
		}
	}
	return c.compileFunc(nd, nd.Type, nd.Body)
}

func (c *Compiler) compileClosureLit(nd *node.ClosureLit) error {
	var stmts []node.Stmt
	if b, ok := nd.Body.(*node.BlockExpr); ok {
		stmts = b.Stmts
		if l := len(stmts); l > 0 {
			switch t := stmts[l-1].(type) {
			case *node.ExprStmt:
				stmts[l-1] = &node.ReturnStmt{Result: t.Expr}
			}
		}
	} else {
		stmts = append(stmts, &node.ReturnStmt{Result: nd.Body})
	}
	return c.compileFunc(nd, nd.Type, &node.BlockStmt{Stmts: stmts})
}

func (c *Compiler) compileFunc(nd ast.Node, typ *node.FuncType, body *node.BlockStmt) (err error) {
	var (
		params      = make([]string, len(typ.Params.Args.Values))
		types       = make([]ParamType, len(typ.Params.Args.Values))
		namedParams = make([]*NamedParam, len(typ.Params.NamedArgs.Names))
		symbolTable = c.symbolTable.Fork(false)
	)

	for i, ident := range typ.Params.Args.Values {
		if params[i], types[i], err = c.nameSymbolsOfTypedIdent(nd, ident); err != nil {
			return
		}
	}

	if typ.Params.Args.Var != nil {
		var (
			name    string
			symbols []*SymbolInfo
		)
		if name, symbols, err = c.nameSymbolsOfTypedIdent(nd, typ.Params.Args.Var); err != nil {
			return
		}
		params = append(params, name)
		types = append(types, symbols)
	}

	if err := symbolTable.SetParams(typ.Params.Args.Var != nil, params, types); err != nil {
		return c.error(nd, err)
	}

	for i, name := range typ.Params.NamedArgs.Names {
		if names, types, err2 := c.nameSymbolsOfTypedIdent(nd, name); err2 != nil {
			return err2
		} else {
			namedParams[i] = NewNamedParam(names, typ.Params.NamedArgs.Values[i].String())
			namedParams[i].Type = types
		}
	}

	if typ.Params.NamedArgs.Var != nil {
		namedParams = append(namedParams, &NamedParam{Name: typ.Params.NamedArgs.Var.Ident.Name})
	}

	if len(namedParams) > 0 {
		if err := symbolTable.SetNamedParams(namedParams...); err != nil {
			return c.error(nd, err)
		}
	}

	if count := len(typ.Params.NamedArgs.Values); count > 0 {
		body.Stmts = append(c.helperBuildKwargsStmts(count, func(index int) (name string, value node.Expr) {
			return typ.Params.NamedArgs.Names[index].Ident.Name, typ.Params.NamedArgs.Values[index]
		}), body.Stmts...)
	}

	fork := c.fork(c.file, c.module, c.moduleMap, symbolTable)
	fork.variadic = typ.Params.Args.Var != nil
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
	bc.Main.AllowMethods = typ.AllowMethods

	if typ.Ident != nil {
		bc.Main.Name = typ.Ident.Name
	}

	if bc.Main.NumLocals > 256 {
		return c.error(nd, ErrSymbolLimit)
	}

	c.constants = bc.Constants

	if len(freeSymbols) > 0 {
		bc.Main.AllowMethods = false
	}

	index := c.addConstant(bc.Main)

	if bc.Main.Name == "" {
		bc.Main.Name = fmt.Sprintf("#%d", index)
	}

	if len(freeSymbols) > 0 {
		c.emit(nd, OpClosure, index, len(freeSymbols))
	} else {
		c.emit(nd, OpConstant, index)
	}
	return nil
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
	case token.NullichCoalesce:
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

func (c *Compiler) compileBinaryExpr(nd *node.BinaryExpr) error {
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
	default:
		if !nd.Token.IsBinaryOperator() {
			return c.errorf(nd, "invalid binary operator: %s",
				nd.Token.String())
		}
		c.emit(nd, OpBinaryOp, int(nd.Token))
	}
	return nil
}

func (c *Compiler) compileUnaryExpr(nd *node.UnaryExpr) error {
	if err := c.Compile(nd.Expr); err != nil {
		return err
	}

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
	return nil
}

func (c *Compiler) compileSelectorExpr(nd *node.SelectorExpr) error {
	defer c.pushSelector()()
	expr, selectors := resolveSelectorExprs(nd)

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
		expr, selectors = resolveIndexExprs(v.Expr)
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
		expr, indexes = resolveIndexExprs(v.Expr)
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
		c.emit(nd, OpNull)
	}

	if nd.High != nil {
		if err := c.Compile(nd.High); err != nil {
			return err
		}
	} else {
		c.emit(nd, OpNull)
	}

	c.emit(nd, OpSliceIndex)
	return nil
}

func (c *Compiler) compileCallExpr(nd *node.CallExpr) error {
	var (
		selExpr    *node.SelectorExpr
		isSelector bool
		flags      OpCallFlag

		op      = OpCall
		numArgs = len(nd.Args.Values)
	)

	if nd.Func != nil {
		selExpr, isSelector = nd.Func.(*node.SelectorExpr)
	}
	if isSelector {
		if err := c.Compile(selExpr.Expr); err != nil {
			return err
		}
		op = OpCallName
	} else {
		if err := c.Compile(nd.Func); err != nil {
			return err
		}
	}

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
		namedArgs := &node.ArrayLit{Elements: make([]node.Expr, numKwargs)}

		for i, name := range nd.NamedArgs.Names {
			value := nd.NamedArgs.Values[i]
			if value == nil {
				// is flag
				value = &node.FlagLit{Value: true}
			}
			namedArgs.Elements[i] = &node.ArrayLit{Elements: []node.Expr{name.NameString(), value}}
		}

		if err := c.Compile(namedArgs); err != nil {
			return err
		}
	}

	if nd.NamedArgs.Var != nil {
		flags |= OpCallFlagVarNamedArgs
		if err := c.Compile(nd.NamedArgs.Var.Value); err != nil {
			return err
		}
	}

	if isSelector {
		if err := c.Compile(selExpr.Sel); err != nil {
			return err
		}
	}

	c.emit(nd, op, numArgs, int(flags))
	return nil
}

func (c *Compiler) compileImportExpr(nd *node.ImportExpr) error {
	moduleName := nd.ModuleName
	if moduleName == "" {
		return c.errorf(nd, "empty module name")
	}

	importer := c.moduleMap.Get(moduleName)
	if importer == nil {
		return c.errorf(nd, "module '%s' not found", moduleName)
	}

	extImp, isExt := importer.(ExtImporter)
	if isExt {
		if name, err := extImp.Name(); name != "" {
			moduleName = name
		} else if err != nil {
			return c.errorf(nd, "resolve name of module '%s': %v", moduleName, err.Error())
		}
	}

	module, exists := c.getModule(moduleName)
	if !exists {
		mod, url, err := importer.Import(c.opts.Context, moduleName)
		if err != nil {
			return c.error(nd, err)
		}
		switch v := mod.(type) {
		case []byte:
			var moduleMap *ModuleMap
			if isExt {
				moduleMap = c.moduleMap.Fork(moduleName)
			} else {
				moduleMap = c.baseModuleMap()
			}

			moduleInfo := &ModuleInfo{moduleName, url}

			cidx, err := c.compileModule(nd, importer, moduleInfo, moduleMap, v)
			if err != nil {
				return err
			}
			module = c.addModule(moduleName, 1, cidx)
			for _, cnt := range c.constants {
				if fn, ok := cnt.(*CompiledFunction); ok {
					fn.module = moduleInfo
				}
			}
		case Object:
			module = c.addModule(moduleName, 2, c.addConstant(v))
		default:
			return c.errorf(nd, "invalid import value type: %T", v)
		}
	}

	switch module.typ {
	case 1:
		var numParams int
		mod := c.constants[module.constantIndex]
		if cf, ok := mod.(*CompiledFunction); ok {
			numParams = cf.Params.Len
			if cf.Params.Var {
				numParams--
			}
		}
		// load module
		// if module is already stored, load from VM.modulesCache otherwise call compiled function
		// and store copy of result to VM.modulesCache.
		c.emit(nd, OpLoadModule, module.constantIndex, module.moduleIndex)
		jumpPos := c.emit(nd, OpJumpFalsy, 0)
		// modules should not accept parameters, to suppress the wrong number of arguments error
		// set all params to nil
		for i := 0; i < numParams; i++ {
			c.emit(nd, OpNull)
		}
		c.emit(nd, OpCall, numParams, 0)
		c.emit(nd, OpStoreModule, module.moduleIndex)
		c.changeOperand(jumpPos, len(c.instructions))
	case 2:
		// load module
		// if module is already stored, load from VM.modulesCache otherwise copy object
		// and store it to VM.modulesCache.
		c.emit(nd, OpLoadModule, module.constantIndex, module.moduleIndex)
		jumpPos := c.emit(nd, OpJumpFalsy, 0)
		c.emit(nd, OpStoreModule, module.moduleIndex)
		c.changeOperand(jumpPos, len(c.instructions))
	default:
		return c.errorf(nd, "invalid module type: %v", module.typ)
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

func (c *Compiler) compileIdent(nd *node.Ident) error {
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

func (c *Compiler) compileArrayLit(nd *node.ArrayLit) error {
	for _, elem := range nd.Elements {
		if err := c.Compile(elem); err != nil {
			return err
		}
	}

	c.emit(nd, OpArray, len(nd.Elements))
	return nil
}

func (c *Compiler) compileDictLit(nd *node.DictLit) error {
	for _, elt := range nd.Elements {
		// key
		c.emit(nd, OpConstant, c.addConstant(Str(elt.Key)))
		// value
		if err := c.Compile(elt.Value); err != nil {
			return err
		}
	}

	c.emit(nd, OpMap, len(nd.Elements)*2)
	return nil
}

func (c *Compiler) compileKeyValueLit(elt *node.KeyValueLit) (err error) {
	// key
	switch t := elt.Key.(type) {
	case *node.Ident:
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
	for _, elt := range nd.Elements {
		if flag, _ := elt.Value.(*node.FlagLit); flag != nil {
			if !flag.Value {
				length--
				continue
			}
		}
		if err = c.compileKeyValueLit(elt); err != nil {
			return
		}
	}

	c.emit(nd, OpKeyValueArray, length)
	return nil
}

func (c *Compiler) helperBuildKwargsStmts(count int, get func(index int) (name string, value node.Expr)) (stmts []node.Stmt) {
	for i := 0; i < count; i++ {
		name, value := get(i)
		nameLit := &node.StringLit{Literal: strconv.Quote(name), Value: name}
		values := []node.Expr{nameLit}
		stmts = append(stmts, &node.AssignStmt{
			Token: token.NullichAssign,
			LHS:   []node.Expr{&node.Ident{Name: name}},
			RHS: []node.Expr{&node.BinaryExpr{
				Token: token.NullichCoalesce,
				LHS: &node.CallExpr{
					Func: &node.NamedArgsKeyword{},
					CallArgs: node.CallArgs{
						Args: node.CallExprArgs{
							Values: values,
						},
					},
				},
				RHS: value,
			}},
		})
	}
	return
}

func (c *Compiler) helperBuildKwargsIfUndefinedStmts(count int, get func(index int) (ident *node.Ident, types []*SymbolInfo, value node.Expr)) (stmts []node.Stmt) {
	for i := 0; i < count; i++ {
		name, types, value := get(i)
		ident := name
		if len(types) > 0 {
			var typesArg node.Expr = &node.Ident{
				NamePos: name.Pos(),
				Name:    types[0].Name,
			}

			if len(types) > 1 {
				var typesElements = make([]node.Expr, len(types))
				for i2, symbol := range types {
					typesElements[i2] = &node.Ident{Name: symbol.Name}
				}
				typesArg = &node.ArrayLit{Elements: typesElements}
			}
			stmts = append(stmts, &node.IfStmt{
				Cond: &node.BinaryExpr{
					Token: token.Equal,
					LHS:   ident,
					RHS:   &node.NilLit{},
				},
				Body: &node.BlockStmt{
					Stmts: []node.Stmt{
						&node.AssignStmt{
							Token: token.Assign,
							LHS:   []node.Expr{ident},
							RHS:   []node.Expr{value},
						},
					},
				},
				Else: &node.ExprStmt{
					Expr: &node.CallExpr{
						Func: &node.Ident{
							NamePos: name.Pos(),
							Name:    BuiltinNamedParamTypeCheck.String(),
						},
						CallArgs: node.CallArgs{
							Args: node.CallExprArgs{
								Values: []node.Expr{
									&node.StringLit{Value: name.Name},
									typesArg,
									ident,
								},
							},
						},
					},
				},
			})
		} else {
			stmts = append(stmts, &node.AssignStmt{
				Token: token.NullichAssign,
				LHS:   []node.Expr{ident},
				RHS:   []node.Expr{value},
			})
		}
	}

	return
}

func (c *Compiler) nameSymbolsOfTypedIdent(nd ast.Node, ti *node.TypedIdent) (name string, symbols []*SymbolInfo, err error) {
	name = ti.Ident.Name
	if len(ti.Type) > 0 {
		symbols = make([]*SymbolInfo, len(ti.Type))
		for i2, tname := range ti.Type {
			symbol, ok := c.symbolTable.Resolve(tname.Name)
			if !ok {
				err = c.errorf(nd, "unresolved reference %q", tname)
				return
			}
			symbols[i2] = &symbol.SymbolInfo
		}
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
