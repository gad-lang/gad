// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

const MainName = "(main)"

var (
	// DefaultCompilerOptions holds default Compiler options.
	DefaultCompilerOptions = CompilerOptions{
		OptimizerMaxCycle: 100,
		OptimizeConst:     true,
		OptimizeExpr:      true,
	}

	DefaultCompileOptions = CompileOptions{
		CompilerOptions: DefaultCompilerOptions,
	}
	// TraceCompilerOptions holds Compiler options to print trace output
	// to stdout for Parser, Optimizer, Compiler.
	TraceCompilerOptions = CompilerOptions{
		Trace:             os.Stdout,
		TraceParser:       true,
		TraceCompiler:     true,
		TraceOptimizer:    true,
		OptimizerMaxCycle: 1<<8 - 1,
		OptimizeConst:     true,
		OptimizeExpr:      true,
	}
)

// errSkip is a sentinel error for compiler.
var errSkip = errors.New("skip")

type (

	// Compiler compiles the AST into a bytecode.
	Compiler struct {
		parent         *Compiler
		file           *parser.SourceFile
		constants      []Object
		constsCache    map[Object]int
		cfuncCache     map[uint32][]int
		symbolTable    *SymbolTable
		instructions   []byte
		sourceMap      map[int]int
		moduleMap      *ModuleMap
		moduleStore    *moduleStore
		module         *ModuleInfo
		variadic       bool
		varNamedParams bool
		loops          []*loopStmts
		loopIndex      int
		tryCatchIndex  int
		iotaVal        int
		opts           CompilerOptions
		trace          io.Writer
		indent         int
		stack          []ast.Node
		selectorStack  [][][]func()
	}

	// CompilerOptions represents customizable options for Compile().
	CompilerOptions struct {
		Context             context.Context
		ModuleMap           *ModuleMap
		Module              *ModuleInfo
		ModuleFile          string
		Constants           []Object
		SymbolTable         *SymbolTable
		Trace               io.Writer
		TraceParser         bool
		TraceCompiler       bool
		TraceOptimizer      bool
		OptimizerMaxCycle   int
		OptimizeConst       bool
		OptimizeExpr        bool
		MixedWriteFunction  node.Expr
		MixedExprToTextFunc node.Expr
		moduleStore         *moduleStore
		constsCache         map[Object]int
	}

	// CompilerError represents a compiler error.
	CompilerError struct {
		FileSet *parser.SourceFileSet
		Node    ast.Node
		Err     error
	}

	// moduleStoreItem represents indexes of a single module.
	moduleStoreItem struct {
		typ           int
		constantIndex int
		moduleIndex   int
		name          string
	}

	// moduleStore represents modules indexes and total count that are defined
	// while compiling.
	moduleStore struct {
		count int
		store map[string]*moduleStoreItem
		items []*moduleStoreItem
	}

	// loopStmts represents a loopStmts construct that the compiler uses to
	// track the current loopStmts.
	loopStmts struct {
		continues         []int
		breaks            []int
		lastTryCatchIndex int
	}
)

func (e *CompilerError) Error() string {
	filePos := e.FileSet.Position(e.Node.Pos())
	return fmt.Sprintf("Compile Error: %s\n\tat %s", e.Err.Error(), filePos)
}

func (e *CompilerError) Unwrap() error {
	return e.Err
}

// NewCompiler creates a new Compiler object.
func NewCompiler(file *parser.SourceFile, opts CompilerOptions) *Compiler {
	if opts.SymbolTable == nil {
		opts.SymbolTable = NewSymbolTable(NewBuiltins())
	}

	if opts.Module == nil {
		opts.Module = &ModuleInfo{
			Name: MainName,
			File: "file:" + file.Name,
		}
	}

	if opts.constsCache == nil {
		opts.constsCache = make(map[Object]int)
		for i := range opts.Constants {
			switch opts.Constants[i].(type) {
			case Int, Uint, Str, Bool, Flag, Float, Char, *NilType:
				opts.constsCache[opts.Constants[i]] = i
			}
		}
	}

	if opts.moduleStore == nil {
		opts.moduleStore = newModuleStore()
	}

	var trace io.Writer
	if opts.TraceCompiler {
		trace = opts.Trace
	}

	return &Compiler{
		file:          file,
		constants:     opts.Constants,
		constsCache:   opts.constsCache,
		cfuncCache:    make(map[uint32][]int),
		symbolTable:   opts.SymbolTable,
		sourceMap:     make(map[int]int),
		moduleMap:     opts.ModuleMap,
		moduleStore:   opts.moduleStore,
		module:        opts.Module,
		loopIndex:     -1,
		tryCatchIndex: -1,
		iotaVal:       -1,
		opts:          opts,
		trace:         trace,
	}
}

