// Package gadbridge exposes Gad's format, diagnose and run capabilities behind
// a small, JSON-friendly API. It is the shared core used by both the HTTP
// server (web/server) and the WebAssembly module (web/wasm), so the editor
// integration behaves identically regardless of backend.
package gadbridge

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
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

// Format formats src as plain Gad with the canonical formatter, preserving
// comments and doc comments. On a parse error the source is returned unchanged
// together with the diagnostics. For dialect-aware formatting use FormatSource.
func Format(src string) FormatResult { return FormatSource(src, "") }

// FormatSource formats src according to sourceType, keeping the same dialect:
// "gadx" formats Gadx (indentation) source, "gadTemplate"/"template" formats a
// mixed template, and "gad"/"" formats plain Gad. On a parse error the source is
// returned unchanged with the diagnostics.
//
// A leading `#!…` shebang line (executable scripts) is preserved verbatim: it is
// split off before formatting and prepended to the result, so `gad fmt` round-
// trips it in every dialect (the parser/compiler already ignores it at run time).
func FormatSource(src, sourceType string) FormatResult {
	shebang, rest := SplitShebang(src)
	res := formatByType(rest, sourceType)
	if shebang != "" && res.OK {
		res.Source = shebang + res.Source
	}
	return res
}

// formatByType dispatches to the per-dialect formatter.
func formatByType(src, sourceType string) FormatResult {
	switch sourceType {
	case "gadx":
		return formatGadx(src)
	case "gadTemplate", "template":
		return formatGad(src, true)
	default:
		return formatGad(src, false)
	}
}

// SplitShebang splits a leading `#!…` line (including its trailing newline) from
// src, returning ("", src) when there is none.
func SplitShebang(src string) (shebang, rest string) {
	if !strings.HasPrefix(src, "#!") {
		return "", src
	}
	if i := strings.IndexByte(src, '\n'); i >= 0 {
		return src[:i+1], src[i+1:]
	}
	return src + "\n", "" // shebang-only file: keep it, add the newline
}

// formatGad formats plain Gad (or a mixed template when mixed is set), preserving
// comments. ParseComments collects the comment groups so CodeWithComments can
// re-emit them with the nodes they are attached to.
func formatGad(src string, mixed bool) FormatResult {
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
	out := node.Code(file.Stmts,
		node.CodeWithFlags(node.CodeWriteContextFlagFormat),
		node.CodeWithPrefix("\t"),
		node.CodeWithComments(srcFile, file.Comments),
	)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out += "\n"
	}
	return FormatResult{OK: true, Source: out}
}

// GadxFormatOptions configures the Gadx source formatter.
type GadxFormatOptions struct {
	// Indent is the indentation unit written per nesting level (default "\t").
	Indent string
	// EmbedFlags are the GAD formatting flags applied to embedded GAD code
	// (attribute expressions, conditions, `~` code, declaration values). Zero
	// keeps the column-aware NEW_LINE_CALC default.
	EmbedFlags node.CodeWriteContextFlag
	// MaxColumns is the column budget for embedded GAD (0 uses the default).
	MaxColumns int
}

// FormatGadx formats Gadx (.gadx) source with the given options, re-emitting it
// in Gadx syntax (tags, components, indentation) and formatting embedded GAD code
// per opts. src must already have any shebang stripped (see SplitShebang).
func FormatGadx(src string, opts GadxFormatOptions) FormatResult {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(sourceName+".gadx", -1, []byte(src))
	f, err := gadxparser.NewParser(srcFile).ParseFile()
	if err != nil {
		return FormatResult{OK: false, Source: src, Diagnostics: errorDiagnostics(err)}
	}
	var buf bytes.Buffer
	cctx := gadxnode.NewGadxCodeContext(&buf)
	if opts.Indent != "" {
		cctx.Prefix = opts.Indent
	}
	if opts.EmbedFlags != 0 {
		cctx.EmbedFlags = opts.EmbedFlags
	}
	cctx.MaxColumns = opts.MaxColumns
	f.WriteGadx(cctx)
	out := buf.String()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out += "\n"
	}
	return FormatResult{OK: true, Source: out}
}

