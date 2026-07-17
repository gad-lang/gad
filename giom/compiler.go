package giom

import (
	"bytes"
	"fmt"

	"github.com/gad-lang/gad"
	gp "github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
	giomnode "github.com/gad-lang/gad/giom/node"
	giomparser "github.com/gad-lang/gad/giom/parser"
)

// Compiler compiles giom v2 source into GAD bytecode. Construct one with
// NewCompiler and call Compile; the same Compiler may compile multiple inputs
// with the same symbol table and options.
type Compiler struct {
	st   *gad.SymbolTable
	opts gad.CompileOptions
}

// NewCompiler returns a Compiler bound to the given symbol table and compile
// options. A nil symbol table is created on demand when compiling.
func NewCompiler(st *gad.SymbolTable, opts gad.CompileOptions) *Compiler {
	return &Compiler{st: st, opts: opts}
}

// Compile parses giom v2 source and compiles it to GAD bytecode.
func (c *Compiler) Compile(input []byte) (*giomnode.File, *gad.Bytecode, error) {
	fs := source.NewFileSet()
	filename := c.opts.CompilerOptions.ModuleFile
	if filename == "" {
		filename = gad.MainName
	}
	f := fs.AddFileData(filename, -1, input)
	p := giomparser.NewParser(f)
	file, err := p.ParseFile()
	if err != nil {
		return nil, nil, err
	}
	bc, err := CompileFile(c.st, &gad.ModuleSpec{ModuleInfo: gad.ModuleInfo{Name: gad.MainName}, Main: true}, file, c.opts)
	return file, bc, err
}

// Compile parses giom v2 source and compiles it to GAD bytecode. It is
// shorthand for NewCompiler(st, opts).Compile(input).
func Compile(st *gad.SymbolTable, input []byte, opts gad.CompileOptions) (*giomnode.File, *gad.Bytecode, error) {
	return NewCompiler(st, opts).Compile(input)
}

// CompileFile compiles a parsed giom v2 file to GAD bytecode.
func CompileFile(st *gad.SymbolTable, module *gad.ModuleSpec, file *giomnode.File, opts gad.CompileOptions) (*gad.Bytecode, error) {
	if st == nil {
		st = gad.NewSymbolTable(AppendBuiltins(gad.NewBuiltins()).NameSet)
	}
	if module == nil {
		module = &gad.ModuleSpec{ModuleInfo: gad.ModuleInfo{Name: gad.MainName}, Main: true}
	}
	if file.InputFile == nil {
		fs := source.NewFileSet()
		file.InputFile = fs.AddFileData(module.Name, -1, nil)
	}

	gadFile := &gp.File{InputFile: file.InputFile, Stmts: giomnode.ConvertFile(file.Stmts)}
	if opts.CompilerOptions.FallbackFunc == nil {
		opts.CompilerOptions.FallbackFunc = CompileFallback
	}
	return gad.CompileFile(st, module, gadFile, opts)
}

// CompileFallback compiles Giom-specific AST nodes through a Gad compiler.
func CompileFallback(c *gad.Compiler, nd ast.Node) error {
	switch n := nd.(type) {
	case *giomnode.File:
		return compileStmts(c, n.Stmts)
	case *giomnode.CodeStmt:
		return compileStmts(c, n.Stmts)
	case *giomnode.WrapStmt:
		return compileStmts(c, n.Body)
	case *giomnode.AssignStmt:
		return c.Compile(&gnode.AssignStmt{
			LHS:      []gnode.Expr{n.LHS},
			RHS:      []gnode.Expr{n.RHS},
			Token:    assignToken(n.Op),
			TokenPos: n.NodePos,
		})
	case *giomnode.CommentStmt:
		if n.Silent {
			return nil
		}
		return compileRendered(c, n)
	case *giomnode.FuncDecl,
		*giomnode.CompDecl,
		*giomnode.CompCallStmt,
		*giomnode.MatchStmt,
		*giomnode.VarStmt,
		*giomnode.ConstStmt,
		*giomnode.GlobalStmt,
		*giomnode.ExportStmt,
		*giomnode.SlotDecl,
		*giomnode.SlotPassStmt,
		*giomnode.ForStmt,
		*giomnode.IfStmt,
		*giomnode.DoctypeStmt,
		*giomnode.TextStmt,
		*giomnode.TagStmt:
		stmt, ok := nd.(gnode.Stmt)
		if !ok {
			return fmt.Errorf("giom v2 fallback: %T is not a statement", nd)
		}
		return compileStmts(c, giomnode.Convert(gnode.Stmts{stmt}))
	default:
		coder, ok := nd.(gnode.Coder)
		if !ok {
			return fmt.Errorf("giom v2 fallback: %T is not compilable", nd)
		}
		return compileRendered(c, coder)
	}
}

func compileStmts(c *gad.Compiler, stmts gnode.Stmts) error {
	for _, stmt := range stmts {
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func compileRendered(c *gad.Compiler, nd gnode.Coder) error {
	var buf bytes.Buffer
	gnode.CodeW(&buf, nd, gnode.CodeWithPrefix("\t"), gnode.CodeFormat())
	parsed, err := gp.Parse(buf.String(), "", nil, nil)
	if err != nil {
		return err
	}
	return compileStmts(c, parsed.Stmts)
}

func assignToken(op string) token.Token {
	switch op {
	case ":=", ":":
		return token.Define
	case "=":
		return token.Assign
	case "+=":
		return token.AddAssign
	case "-=":
		return token.SubAssign
	case "*=":
		return token.MulAssign
	case "/=":
		return token.QuoAssign
	case "%=":
		return token.RemAssign
	case "??=":
		return token.NullichAssign
	default:
		return token.Assign
	}
}