type CompileOptions struct {
	CompilerOptions
	ParserOptions  parser.ParserOptions
	ScannerOptions parser.ScannerOptions
}

// Compile compiles given script to Bytecode.
func Compile(script []byte, opts CompileOptions) (*Bytecode, error) {
	var (
		fileSet    = parser.NewFileSet()
		moduleName string
	)

	if opts.Module != nil {
		moduleName = opts.Module.Name
	}

	if moduleName == "" {
		moduleName = MainName
	}

	srcFile := fileSet.AddFile(moduleName, -1, len(script))
	if opts.TraceParser && opts.ParserOptions.Trace == nil {
		opts.ParserOptions.Trace = opts.Trace
	}

	p := parser.NewParserWithOptions(srcFile, script, &opts.ParserOptions, &opts.ScannerOptions)
	pf, err := p.ParseFile()
	if err != nil {
		return nil, err
	}

	compiler := NewCompiler(srcFile, opts.CompilerOptions)
	compiler.SetGlobalSymbolsIndex()

	if opts.OptimizeConst || opts.OptimizeExpr {
		err := compiler.optimize(pf)
		if err != nil && err != errSkip {
			return nil, err
		}
	}

	if err := compiler.Compile(pf); err != nil {
		return nil, err
	}

	bc := compiler.Bytecode()
	if bc.Main.NumLocals > 256 {
		return nil, ErrSymbolLimit
	}
	return bc, nil
}

// SetGlobalSymbolsIndex sets index of a global symbol. This is only required
// when a global symbol is defined in SymbolTable and provided to compiler.
// Otherwise, caller needs to append the constant to Constants, set the symbol
// index and provide it to the Compiler. This should be called before
// Compiler.Compile call.
func (c *Compiler) SetGlobalSymbolsIndex() {
	symbols := c.symbolTable.Symbols()
	for _, s := range symbols {
		if s.Scope == ScopeGlobal && s.Index == -1 {
			s.Index = c.addConstant(Str(s.Name))
		}
	}
}

// optimize runs the Optimizer and returns Optimizer object and error from Optimizer.
// Note:If optimizer cannot run for some reason, a nil optimizer and errSkip
// error will be returned.
func (c *Compiler) optimize(file *parser.File) error {
	if c.opts.OptimizerMaxCycle < 1 {
		return errSkip
	}

	optim := NewOptimizer(file, c.symbolTable, c.opts)

	if err := optim.Optimize(); err != nil {
		return err
	}

	c.opts.OptimizerMaxCycle -= optim.Total()
	return nil
}

// Bytecode returns compiled Bytecode ready to run in VM.
func (c *Compiler) Bytecode() *Bytecode {
	var lastOp Opcode
	var operands = make([]int, 0, 4)
	var jumpPos = make(map[int]struct{})
	var offset int
	var i int

	for i < len(c.instructions) {
		lastOp = Opcode(c.instructions[i])
		numOperands := OpcodeOperands[lastOp]
		operands, offset = ReadOperands(
			numOperands,
			c.instructions[i+1:],
			operands,
		)

		switch lastOp {
		case OpJump, OpJumpFalsy, OpAndJump, OpOrJump, OpJumpNotNil:
			jumpPos[operands[0]] = struct{}{}
		}

		delete(jumpPos, i)
		i += offset + 1
	}

	if lastOp != OpReturn || len(jumpPos) > 0 {
		c.emit(nil, OpReturn, 0)
	}

	cf := &CompiledFunction{
		Params:       c.symbolTable.params,
		NamedParams:  c.symbolTable.namedParams,
		NumLocals:    c.symbolTable.maxDefinition,
		Instructions: c.instructions,
		SourceMap:    c.sourceMap,
		sourceFile:   c.file,
		module:       c.module,
	}

	return &Bytecode{
		FileSet:    c.file.Set(),
		Constants:  c.constants,
		Main:       cf,
		NumModules: c.moduleStore.count,
	}
}