// formatGadx formats Gadx source with the default options (used by FormatSource).
func formatGadx(src string) FormatResult {
	return FormatGadx(src, GadxFormatOptions{})
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

// EvalSource builds a runnable script that evaluates expr (rendered by render,
// "str" or "repr") in the context of an optional prelude. Top-level `return`
// statements in the prelude are dropped so the file's definitions stay in scope
// but the file's own return value does not short-circuit the evaluation. An
// unparseable prelude is used as-is so the compile error surfaces.
func EvalSource(prelude, expr, render string) string {
	body := ""
	if strings.TrimSpace(prelude) != "" {
		if file, err := parseSource(prelude); err == nil {
			var kept node.Stmts
			for _, s := range file.Stmts {
				if _, ok := s.(*node.ReturnStmt); ok {
					continue
				}
				kept = append(kept, s)
			}
			body = node.Code(kept,
				node.CodeWithFlags(node.CodeWriteContextFlagFormat),
				node.CodeWithPrefix("\t"))
		} else {
			body = prelude
		}
	}
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body + "return " + render + "(" + expr + ")\n"
}

// DocComment is one rendered doc comment (`///`, `/**` or `/***`) from a source
// file: its 1-based start line, kind, the following code line as a title (when
// any) and the Markdown content.
type DocComment struct {
	Line    int    `json:"line"`
	Kind    string `json:"kind"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// DocComments extracts the doc comments from src, in source order. A parse error
// yields whatever comments were collected before it (best effort), so the panel
// still shows docs for a file being edited.
func DocComments(src string) []DocComment {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(sourceName, -1, []byte(src))
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	file, _ := parser.NewParserWithOptions(srcFile, po, nil).ParseFile()
	if file == nil {
		return nil
	}
	lines := strings.Split(src, "\n")
	var out []DocComment
	for _, g := range file.Comments {
		kind, content, ok := docContent(g)
		if !ok {
			continue
		}
		out = append(out, DocComment{
			Line:    fileSet.Position(g.Pos()).Line,
			Kind:    kind,
			Title:   nextCodeLine(lines, fileSet.Position(g.End()).Line),
			Content: content,
		})
	}
	return out
}

// docContent classifies a comment group as a doc comment and returns its kind
// ("single"/"block"/"root") and Markdown content (markers stripped). ok is false
// for non-doc comments.
func docContent(g *ast.CommentGroup) (kind, content string, ok bool) {
	if g == nil || len(g.List) == 0 {
		return "", "", false
	}
	first := g.List[0].Text
	switch {
	case strings.HasPrefix(first, "/***"):
		return "root", trimFence(first, "/***", "***/"), true
	case strings.HasPrefix(first, "/**"):
		return "block", trimFence(first, "/**", "**/"), true
	case strings.HasPrefix(first, "///") && !strings.HasPrefix(first, "////"):
		ls := make([]string, len(g.List))
		for i, c := range g.List {
			ls[i] = strings.TrimPrefix(strings.TrimPrefix(c.Text, "///"), " ")
		}
		return "single", strings.Join(ls, "\n"), true
	}
	return "", "", false
}

// trimFence returns the inner text of a fenced block doc, dropping the opening
// (`/**`/`/***`) and closing (`**/`/`***/`) fence.
func trimFence(text, open, close string) string {
	body := strings.TrimPrefix(text, open)
	body = strings.TrimSuffix(body, close)
	return strings.Trim(body, "\n")
}

// nextCodeLine returns the first non-empty, non-comment source line at or after
// the 1-based line afterLine+1 (trimmed, truncated), as a doc-entry title.
func nextCodeLine(lines []string, afterLine int) string {
	for i := afterLine; i < len(lines); i++ {
		s := strings.TrimSpace(lines[i])
		if s == "" || strings.HasPrefix(s, "//") || strings.HasPrefix(s, "/*") || strings.HasPrefix(s, "**/") || strings.HasPrefix(s, "***/") {
			continue
		}
		if len(s) > 80 {
			s = s[:80] + "…"
		}
		return s
	}
	return ""
}

// InspectEntry is one child of an inspected container value.
type InspectEntry struct {
	Key        string `json:"key"`        // display key (dict key / array index)
	Accessor   string `json:"accessor"`   // Gad index suffix to reach it from the parent (`["a"]`, `[0]`)
	Type       string `json:"type"`       // child type name
	Value      string `json:"value"`      // child str(), truncated
	Expandable bool   `json:"expandable"` // child is itself a container
}

// InspectResult describes a value for the tree navigator: its type, rendered
// value and (for containers) its immediate children.
type InspectResult struct {
	Type       string         `json:"type"`
	Value      string         `json:"value"`
	Expandable bool           `json:"expandable"`
	Entries    []InspectEntry `json:"entries"`
}

const inspectValueMax = 200

// InspectObject describes obj for the tree navigator. When obj is a container
// (any ItemsGetter: dict, array, keyValueArray, module namespace, …) its
// immediate children are enumerated with a Gad accessor so the caller can drill
// in by appending the accessor to the parent expression.
func InspectObject(vm *gad.VM, obj gad.Object) InspectResult {
	res := InspectResult{
		Type:       objectTypeName(obj),
		Value:      truncate(objectToString(obj), inspectValueMax),
		Expandable: isExpandable(obj),
	}
	ig, ok := obj.(gad.ItemsGetter)
	if !ok {
		return res
	}
	_, isArray := obj.(gad.Array)
	_ = ig.Items(vm, func(i int, item *gad.KeyValue) error {
		key, accessor := inspectKey(item.K, i, isArray)
		res.Entries = append(res.Entries, InspectEntry{
			Key:        key,
			Accessor:   accessor,
			Type:       objectTypeName(item.V),
			Value:      truncate(objectToString(item.V), inspectValueMax),
			Expandable: isExpandable(item.V),
		})
		return nil
	})
	return res
}

// inspectKey renders a child's display key and the Gad accessor that reaches it
// from the parent expression. Array elements index by position; map-like keys
// index by their value (string keys quoted, integer keys bare).
func inspectKey(k gad.Object, i int, isArray bool) (display, accessor string) {
	if isArray {
		s := intToString(i)
		return s, "[" + s + "]"
	}
	switch kv := k.(type) {
	case gad.Str:
		return string(kv), "[" + quoteGad(string(kv)) + "]"
	case gad.Int:
		s := k.ToString()
		return s, "[" + s + "]"
	default:
		return k.ToString(), ""
	}
}

func isExpandable(o gad.Object) bool {
	if o == nil {
		return false
	}
	_, ok := o.(gad.ItemsGetter)
	return ok
}

func objectTypeName(o gad.Object) string {
	if o == nil || o == gad.Nil {
		return "nil"
	}
	return o.Type().Name()
}

func objectToString(o gad.Object) string {
	if o == nil {
		return "nil"
	}
	return o.ToString()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func quoteGad(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func intToString(i int) string {
	return gad.Int(i).ToString()
}

// Diagnose returns the syntax and compile diagnostics for src (empty when the
// source is valid).
func Diagnose(src string) []Diagnostic { return DiagnoseSource(src, "gad") }

// DiagnoseSource compiles src in the given dialect and returns any positioned
// diagnostics. sourceType is "gadx", "gadTemplate" (or "template"), or "gad"
// (default) — the latter two matter for `.gadt`/`.gadx` files, whose template
// syntax (`{% … %}`, tags) is a parse error under plain Gad.
func DiagnoseSource(src, sourceType string) []Diagnostic {
	if _, _, err := compileSource(src, sourceType); err != nil {
		return errorDiagnostics(err)
	}
	return nil
}

// EvalResult is the outcome of evaluating a single expression (the IDE's
// Evaluate panel): the rendered value, or an error, plus any prelude output.
type EvalResult struct {
	OK     bool   `json:"ok"`
	Value  string `json:"value"`
	Error  string `json:"error"`
	Stdout string `json:"stdout"`
}

// EvalExpr evaluates expr in the context of an optional source prelude (the
// file's top-level definitions), rendering the result with repr() or str(). It
// runs in a fresh VM — not a paused debug frame.
func EvalExpr(source, expr string, repr bool) EvalResult {
	render := "str"
	if repr {
		render = "repr"
	}
	res := RunSource(EvalSource(source, expr, render), "gad")
	out := EvalResult{OK: res.OK, Stdout: res.Stdout}
	if res.OK {
		out.Value = res.Result
		return out
	}
	out.Error = res.Stderr
	if out.Error == "" && len(res.Diagnostics) > 0 {
		d := res.Diagnostics[0]
		out.Error = fmt.Sprintf("%d:%d %s", d.Line, d.Column, d.Message)
	}
	return out
}

// InspectSource evaluates expr with source's top-level definitions in scope (a
// fresh VM, no debug session) and returns its tree-navigator description. It is
// the session-less counterpart of DebugManager.Inspect, for inspecting a value
// outside a paused frame.
func InspectSource(source, expr string) (InspectResult, error) {
	cr, builtins, err := compileSource(EvalSource(source, expr, ""), "gad")
	if err != nil {
		return InspectResult{}, err
	}
	var stdout, stderr bytes.Buffer
	obj, err := gad.NewVM(builtins.Build(), cr.Bytecode).SetRecover(true).
		RunOpts(&gad.RunOpts{StdOut: &stdout, StdErr: &stderr})
	if err != nil {
		return InspectResult{}, err
	}
	return InspectObject(nil, obj), nil
}

// Run compiles and executes src as plain Gad. See RunSource for `.gadt`/`.gadx`.
func Run(src string) RunResult { return RunSource(src, "gad") }

// RunSource compiles and executes src in the given dialect ("gadx",
// "gadTemplate"/"template", or "gad"), capturing stdout/stderr and the return
// value. A gadx `@main` template returns its rendered tree as a gadx.Element,
// which is written to stdout so the output panel shows the HTML.
func RunSource(src, sourceType string) RunResult { return RunSourceArgs(src, sourceType, nil, "") }

// RunSourceArgs is RunSource with command-line arguments passed to the script
// (reachable via its top-level `param` — the same convention `gad run` uses) and
// a tagEncode mode for gadx output: "" (or "html"/"render") renders the returned
// gadx.Element as HTML to stdout (the default), while "json" / "yaml" encode the
// element's structured data ({tag, attrs, children}) to stdout instead.
func RunSourceArgs(src, sourceType string, cmdArgs []string, tagEncode string) RunResult {
	cr, builtins, err := compileSource(src, sourceType)
	if err != nil {
		return RunResult{OK: false, Diagnostics: errorDiagnostics(err)}
	}

	// Surface compiler warnings in the STDERR panel, before program output.
	var stdout, stderr bytes.Buffer
	for _, w := range cr.Warnings {
		stderr.WriteString(w.Error())
		stderr.WriteByte('\n')
	}
	vm := gad.NewVM(builtins.Build(), cr.Bytecode).SetRecover(true)
	runOpts := &gad.RunOpts{StdOut: &stdout, StdErr: &stderr}
	if len(cmdArgs) > 0 {
		runOpts.Args, runOpts.NamedArgs = gad.ParseArgsToRunOpts(cmdArgs)
	}
	ret, runErr := vm.RunOpts(runOpts)
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
	if el, ok := ret.(gadx.Element); ok {
		switch strings.ToLower(tagEncode) {
		case "json", "yaml":
			enc, eerr := encodeElement(el, tagEncode)
			if eerr != nil {
				res.Stderr += eerr.Error()
			} else {
				stdout.WriteString(enc)
			}
			res.Stdout = stdout.String()
		default:
			// Default: render the returned tree as HTML into stdout.
			if _, werr := el.WriteTo(vm, &stdout); werr == nil {
				res.Stdout = stdout.String()
			}
		}
	} else if ret != nil && ret != gad.Nil {
		res.Result = ret.ToString()
	}
	return res
}

// encodeElement serializes a gadx.Element's structured data ({tag, attrs,
// children}) to JSON or YAML (see gadx.EncodeElement).
func encodeElement(el gadx.Element, format string) (string, error) {
	out, err := gadx.EncodeElement(el, format)
	return string(out), err
}

// sourceKind maps a bridge/wasm dialect string ("gad"/"gadTemplate"/"gadx", plus
// the "template" and "gadt" aliases) to the core gad.SourceKind, so the bridge
// selects dialects through the same enum the compiler and importers use.
func sourceKind(sourceType string) gad.SourceKind {
	switch sourceType {
	case "gadx":
		return gad.SourceKindGadx
	case "gadTemplate", "template", "gadt":
		return gad.SourceKindGadt
	default:
		return gad.SourceKindGad
	}
}

// compileSource builds the bytecode for src in the given dialect, returning the
// result and the builtins the VM must run with (gadx adds its namespace).
func compileSource(src, sourceType string) (*gad.CompileResult, *gad.Builtins, error) {
	builtins := gad.NewBuiltins()
	opts := gad.CompileOptions{}
	switch sourceKind(sourceType) {
	case gad.SourceKindGadx:
		// The .gadx ModuleFile selects gad's native Gadx front-end.
		builtins = gadx.AppendBuiltins(builtins)
		opts.ModuleFile = sourceName + ".gadx"
	case gad.SourceKindGadt:
		opts.ParserOptions.Mode |= parser.ParseMixed
		opts.ScannerOptions.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
		opts.ScannerOptions.MixedDelimiter = parser.DefaultMixedDelimiter
	}
	st := gad.NewSymbolTable(builtins.NameSet)
	cr, err := gad.Compile(st, []byte(src), opts)
	return cr, builtins, err
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
