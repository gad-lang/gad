package gad

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// Gadx (.gadx) compilation is selected by the ModuleFile extension on
// CompileOptions: a ".gadx" ModuleFile is parsed with Gadx's indentation-based
// syntax (tags, components, slots) and lowered to Gad statements before the
// ordinary Gad compiler runs. The lowered code references the `gadx` builtin
// namespace (gadx.Tag, gadx.attr, …), so the symbol table and VM must have
// those builtins registered (gadx.AppendBuiltins).

// parseGadxFile parses Gadx source into a Gad *parser.File by running the Gadx
// parser and lowering the resulting Gadx AST to Gad statements.
func parseGadxFile(srcFile *source.File) (*parser.File, error) {
	p := gadxparser.NewParser(srcFile)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}
	return &parser.File{InputFile: srcFile, Stmts: gadxnode.ConvertFile(file.Stmts)}, nil
}

// TranspileGadx parses Gadx source and writes the equivalent Gad source to
// outPath (a ".gad" suffix is appended when missing). It is the on-disk
// counterpart of the Gadx front-end: the same lowering used by compilation,
// emitted as readable Gad instead of bytecode.
func TranspileGadx(name string, src []byte, outPath string) error {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AppendFileData(name, src)
	p := gadxparser.NewParser(srcFile)
	parsed, err := p.ParseFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create transpile dir: %w", err)
	}
	converted := gadxnode.ConvertFile(parsed.Stmts)
	var buf bytes.Buffer
	node.CodeW(&buf, converted, node.CodeWithPrefix("\t"), node.CodeFormat())
	if !strings.HasSuffix(outPath, ".gad") {
		outPath += ".gad"
	}
	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write transpiled %s: %w", outPath, err)
	}
	return nil
}

// CompileGadxModule compiles Gadx source as an imported module through the given
// BuiltinCompileModuleContext. It parses src with the Gadx front-end, lowers it
// to Gad statements and compiles them, installing the Gadx fallback on the
// module's compiler so Gadx nodes lower correctly even when the importing file
// is plain Gad. Use it from an ExtImporter to support importing `.gadx` files.
func CompileGadxModule(ctx *BuiltinCompileModuleContext, src []byte) error {
	file := ctx.SetFileData(src)
	pf, err := parseGadxFile(file)
	if err != nil {
		return ctx.Compiler.Errorf(ctx.Node, "parse gadx %q: %w", file.Name, err)
	}
	if ctx.Compiler.Options().FallbackFunc == nil {
		ctx.Compiler.Options().FallbackFunc = gadxCompileFallback
	}
	return ctx.Compile(pf.Stmts)
}

// gadxCompileFallback compiles Gadx-specific AST nodes through the Gad compiler.
// It is installed as CompileOptions.FallbackFunc for Gadx compilation so the Gad
// compiler can lower the Gadx nodes emitted by gadxnode.ConvertFile.
func gadxCompileFallback(c *Compiler, nd ast.Node) error {
	switch n := nd.(type) {
	case *gadxnode.File:
		return gadxCompileStmts(c, n.Stmts)
	case *gadxnode.CodeStmt:
		return gadxCompileStmts(c, n.Stmts)
	case *gadxnode.WrapStmt:
		return gadxCompileStmts(c, n.Body)
	case *gadxnode.AssignStmt:
		return c.Compile(&node.AssignStmt{
			LHS:      []node.Expr{n.LHS},
			RHS:      []node.Expr{n.RHS},
			Token:    gadxAssignToken(n.Op),
			TokenPos: n.NodePos,
		})
	case *gadxnode.CommentStmt:
		if n.Silent {
			return nil
		}
		return gadxCompileRendered(c, n)
	case *gadxnode.FuncDecl,
		*gadxnode.CompDecl,
		*gadxnode.CompCallStmt,
		*gadxnode.MatchStmt,
		*gadxnode.VarStmt,
		*gadxnode.ConstStmt,
		*gadxnode.GlobalStmt,
		*gadxnode.ExportStmt,
		*gadxnode.SlotDecl,
		*gadxnode.SlotPassStmt,
		*gadxnode.ForStmt,
		*gadxnode.IfStmt,
		*gadxnode.DoctypeStmt,
		*gadxnode.TextStmt,
		*gadxnode.TagStmt:
		stmt, ok := nd.(node.Stmt)
		if !ok {
			return fmt.Errorf("gadx fallback: %T is not a statement", nd)
		}
		return gadxCompileStmts(c, gadxnode.Convert(node.Stmts{stmt}))
	default:
		coder, ok := nd.(node.Coder)
		if !ok {
			return fmt.Errorf("gadx fallback: %T is not compilable", nd)
		}
		return gadxCompileRendered(c, coder)
	}
}

func gadxCompileStmts(c *Compiler, stmts node.Stmts) error {
	for _, stmt := range stmts {
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func gadxCompileRendered(c *Compiler, nd node.Coder) error {
	// The fallback lowers a non-Gadx node by rendering it to source and
	// recompiling. A node the Gad compiler genuinely cannot compile would render
	// to source that re-parses to the same node, recursing forever; cap the depth
	// so such a case fails with a clear error instead of a stack overflow.
	if c.fallbackDepth >= 64 {
		return fmt.Errorf("gadx fallback: %T cannot be compiled (rendered form does not lower)", nd)
	}
	c.fallbackDepth++
	defer func() { c.fallbackDepth-- }()

	var buf bytes.Buffer
	node.CodeW(&buf, nd, node.CodeWithPrefix("\t"), node.CodeFormat())
	parsed, err := parser.Parse(buf.String(), "", nil, nil)
	if err != nil {
		return err
	}
	return gadxCompileStmts(c, parsed.Stmts)
}

func gadxAssignToken(op string) token.Token {
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