// CompileStmts compiles parser.Stmt and builds Bytecode.
func (c *Compiler) compileStmts(stmt ...node.Stmt) (err error) {
	l := len(stmt)

	if l == 0 {
		return nil
	}

stmts:
	for i := 0; i < l; i++ {
		switch stmt[i].(type) {
		case *node.RawStringStmt, *node.ExprToTextStmt:
			var j = i + 1
		l2:
			for j < l {
				switch stmt[j].(type) {
				case *node.RawStringStmt, *node.ExprToTextStmt:
					j++
				default:
					break l2
				}
			}

			var exprs = make([]node.Expr, j-i)

			for z, s := range stmt[i:j] {
				switch t := s.(type) {
				case *node.RawStringStmt:
					if len(t.Lits) == 1 {
						exprs[z] = t.Lits[0]
					} else {
						exprs[z] = &node.RawStringLit{Literal: t.Unquoted()}
					}
				case *node.ExprToTextStmt:
					exprs[z] = t.Expr
				}
			}

			var (
				wf = c.opts.MixedWriteFunction
				na node.CallExprNamedArgs
			)
			if wf == nil {
				wf = &node.Ident{Name: "write"}
			}
			if c.opts.MixedExprToTextFunc != nil {
				na = *new(node.CallExprNamedArgs).AppendS("convert", c.opts.MixedExprToTextFunc)
			}
			err = c.compileCallExpr(&node.CallExpr{
				Func: wf,
				CallArgs: node.CallArgs{
					Args:      node.CallExprArgs{Values: exprs},
					NamedArgs: na,
				},
			})
			if err != nil {
				return
			}
			i = j - 1
			continue stmts
		default:
			if err = c.Compile(stmt[i]); err != nil {
				return
			}
		}
	}

	return nil
}

