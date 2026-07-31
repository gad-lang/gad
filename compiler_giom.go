package gad

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	giomnode "github.com/gad-lang/gad/giom/node"
	giomparser "github.com/gad-lang/gad/giom/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// GiomOptions configures Giom (.giom) template compilation. A non-nil
// *GiomOptions on CompileOptions selects the Giom front-end: the source is
// parsed with Giom's indentation-based syntax (tags, components, slots) and
// lowered to Gad statements before the ordinary Gad compiler runs. The lowered
// code references the `giom` builtin namespace (giom.Tag, giom.attr, …), so the
// symbol table and VM must have those builtins registered (giom.AppendBuiltins).
type GiomOptions struct {
	// Reserved for future Giom-specific compile settings.
}

// parseGiomFile parses Giom source into a Gad *parser.File by running the Giom
// parser and lowering the resulting Giom AST to Gad statements.
func parseGiomFile(srcFile *source.File) (*parser.File, error) {
	p := giomparser.NewParser(srcFile)
	file, err := p.ParseFile()
	if err != nil {
		return nil, err
	}
	return &parser.File{InputFile: srcFile, Stmts: giomnode.ConvertFile(file.Stmts)}, nil
}

// TranspileGiom parses Giom source and writes the equivalent Gad source to
// outPath (a ".gad" suffix is appended when missing). It is the on-disk
// counterpart of the Giom front-end: the same lowering used by compilation,
// emitted as readable Gad instead of bytecode.
func TranspileGiom(name string, src []byte, outPath string) error {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AppendFileData(name, src)
	p := giomparser.NewParser(srcFile)
	parsed, err := p.ParseFile()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create transpile dir: %w", err)
	}
	converted := giomnode.ConvertFile(parsed.Stmts)
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

// CompileGiomModule compiles Giom source as an imported module through the given
// BuiltinCompileModuleContext. It parses src with the Giom front-end, lowers it
// to Gad statements and compiles them, installing the Giom fallback on the
// module's compiler so Giom nodes lower correctly even when the importing file
// is plain Gad. Use it from an ExtImporter to support importing `.giom` files.
func CompileGiomModule(ctx *BuiltinCompileModuleContext, src []byte) error {
	file := ctx.SetFileData(src)
	pf, err := parseGiomFile(file)
	if err != nil {
		return ctx.Compiler.Errorf(ctx.Node, "parse giom %q: %w", file.Name, err)
	}
	if ctx.Compiler.Options().FallbackFunc == nil {
		ctx.Compiler.Options().FallbackFunc = giomCompileFallback
	}
	return ctx.Compile(pf.Stmts)
}

// giomCompileFallback compiles Giom-specific AST nodes through the Gad compiler.
// It is installed as CompileOptions.FallbackFunc for Giom compilation so the Gad
// compiler can lower the Giom nodes emitted by giomnode.ConvertFile.
func giomCompileFallback(c *Compiler, nd ast.Node) error {
	switch n := nd.(type) {
	case *giomnode.File:
		return giomCompileStmts(c, n.Stmts)
	case *giomnode.CodeStmt:
		return giomCompileStmts(c, n.Stmts)
	case *giomnode.WrapStmt:
		return giomCompileStmts(c, n.Body)
	case *giomnode.AssignStmt:
		return c.Compile(&node.AssignStmt{
			LHS:      []node.Expr{n.LHS},
			RHS:      []node.Expr{n.RHS},
			Token:    giomAssignToken(n.Op),
			TokenPos: n.NodePos,
		})
	case *giomnode.CommentStmt:
		if n.Silent {
			return nil
		}
		return giomCompileRendered(c, n)
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
		stmt, ok := nd.(node.Stmt)
		if !ok {
			return fmt.Errorf("giom fallback: %T is not a statement", nd)
		}
		return giomCompileStmts(c, giomnode.Convert(node.Stmts{stmt}))
	default:
		coder, ok := nd.(node.Coder)
		if !ok {
			return fmt.Errorf("giom fallback: %T is not compilable", nd)
		}
		return giomCompileRendered(c, coder)
	}
}

func giomCompileStmts(c *Compiler, stmts node.Stmts) error {
	for _, stmt := range stmts {
		if err := c.Compile(stmt); err != nil {
			return err
		}
	}
	return nil
}

func giomCompileRendered(c *Compiler, nd node.Coder) error {
	var buf bytes.Buffer
	node.CodeW(&buf, nd, node.CodeWithPrefix("\t"), node.CodeFormat())
	parsed, err := parser.Parse(buf.String(), "", nil, nil)
	if err != nil {
		return err
	}
	return giomCompileStmts(c, parsed.Stmts)
}

func giomAssignToken(op string) token.Token {
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
