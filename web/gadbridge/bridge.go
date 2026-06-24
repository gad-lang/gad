// Package gadbridge exposes Gad's format, diagnose and run capabilities behind
// a small, JSON-friendly API. It is the shared core used by both the HTTP
// server (web/server) and the WebAssembly module (web/wasm), so the editor
// integration behaves identically regardless of backend.
package gadbridge

import (
	"bytes"
	"errors"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// Severity classifies a diagnostic.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Diagnostic is a positioned message for the editor. Lines and columns are
// 1-based, matching Gad's parser positions.
type Diagnostic struct {
	Line     int      `json:"line"`
	Column   int      `json:"column"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
}

// FormatResult is the outcome of formatting a source.
type FormatResult struct {
	OK          bool         `json:"ok"`
	Source      string       `json:"source"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// RunResult is the outcome of compiling and executing a source.
type RunResult struct {
	OK          bool         `json:"ok"`
	Stdout      string       `json:"stdout"`
	Stderr      string       `json:"stderr"`
	Result      string       `json:"result"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

const sourceName = "(editor)"

// parseSource parses src into an AST file without touching the filesystem.
func parseSource(src string) (*parser.File, error) {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(sourceName, -1, []byte(src))
	return parser.NewParserWithOptions(srcFile, nil, nil).ParseFile()
}

// Format formats src with the canonical formatter. On a parse error the source
// is returned unchanged together with the diagnostics.
func Format(src string) FormatResult {
	file, err := parseSource(src)
	if err != nil {
		return FormatResult{OK: false, Source: src, Diagnostics: errorDiagnostics(err)}
	}
	out := node.Code(file.Stmts,
		node.CodeWithFlags(node.CodeWriteContextFlagFormat),
		node.CodeWithPrefix("\t"),
	)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out += "\n"
	}
	return FormatResult{OK: true, Source: out}
}

// Transpile rewrites a Gad source into plain Gad with template text and
// `{%= … %}` expressions turned into write(...) calls, using the default
// gad.TranspileOptions unless overridden. When mixed is set the source is parsed
// in mixed (template) mode — the form used for `.gadt` files. On a parse error
// the source is returned unchanged with the diagnostics. Mirrors Format.
func Transpile(src string, mixed bool, opts *node.TranspileOptions) FormatResult {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(sourceName, -1, []byte(src))
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	var so *parser.ScannerOptions
	if mixed {
		po.Mode |= parser.ParseMixed
		so = &parser.ScannerOptions{Mode: parser.ScanMixed | parser.ScanConfigDisabled}
	}
	file, err := parser.NewParserWithOptions(srcFile, po, so).ParseFile()
	if err != nil {
		return FormatResult{OK: false, Source: src, Diagnostics: errorDiagnostics(err)}
	}
	if opts == nil {
		opts = gad.TranspileOptions()
	}
	out := node.Code(file.Stmts,
		node.CodeWithFlags(node.CodeWriteContextFlagFormat),
		node.CodeWithPrefix("\t"),
		node.CodeTranspile(opts),
	)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out += "\n"
	}
	return FormatResult{OK: true, Source: out}
}

// Diagnose returns the syntax and compile diagnostics for src (empty when the
// source is valid).
func Diagnose(src string) []Diagnostic {
	if _, err := parseSource(src); err != nil {
		return errorDiagnostics(err)
	}
	if _, _, err := compile(src); err != nil {
		return errorDiagnostics(err)
	}
	return nil
}

// Run compiles and executes src, capturing stdout/stderr and the return value.
func Run(src string) RunResult {
	_, bc, err := compile(src)
	if err != nil {
		return RunResult{OK: false, Diagnostics: errorDiagnostics(err)}
	}

	var stdout, stderr bytes.Buffer
	ret, runErr := gad.NewVM(gad.NewBuiltins().Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{
		StdOut: &stdout,
		StdErr: &stderr,
	})
	res := RunResult{
		OK:     runErr == nil,
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if runErr != nil {
		res.Diagnostics = errorDiagnostics(runErr)
		if res.Stderr == "" {
			res.Stderr = runErr.Error()
		}
		return res
	}
	if ret != nil && ret != gad.Nil {
		res.Result = ret.ToString()
	}
	return res
}

// compile builds the bytecode for src with a fresh builtins/symbol table.
func compile(src string) (*parser.File, *gad.Bytecode, error) {
	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)
	return gad.Compile(st, []byte(src), gad.CompileOptions{})
}

// ErrorDiagnostics converts a Gad parse/compile/runtime error into positioned
// diagnostics, for callers (such as the IDE backend) that run Gad with custom
// options but still want editor-friendly error positions.
func ErrorDiagnostics(err error) []Diagnostic { return errorDiagnostics(err) }

// errorDiagnostics converts a Gad parse/compile/runtime error into positioned
// diagnostics. parser.ErrorList yields one diagnostic per error; positioned
// single errors (parser/compiler) are mapped to their line/column; anything
// else is reported at the start of the file.
func errorDiagnostics(err error) []Diagnostic {
	var list parser.ErrorList
	if errors.As(err, &list) {
		out := make([]Diagnostic, len(list))
		for i, e := range list {
			out[i] = Diagnostic{
				Line:     e.Pos.Line,
				Column:   e.Pos.Column,
				Message:  e.Msg,
				Severity: SeverityError,
			}
		}
		return out
	}

	var perr *parser.Error
	if errors.As(err, &perr) {
		return []Diagnostic{{
			Line:     perr.Pos.Line,
			Column:   perr.Pos.Column,
			Message:  perr.Msg,
			Severity: SeverityError,
		}}
	}

	var cerr *gad.CompilerError
	if errors.As(err, &cerr) && cerr.FileSet != nil && cerr.Node != nil {
		pos := cerr.FileSet.Position(cerr.Node.Pos())
		return []Diagnostic{{
			Line:     pos.Line,
			Column:   pos.Column,
			Message:  cerr.Err.Error(),
			Severity: SeverityError,
		}}
	}

	return []Diagnostic{{Line: 1, Column: 1, Message: err.Error(), Severity: SeverityError}}
}