// Compile compiles parser.Node and builds Bytecode.
func (c *Compiler) Compile(nd ast.Node) error {
	defer c.at(nd)()
	if c.trace != nil {
		if nd != nil {
			defer untracec(tracec(c, fmt.Sprintf("%s (%s)",
				nd.String(), reflect.TypeOf(nd).Elem().Name())))
		} else {
			defer untracec(tracec(c, ReprQuote("nil")))
		}
	}

	switch nt := nd.(type) {
	case *parser.File:
		if err := c.compileStmts(nt.Stmts...); err != nil {
			return err
		}
	case *node.ExprStmt:
		if err := c.Compile(nt.Expr); err != nil {
			return err
		}
		if f, _ := nt.Expr.(*node.FuncLit); f != nil && f.Type.Ident != nil {
			return nil
		}
		c.emit(nt, OpPop)
	case *node.IncDecStmt:
		op := token.AddAssign
		if nt.Token == token.Dec {
			op = token.SubAssign
		}
		return c.compileAssignStmt(
			nt,
			[]node.Expr{nt.Expr},
			[]node.Expr{&node.IntLit{Value: 1, ValuePos: nt.TokenPos}},
			token.Var,
			op,
		)
	case *node.ParenExpr:
		return c.Compile(nt.Expr)
	case *node.BinaryExpr:
		switch nt.Token {
		case token.LAnd, token.LOr, token.NullichCoalesce:
			return c.compileLogical(nt)
		default:
			return c.compileBinaryExpr(nt)
		}
	case *node.IntLit:
		c.emit(nt, OpConstant, c.addConstant(Int(nt.Value)))
	case *node.UintLit:
		c.emit(nt, OpConstant, c.addConstant(Uint(nt.Value)))
	case *node.FloatLit:
		c.emit(nt, OpConstant, c.addConstant(Float(nt.Value)))
	case *node.DecimalLit:
		c.emit(nt, OpConstant, c.addConstant(Decimal(nt.Value)))
	case *node.BoolLit:
		if nt.Value {
			c.emit(nt, OpTrue)
		} else {
			c.emit(nt, OpFalse)
		}
	case *node.FlagLit:
		if nt.Value {
			c.emit(nt, OpYes)
		} else {
			c.emit(nt, OpNo)
		}
	case *node.StringLit:
		c.emit(nt, OpConstant, c.addConstant(Str(nt.Value)))
	case *node.RawStringLit:
		c.emit(nt, OpConstant, c.addConstant(RawStr(nt.UnquotedValue())))
	case *node.CharLit:
		c.emit(nt, OpConstant, c.addConstant(Char(nt.Value)))
	case *node.NilLit:
		c.emit(nt, OpNull)
	case *node.StdInLit:
		c.emit(nt, OpStdIn)
	case *node.StdOutLit:
		c.emit(nt, OpStdOut)
	case *node.StdErrLit:
		c.emit(nt, OpStdErr)
	case *node.DotFileNameLit:
		c.emit(nt, OpDotName)
	case *node.DotFileLit:
		c.emit(nt, OpDotFile)
	case *node.IsModuleLit:
		c.emit(nt, OpIsModule)
	case *node.CalleeKeyword:
		c.emit(nt, OpCallee)
	case *node.ArgsKeyword:
		c.emit(nt, OpArgs)
	case *node.NamedArgsKeyword:
		c.emit(nt, OpNamedArgs)
	case *node.UnaryExpr:
		return c.compileUnaryExpr(nt)
	case *node.ThrowExpr:
		return c.compileThrowExpr(nt)
	case *node.IfStmt:
		return c.compileIfStmt(nt)
	case *node.TryStmt:
		return c.compileTryStmt(nt)
	case *node.CatchStmt:
		return c.compileCatchStmt(nt)
	case *node.FinallyStmt:
		return c.compileFinallyStmt(nt)
	case *node.ThrowStmt:
		return c.compileThrowStmt(nt)
	case *node.ForStmt:
		return c.compileForStmt(nt)
	case *node.ForInStmt:
		return c.compileForInStmt(nt)
	case *node.BranchStmt:
		return c.compileBranchStmt(nt)
	case *node.BlockStmt:
		return c.compileBlockStmt(nt)
	case *node.DeclStmt:
		return c.compileDeclStmt(nt)
	case *node.AssignStmt:
		return c.compileAssignStmt(nt,
			nt.LHS, nt.RHS, token.Var, nt.Token)
	case *node.Ident:
		return c.compileIdent(nt)
	case *node.ArrayLit:
		return c.compileArrayLit(nt)
	case *node.DictLit:
		return c.compileDictLit(nt)
	case *node.KeyValueArrayLit:
		return c.compileKeyValueArrayLit(nt)
	case *node.SelectorExpr: // selector on RHS side
		return c.compileSelectorExpr(nt)
	case *node.NullishSelectorExpr: // selector on RHS side
		return c.compileNullishSelectorExpr(nt)
	case *node.IndexExpr:
		return c.compileIndexExpr(nt)
	case *node.SliceExpr:
		return c.compileSliceExpr(nt)
	case *node.FuncLit:
		return c.compileFuncLit(nt)
	case *node.ClosureLit:
		return c.compileClosureLit(nt)
	case *node.KeyValueLit:
		return c.compileKeyValueLit(nt)
	case *node.ReturnStmt:
		return c.compileReturnStmt(nt)
	case *node.CallExpr:
		return c.compileCallExpr(nt)
	case *node.ImportExpr:
		return c.compileImportExpr(nt)
	case *node.CondExpr:
		return c.compileCondExpr(nt)
	case *node.RawStringStmt:
		return c.compileStmts(nt)
	case *node.EmptyStmt:
	case *node.ConfigStmt:
		if nt.Options.WriteFunc != nil {
			c.opts.MixedWriteFunction = nt.Options.WriteFunc
		}
		if nt.Options.ExprToTextFunc != nil {
			c.opts.MixedExprToTextFunc = nt.Options.ExprToTextFunc
		}
	case nil:
	default:
		return c.errorf(nt, `%[1]T "%[1]v" not implemented`, nt)
	}
	return nil
}

func (c *Compiler) at(nd ast.Node) func() {
	c.stack = append(c.stack, nd)
	return func() {
		c.stack = c.stack[:len(c.stack)-1]
	}
}

func (c *Compiler) changeOperand(opPos int, operand ...int) {
	op := c.instructions[opPos]
	inst := make([]byte, 0, 8)
	inst, err := MakeInstruction(inst, Opcode(op), operand...)
	if err != nil {
		panic(err)
	}
	c.replaceInstruction(opPos, inst)
}

func (c *Compiler) replaceInstruction(pos int, inst []byte) {
	copy(c.instructions[pos:], inst)
	if c.trace != nil {
		printTrace(c.indent, c.trace, fmt.Sprintf("REPLC %s",
			FormatInstructions(c.instructions[pos:], pos)[0]))
	}
}

func (c *Compiler) addConstant(obj Object) (index int) {
	defer func() {
		if c.trace != nil {
			printTrace(c.indent, c.trace,
				fmt.Sprintf("CONST %04d %v", index, obj))
		}
	}()

	switch obj.(type) {
	case Int, Uint, Str, RawStr, Bool, Flag, Float, Char, *NilType, Decimal:
		i, ok := c.constsCache[obj]
		if ok {
			index = i
			return
		}
	case *CompiledFunction:
		return c.addCompiledFunction(obj)
	default:
		// unhashable types cannot be stored in constsCache, append them to constants slice
		// and return index
		index = len(c.constants)
		c.constants = append(c.constants, obj)
		return
	}

	index = len(c.constants)
	c.constants = append(c.constants, obj)
	c.constsCache[obj] = index
	return
}

func (c *Compiler) addCompiledFunction(obj Object) (index int) {
	// Currently, caching compiled functions is only effective for functions
	// used in const declarations.
	// e.g.
	// const (
	// 	f = func() { return 1 }
	// 	g
	// )
	//
	cf := obj.(*CompiledFunction)
	key := cf.hash32()
	arr, ok := c.cfuncCache[key]
	if ok {
		for _, idx := range arr {
			var f *CompiledFunction
			switch t := c.constants[idx].(type) {
			case *CompiledFunction:
				f = t
			case *CallerObjectWithMethods:
				f = t.CallerObject.(*CompiledFunction)
			}
			if f.identical(cf) && f.equalSourceMap(cf) {
				return idx
			}
		}
	}
	index = len(c.constants)
	var co CallerObject = cf
	if cf.AllowMethods {
		co = NewCallerObjectWithMethods(cf)
	}
	c.constants = append(c.constants, co)
	c.cfuncCache[key] = append(c.cfuncCache[key], index)
	return
}

func (c *Compiler) emit(nd ast.Node, opcode Opcode, operands ...int) int {
	filePos := source.NoPos
	if nd != nil {
		filePos = nd.Pos()
	}

	inst := make([]byte, 0, 8)
	inst, err := MakeInstruction(inst, opcode, operands...)
	if err != nil {
		panic(err)
	}

	pos := c.addInstruction(inst)
	c.sourceMap[pos] = int(filePos)

	if c.trace != nil {
		printTrace(c.indent, c.trace, fmt.Sprintf("EMIT  %s",
			FormatInstructions(c.instructions[pos:], pos)[0]))
	}
	return pos
}

func (c *Compiler) addInstruction(b []byte) int {
	posNewIns := len(c.instructions)
	c.instructions = append(c.instructions, b...)
	return posNewIns
}

func (c *Compiler) checkCyclicImports(nd ast.Node, modulePath string) error {
	if c.module.Name == modulePath {
		return c.errorf(nd, "cyclic module import: %s", modulePath)
	} else if c.parent != nil {
		return c.parent.checkCyclicImports(nd, modulePath)
	}
	return nil
}

func (c *Compiler) addModule(name string, typ, constantIndex int) *moduleStoreItem {
	moduleIndex := c.moduleStore.count
	c.moduleStore.count++
	item := &moduleStoreItem{
		typ:           typ,
		constantIndex: constantIndex,
		moduleIndex:   moduleIndex,
		name:          name,
	}
	c.moduleStore.store[name] = item
	c.moduleStore.items = append(c.moduleStore.items, item)
	return item
}

func (c *Compiler) getModule(name string) (*moduleStoreItem, bool) {
	indexes, ok := c.moduleStore.store[name]
	return indexes, ok
}

func (c *Compiler) baseModuleMap() *ModuleMap {
	if c.parent == nil {
		return c.moduleMap
	}
	return c.parent.baseModuleMap()
}

func (c *Compiler) CompileModule(
	nd ast.Node,
	module *ModuleInfo,
	moduleMap *ModuleMap,
	src []byte,
	parserOptions *parser.ParserOptions,
	scannerOptions *parser.ScannerOptions,
) (bc *Bytecode, err error) {
	modFile := c.file.Set().AddFile(module.Name, -1, len(src))
	var trace io.Writer
	if c.opts.TraceParser {
		trace = c.trace
	}

	if parserOptions == nil {
		parserOptions = &parser.ParserOptions{Trace: trace}
	}

	p := parser.NewParserWithOptions(modFile, src, parserOptions, scannerOptions)

	var file *parser.File
	file, err = p.ParseFile()
	if err != nil {
		return
	}

	symbolTable := NewSymbolTable(c.symbolTable.builtins).
		DisableBuiltin(c.symbolTable.DisabledBuiltins()...)

	fork := c.fork(modFile, module, moduleMap, symbolTable)
	err = fork.optimize(file)
	if err != nil && err != errSkip {
		err = c.error(nd, err)
		return
	}
	if err = fork.Compile(file); err != nil {
		return
	}

	bc = fork.Bytecode()
	return
}

func (c *Compiler) compileModule(
	nd ast.Node,
	importable Importable,
	module *ModuleInfo,
	moduleMap *ModuleMap,
	src []byte,
) (int, error) {
	var err error
	if err = c.checkCyclicImports(nd, module.Name); err != nil {
		return 0, err
	}

	var bc *Bytecode
	if cimp, ok := importable.(CompilableImporter); ok {
		if bc, err = cimp.CompileModule(c, nd, module, moduleMap, src); err != nil {
			return 0, err
		}
	} else if bc, err = c.CompileModule(nd, module, moduleMap, src, nil, nil); err != nil {
		return 0, err
	}

	if bc.Main.NumLocals > 256 {
		return 0, c.error(nd, ErrSymbolLimit)
	}

	c.constants = bc.Constants
	index := c.addConstant(bc.Main)
	return index, nil
}

func (c *Compiler) enterLoop() *loopStmts {
	loop := &loopStmts{lastTryCatchIndex: c.tryCatchIndex}
	c.loops = append(c.loops, loop)
	c.loopIndex++

	if c.trace != nil {
		printTrace(c.indent, c.trace, "LOOPE", c.loopIndex)
	}
	return loop
}

func (c *Compiler) leaveLoop() {
	if c.trace != nil {
		printTrace(c.indent, c.trace, "LOOPL", c.loopIndex)
	}
	c.loops = c.loops[:len(c.loops)-1]
	c.loopIndex--
}

func (c *Compiler) currentLoop() *loopStmts {
	if c.loopIndex >= 0 {
		return c.loops[c.loopIndex]
	}
	return nil
}

func (c *Compiler) fork(
	file *parser.SourceFile,
	module *ModuleInfo,
	moduleMap *ModuleMap,
	symbolTable *SymbolTable,
) *Compiler {
	child := NewCompiler(file, CompilerOptions{
		Context:           c.opts.Context,
		ModuleMap:         moduleMap,
		Module:            module,
		Constants:         c.constants,
		SymbolTable:       symbolTable,
		Trace:             c.trace,
		TraceParser:       c.opts.TraceParser,
		TraceCompiler:     c.opts.TraceCompiler,
		TraceOptimizer:    c.opts.TraceOptimizer,
		OptimizerMaxCycle: c.opts.OptimizerMaxCycle,
		OptimizeConst:     c.opts.OptimizeConst,
		OptimizeExpr:      c.opts.OptimizeExpr,
		moduleStore:       c.moduleStore,
		constsCache:       c.constsCache,
	})

	child.parent = c
	child.cfuncCache = c.cfuncCache

	if module.Name == c.module.Name {
		child.indent = c.indent
	}
	return child
}

func (c *Compiler) error(nd ast.Node, err error) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    nd,
		Err:     err,
	}
}

func (c *Compiler) errorf(
	nd ast.Node,
	format string,
	args ...any,
) error {
	return &CompilerError{
		FileSet: c.file.Set(),
		Node:    nd,
		Err:     fmt.Errorf(format, args...),
	}
}

func printTrace(indent int, trace io.Writer, a ...any) {
	const (
		dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
		n    = len(dots)
	)

	i := 2 * indent
	for i > n {
		_, _ = fmt.Fprint(trace, dots)
		i -= n
	}

	_, _ = fmt.Fprint(trace, dots[0:i])
	_, _ = fmt.Fprintln(trace, a...)
}

func tracec(c *Compiler, msg string) *Compiler {
	printTrace(c.indent, c.trace, msg, "{")
	c.indent++
	return c
}

func untracec(c *Compiler) {
	c.indent--
	printTrace(c.indent, c.trace, "}")
}

// MakeInstruction returns a bytecode for an Opcode and the operands.
//
// Provide "buf" slice which is a returning value to reduce allocation or nil
// to create new byte slice. This is implemented to reduce compilation
// allocation that resulted in -15% allocation, +2% speed in compiler.
// It takes ~8ns/op with zero allocation.
//
// Returning error is required to identify bugs faster when VM and Opcodes are
// under heavy development.
//
// Warning: Unknown Opcode causes panic!
func MakeInstruction(buf []byte, op Opcode, args ...int) ([]byte, error) {
	operands := OpcodeOperands[op]
	if len(operands) != len(args) {
		return buf, fmt.Errorf(
			"MakeInstruction: %s expected %d operands, but got %d",
			OpcodeNames[op], len(operands), len(args),
		)
	}

	buf = append(buf[:0], byte(op))
	switch op {
	case OpGetBuiltin, OpConstant, OpMap, OpArray, OpGetGlobal, OpSetGlobal, OpJump,
		OpJumpFalsy, OpAndJump, OpOrJump, OpStoreModule, OpKeyValueArray,
		OpJumpNil, OpJumpNotNil:
		buf = append(buf, byte(args[0]>>8))
		buf = append(buf, byte(args[0]))
		return buf, nil
	case OpLoadModule, OpSetupTry, OpIterNextElse:
		buf = append(buf, byte(args[0]>>8))
		buf = append(buf, byte(args[0]))
		buf = append(buf, byte(args[1]>>8))
		buf = append(buf, byte(args[1]))
		return buf, nil
	case OpClosure:
		buf = append(buf, byte(args[0]>>8))
		buf = append(buf, byte(args[0]))
		buf = append(buf, byte(args[1]))
		return buf, nil
	case OpCall, OpCallName:
		buf = append(buf, byte(args[0]))
		buf = append(buf, byte(args[1]))
		return buf, nil
	case OpReturn, OpBinaryOp, OpUnary, OpGetIndex, OpGetLocal,
		OpSetLocal, OpGetFree, OpSetFree, OpGetLocalPtr, OpGetFreePtr, OpThrow,
		OpFinalizer, OpDefineLocal, OpKeyValue:
		buf = append(buf, byte(args[0]))
		return buf, nil
	case OpEqual, OpNotEqual, OpNull, OpTrue, OpFalse, OpYes, OpNo, OpPop, OpSliceIndex,
		OpSetIndex, OpIterInit, OpIterNext, OpIterKey, OpIterValue,
		OpSetupCatch, OpSetupFinally, OpNoOp, OpCallee, OpArgs, OpNamedArgs,
		OpStdIn, OpStdOut, OpStdErr, OpIsNil, OpNotIsNil, OpDotName, OpDotFile, OpIsModule:
		return buf, nil
	default:
		return buf, &Error{
			Name:    "MakeInstruction",
			Message: fmt.Sprintf("unknown Opcode %d %s", op, OpcodeNames[op]),
		}
	}
}

// FormatInstructions returns string representation of bytecode instructions.
func FormatInstructions(b []byte, posOffset int) []string {
	var out []string
	var operands = make([]int, 0, 4)
	var offset int
	var i int

	for i < len(b) {
		numOperands := OpcodeOperands[b[i]]
		operands, offset = ReadOperands(numOperands, b[i+1:], operands)

		switch len(numOperands) {
		case 0:
			out = append(out, fmt.Sprintf("%04d %-7s",
				posOffset+i, OpcodeNames[b[i]]))
		case 1:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d",
				posOffset+i, OpcodeNames[b[i]], operands[0]))
		case 2:
			out = append(out, fmt.Sprintf("%04d %-7s %-5d %-5d",
				posOffset+i, OpcodeNames[b[i]],
				operands[0], operands[1]))
		}
		i += 1 + offset
	}
	return out
}

// IterateInstructions iterate instructions and call given function for each instruction.
// Note: Do not use operands slice in callback, it is reused for less allocation.
func IterateInstructions(insts []byte,
	fn func(pos int, opcode Opcode, operands []int, offset int) bool) {
	operands := make([]int, 0, 4)
	var offset int

	for i := 0; i < len(insts); i++ {
		numOperands := OpcodeOperands[insts[i]]
		operands, offset = ReadOperands(numOperands, insts[i+1:], operands)
		if !fn(i, Opcode(insts[i]), operands, offset) {
			break
		}
		i += offset
	}
}

func newModuleStore() *moduleStore {
	return &moduleStore{
		store: make(map[string]*moduleStoreItem),
	}
}

func (ms *moduleStore) reset() *moduleStore {
	ms.count = 0
	for k := range ms.store {
		delete(ms.store, k)
	}
	return ms
}
