package node

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// DefaultMaxColumns is the line-width budget used by the NEW_LINE_CALC
// formatting mode when no explicit MaxColumns is set.
const DefaultMaxColumns = 80

// mixedEndTagRe matches a template block terminator `{% end %}` (with optional
// `-` trim markers and any surrounding spaces) so it can be normalized to a
// canonical `{% end %}`.
var mixedEndTagRe = regexp.MustCompile(`(\{%-?)\s*end\s*(-?%\})`)

// normalizeMixedEndTags rewrites template block terminators to `{% end %}`
// (preserving the `-` trim markers). It is a no-op for sources without `{%`.
func normalizeMixedEndTags(s string) string {
	if !strings.Contains(s, "{%") {
		return s
	}
	return mixedEndTagRe.ReplaceAllString(s, "${1} end ${2}")
}

type Coder interface {
	WriteCode(ctx *CodeWriteContext)
}

type CodeWriter interface {
	io.Writer
	WriteString(s ...string)
	WriteSingleByte(b byte)
	WriteRune(b rune)
	WriteLine(s ...string)
	WriteLines(l ...string)
}

type cw struct {
	io.Writer
	// col is the current column (runes written since the last newline); max is
	// the widest column reached; multiline records whether a newline was seen.
	col, max  int
	multiline bool
}

// Write tracks the cursor column so the NEW_LINE_CALC formatter can measure how
// wide a construct renders.
func (w *cw) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	s := string(p)
	for {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			w.col += utf8.RuneCountInString(s)
			break
		}
		w.col += utf8.RuneCountInString(s[:i])
		if w.col > w.max {
			w.max = w.col
		}
		w.multiline = true
		w.col = 0
		s = s[i+1:]
	}
	if w.col > w.max {
		w.max = w.col
	}
	return
}

func (w *cw) WriteString(s ...string) {
	for _, s := range s {
		w.Write([]byte(s))
	}
}

func (w *cw) WriteRune(r rune) {
	w.WriteString(string([]rune{r}))
}

func (w *cw) WriteSingleByte(c byte) {
	w.Write([]byte{c})
}

func (w *cw) WriteLine(s ...string) {
	w.WriteString(s...)
	w.Write([]byte{'\n'})
}

func (w *cw) WriteLines(l ...string) {
	for _, s := range l {
		w.WriteLine(s)
	}
}

func NewCodeWriter(w io.Writer) CodeWriter {
	return &cw{Writer: w}
}

type TranspileOptions struct {
	RawStrFuncStart string
	RawStrFuncEnd   string
	WriteFunc       string
}

type CodeWriteContextFlag uint16

func (b *CodeWriteContextFlag) Set(flag CodeWriteContextFlag) *CodeWriteContextFlag {
	*b = *b | flag
	return b
}
func (b *CodeWriteContextFlag) Clear(flag CodeWriteContextFlag) *CodeWriteContextFlag {
	*b = *b &^ flag
	return b
}
func (b *CodeWriteContextFlag) Toggle(flag CodeWriteContextFlag) *CodeWriteContextFlag {
	*b = *b ^ flag
	return b
}
func (b CodeWriteContextFlag) Has(flag ...CodeWriteContextFlag) bool {
	for _, f := range flag {
		if b&f != 0 {
			return true
		}
	}
	return false
}

const (
	CodeWriteContextFlagNone CodeWriteContextFlag = 1 << iota
	CodeWriteContextFlagFormatArrayItemInNewLine
	CodeWriteContextFlagFormatDictItemInNewLine
	CodeWriteContextFlagFormatKeyValueArrayItemInNewLine
	CodeWriteContextFlagFormatCallParamsInNewLine
	CodeWriteContextFlagFormatParemValuesInNewLine
	CodeWriteContextFlagFormatDeclItemInNewLine
	CodeWriteContextFlagFormatMatchExprArmsInNewLine
	CodeWriteContextFlagFormatMatchStmtArmsInNewLine
	CodeWriteContextFlagFormatMethodInterfaceInNewLine
	// CodeWriteContextFlagFormatNewLineCalc (NEW_LINE_CALC) switches the
	// formatter from "force all to new lines" to column-aware wrapping: a list
	// construct stays inline unless it would overflow ctx.MaxColumns. It is not
	// part of CodeWriteContextFlagFormat.
	CodeWriteContextFlagFormatNewLineCalc

	CodeWriteContextFlagFormat = CodeWriteContextFlagFormatArrayItemInNewLine |
		CodeWriteContextFlagFormatDictItemInNewLine |
		CodeWriteContextFlagFormatKeyValueArrayItemInNewLine |
		CodeWriteContextFlagFormatCallParamsInNewLine |
		CodeWriteContextFlagFormatParemValuesInNewLine |
		CodeWriteContextFlagFormatDeclItemInNewLine |
		CodeWriteContextFlagFormatMatchExprArmsInNewLine |
		CodeWriteContextFlagFormatMatchStmtArmsInNewLine |
		CodeWriteContextFlagFormatMethodInterfaceInNewLine
)

type CodeWriteSkiper interface {
	SkipCode(ctx *CodeWriteContext) bool
}

type CodeWriteContext struct {
	Stack  []ast.Node
	Depth  int
	Prefix string
	Flags  CodeWriteContextFlag
	// MaxColumns is the line-width budget for the NEW_LINE_CALC mode (0 uses
	// DefaultMaxColumns).
	MaxColumns int
	Transpile  *TranspileOptions
	CodeWriter

	// srcFile and comments support preserving source comments during
	// formatting. comments is flattened and sorted by position; commentIdx is
	// the cursor into it, advanced (across nested statement lists) as comments
	// are emitted in position order.
	srcFile    *source.File
	comments   []*ast.Comment
	commentIdx int

	// docClaim holds the doc comments (`/?`, `/??`, `/???`) that are emitted by
	// their owning AST node (via its Doc field) rather than by position. Claimed
	// comments are filtered out of `comments` so the position machinery does not
	// also emit them; this lets a lead doc travel with its node through
	// declaration merges and reordering. Only lead docs are claimed; trailing/
	// inline docs stay with the position machinery.
	docClaim map[*ast.Comment]bool
}

type CodeOption func(ctx *CodeWriteContext)

func CodeWithPrefix(prefix string) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Prefix = prefix
	}
}

func CodeWithFlags(flag CodeWriteContextFlag) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Flags.Set(flag)
	}
}

func CodeFormat() CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Flags |= CodeWriteContextFlagFormat
	}
}

// CodeNewLineCalc enables the column-aware (NEW_LINE_CALC) formatting mode with
// the given column budget (<= 0 uses DefaultMaxColumns). List constructs stay
// inline unless they would overflow the budget.
func CodeNewLineCalc(maxColumns int) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Flags |= CodeWriteContextFlagFormat | CodeWriteContextFlagFormatNewLineCalc
		ctx.MaxColumns = maxColumns
	}
}

func CodeTranspile(v *TranspileOptions) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Transpile = v
	}
}

// CodeWithComments threads the source comments (collected by parsing with
// ParseComments) into the formatter so they are preserved in the output. The
// file is required for line lookups (to tell trailing same-line comments from
// own-line ones).
func CodeWithComments(f *source.File, groups []*ast.CommentGroup) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.srcFile = f
		for _, g := range groups {
			ctx.comments = append(ctx.comments, g.List...)
		}
		sort.SliceStable(ctx.comments, func(i, j int) bool {
			return ctx.comments[i].Pos() < ctx.comments[j].Pos()
		})
	}
}

// hasComments reports whether comment preservation is active.
func (ctx *CodeWriteContext) hasComments() bool {
	return ctx.srcFile != nil && ctx.commentIdx < len(ctx.comments)
}

// peekComment returns the next un-emitted comment, or nil.
func (ctx *CodeWriteContext) peekComment() *ast.Comment {
	if ctx.commentIdx < len(ctx.comments) {
		return ctx.comments[ctx.commentIdx]
	}
	return nil
}

// lineOf returns the 1-based source line of pos (0 when unknown).
func (ctx *CodeWriteContext) lineOf(pos source.Pos) int {
	if ctx.srcFile == nil {
		return 0
	}
	return source.MustFileLine(ctx.srcFile, pos)
}

// wroteAny reports whether any output has been produced yet.
func (ctx *CodeWriteContext) wroteAny() bool {
	if c, ok := ctx.CodeWriter.(*cw); ok {
		return c.col > 0 || c.multiline
	}
	return false
}

// flushRemainingComments writes any comments not yet emitted, each on its own
// line. Used at the very end of the top-level statement list for file-trailing
// comments (and comment-only files).
func (ctx *CodeWriteContext) flushRemainingComments() {
	for c := ctx.peekComment(); c != nil; c = ctx.peekComment() {
		if ctx.wroteAny() {
			ctx.WriteSemi()
		}
		ctx.WriteString(c.Text)
		ctx.commentIdx++
	}
}

// claimLeadDocs records the lead doc comments attached to top-level decl/func
// nodes (and the value specs inside a group) so each is emitted by its owning
// node instead of by position, then removes them from the position-based comment
// stream. Only lead docs — those appearing before their node — are claimed;
// trailing/inline docs are left to the position machinery.
func (ctx *CodeWriteContext) claimLeadDocs(stmts []Stmt) {
	for _, s := range stmts {
		switch t := s.(type) {
		case *DeclStmt:
			if gd, _ := t.Decl.(*GenDecl); gd != nil {
				ctx.claimLeadDoc(gd.Doc, gd)
				for _, sp := range gd.Specs {
					if vs, _ := sp.(*ValueSpec); vs != nil {
						// A spec doc is emitted by the spec whether it is a lead
						// doc (before the ident) or a trailing inline doc, since
						// the position machinery does not reach inside a group.
						ctx.claimDoc(vs.Doc)
					}
				}
			}
		case *FuncStmt:
			if t.Func != nil {
				ctx.claimLeadDoc(t.Func.Doc, t.Func)
			}
		case *FuncWithMethodsStmt:
			ctx.claimLeadDoc(t.Doc, &t.FuncWithMethodsExpr)
			for _, m := range t.Methods {
				ctx.claimLeadDoc(m.Doc, m)
			}
		case *PropStmt:
			ctx.claimLeadDoc(t.Doc, &t.PropExpr)
			for _, m := range t.Methods {
				ctx.claimLeadDoc(m.Doc, m)
			}
		case *MethodInterfaceStmt:
			ctx.claimLeadDoc(t.Doc, &t.MethodInterfaceExpr)
			for _, h := range t.Headers {
				ctx.claimLeadDoc(h.Doc, h)
			}
		case *ClassStmt:
			// The class lead doc precedes `class Name`; the body docs are emitted
			// by their own nodes (the position machinery does not reach inside).
			ctx.claimLeadDoc(t.Doc, &t.ClassExpr)
			ctx.claimClassBodyDocs(&t.ClassExpr)
		case *AssignStmt:
			// expression-form class, e.g. `X := class { … }`: its lead doc stays
			// with the statement (position machinery), but its body docs are
			// claimed so they travel with the field/member nodes.
			for _, rhs := range t.RHS {
				if ce, _ := rhs.(*ClassExpr); ce != nil {
					ctx.claimClassBodyDocs(ce)
				}
			}
		case *ExprStmt:
			if ce, _ := t.Expr.(*ClassExpr); ce != nil {
				ctx.claimClassBodyDocs(ce)
			}
		}
	}
	if len(ctx.docClaim) == 0 {
		return
	}
	filtered := ctx.comments[:0]
	for _, c := range ctx.comments {
		if !ctx.docClaim[c] {
			filtered = append(filtered, c)
		}
	}
	ctx.comments = filtered
}

// claimClassBodyDocs claims the doc comments of a class body — fields, the
// `props`/`new`/`methods` group keywords, the property/method entries and their
// accessor/overload methods — so each is emitted in place by its own node
// instead of being flushed by position at the end of the file.
func (ctx *CodeWriteContext) claimClassBodyDocs(e *ClassExpr) {
	for _, f := range e.Fields {
		ctx.claimDoc(f.Doc)
	}
	ctx.claimDoc(e.PropsDoc)
	for _, m := range e.Props {
		ctx.claimDoc(m.Doc)
		for _, fm := range m.Methods {
			ctx.claimDoc(fm.Doc)
		}
	}
	ctx.claimDoc(e.NewDoc)
	for _, fm := range e.New {
		ctx.claimDoc(fm.Doc)
	}
	ctx.claimDoc(e.MethodsDoc)
	for _, m := range e.Methods {
		ctx.claimDoc(m.Doc)
		for _, fm := range m.Methods {
			ctx.claimDoc(fm.Doc)
		}
	}
}

// claimLeadDoc claims g as the lead doc of n when g precedes n; trailing/inline
// docs (g.End() > n.Pos()) are ignored.
func (ctx *CodeWriteContext) claimLeadDoc(g *ast.CommentGroup, n ast.Node) {
	if g == nil || len(g.List) == 0 || g.End() > n.Pos() {
		return
	}
	ctx.claimDoc(g)
}

// claimDoc claims every comment of g for node-based emission.
func (ctx *CodeWriteContext) claimDoc(g *ast.CommentGroup) {
	if g == nil || len(g.List) == 0 {
		return
	}
	if ctx.docClaim == nil {
		ctx.docClaim = map[*ast.Comment]bool{}
	}
	for _, c := range g.List {
		ctx.docClaim[c] = true
	}
}

// isClaimedDoc reports whether g is a doc group claimed for node-based emission.
func (ctx *CodeWriteContext) isClaimedDoc(g *ast.CommentGroup) bool {
	if g == nil || ctx.docClaim == nil {
		return false
	}
	for _, c := range g.List {
		if !ctx.docClaim[c] {
			return false
		}
	}
	return true
}

// docLines returns the lines of g for emission. In a format mode the doc is
// reflowed (Markdown paragraphs re-wrapped, SINGLE<->BLOCK chosen by the column
// budget); otherwise the source lines are kept verbatim. Continuation lines
// carry no prefix — callers re-indent them to the current prefix.
func (ctx *CodeWriteContext) docLines(g *ast.CommentGroup) []string {
	if ctx.Flags.Has(CodeWriteContextFlagFormat) {
		if d, ok := parseDocComment(g); ok {
			return renderDocLines(d, ctx.maxColumns()-len(ctx.CurrentPrefix()))
		}
	}
	var lines []string
	for _, c := range g.List {
		lines = append(lines, strings.Split(c.Text, "\n")...)
	}
	return lines
}

// writeDocComment writes a doc comment group, re-indenting continuation lines to
// the current prefix so a block doc (`/??` … `??`) aligns with its construct.
func (ctx *CodeWriteContext) writeDocComment(g *ast.CommentGroup) {
	prefix := ctx.CurrentPrefix()
	for i, line := range ctx.docLines(g) {
		if i > 0 {
			ctx.WriteString("\n", prefix)
		}
		ctx.WriteString(line)
	}
}

// WriteLeadDoc emits g as a lead doc — the documented construct follows on its
// own (prefixed) line — when g has been claimed for node-based emission. It
// returns true when it emitted the doc. Nodes call this at the start of their
// WriteCode so the doc travels with the node.
func (ctx *CodeWriteContext) WriteLeadDoc(g *ast.CommentGroup) bool {
	if !ctx.isClaimedDoc(g) {
		return false
	}
	ctx.writeDocComment(g)
	ctx.WriteString("\n", ctx.CurrentPrefix())
	return true
}

// WriteTrailingDoc emits g as a trailing inline doc (` /// …`) on the current
// line when g is a claimed doc. Inline docs are kept on one line regardless of
// width. Returns true when it emitted the doc.
func (ctx *CodeWriteContext) WriteTrailingDoc(g *ast.CommentGroup) bool {
	if !ctx.isClaimedDoc(g) {
		return false
	}
	if ctx.Flags.Has(CodeWriteContextFlagFormat) {
		if d, ok := parseDocComment(g); ok {
			ctx.WriteString(" ", "/// "+strings.Join(strings.Fields(d.content), " "))
			return true
		}
	}
	ctx.WriteString(" ", g.List[0].Text)
	return true
}

// isLeadDoc reports whether g precedes n (a lead doc) rather than trailing it.
func isLeadDoc(g *ast.CommentGroup, n ast.Node) bool {
	return g != nil && len(g.List) > 0 && g.End() <= n.Pos()
}

func NewCodeWriteContext(codeWriter CodeWriter, opt ...CodeOption) *CodeWriteContext {
	ctx := &CodeWriteContext{CodeWriter: codeWriter}
	for _, opt := range opt {
		opt(ctx)
	}
	return ctx
}

func (ctx CodeWriteContext) WithoutPrefix() *CodeWriteContext {
	ctx.Prefix = ""
	return &ctx
}

func (ctx CodeWriteContext) Buffer(do func(ctx *CodeWriteContext)) string {
	var buf bytes.Buffer
	ctx.CodeWriter = NewCodeWriter(&buf)
	do(&ctx)
	return buf.String()
}

func (ctx *CodeWriteContext) HasPrefix() bool {
	return ctx.Prefix != ""
}

func (ctx *CodeWriteContext) CurrentPrefix() string {
	return strings.Repeat(ctx.Prefix, ctx.Depth)
}

func (ctx *CodeWriteContext) WritePrefix() {
	ctx.WriteString(ctx.CurrentPrefix())
}

func (ctx *CodeWriteContext) PrevPrefix() string {
	if ctx.Depth == 0 {
		return ""
	}
	return strings.Repeat(ctx.Prefix, ctx.Depth-1)
}

func (ctx *CodeWriteContext) WritePrevPrefix() {
	ctx.WriteString(ctx.PrevPrefix())
}

func (ctx *CodeWriteContext) WriteLine(s string) {
	ctx.WriteString(s)
	ctx.WriteString("\n")
}

func (ctx *CodeWriteContext) WritePrefixedLine() {
	ctx.WriteString("\n", ctx.CurrentPrefix())
}

func (ctx *CodeWriteContext) WriteSecondLine() {
	if ctx.HasPrefix() {
		ctx.WriteString("\n")
	}
}

func (ctx *CodeWriteContext) WriteSemi() {
	if !ctx.HasPrefix() {
		ctx.WriteString("; ")
	} else {
		ctx.WriteString("\n" + ctx.CurrentPrefix())
	}
}

func (ctx *CodeWriteContext) WriteSemiOrDoubleLine() {
	if !ctx.HasPrefix() {
		ctx.WriteString("; ")
	} else {
		ctx.WriteString("\n\n")
	}
}

func (ctx *CodeWriteContext) Printf(format string, args ...interface{}) {
	fmt.Fprintf(ctx.CodeWriter, format, args...)
}

func (ctx *CodeWriteContext) Top() ast.Node {
	return ctx.Stack[len(ctx.Stack)-1]
}

func (ctx *CodeWriteContext) Push(n ast.Node) {
	ctx.Stack = append(ctx.Stack, n)
}

func (ctx *CodeWriteContext) Pop() {
	ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
}

func (ctx *CodeWriteContext) With(n ast.Node, cb func() error) (err error) {
	ctx.Push(n)
	err = cb()
	ctx.Pop()
	return
}

func (ctx *CodeWriteContext) simplifyStmts(stmt []Stmt) (ret []Stmt) {
	l := len(stmt)

loop:
	for i := 0; i < l; i++ {
		if i > 0 {
			switch e := stmt[i].(type) {
			case *DeclStmt:
				if ge, _ := e.Decl.(*GenDecl); ge != nil {
					if last, _ := ret[len(ret)-1].(*DeclStmt); last != nil {
						if lge, _ := last.Decl.(*GenDecl); lge != nil {
							if ge.Tok == lge.Tok {
								// Preserve the merged decl's own lead doc by moving
								// it onto its first spec, so it is not lost when ge
								// is dropped into lge.
								if ge.Doc != nil && len(ge.Specs) > 0 {
									if vs, _ := ge.Specs[0].(*ValueSpec); vs != nil && vs.Doc == nil {
										vs.Doc = ge.Doc
									}
								}
								lge.Specs = append(lge.Specs, ge.Specs...)
								continue loop
							}
						}
					}
				}
			}
		}
		ret = append(ret, stmt[i])
	}
	return
}

// statement separator kinds used by WriteStmts.
const (
	sepNewline = iota // normal: a newline+indent (or "; " inline) between stmts
	sepSpace          // a single space (the code right after a `{%` tag)
	sepGlue           // no separator (template text/value segments)
)

func (ctx *CodeWriteContext) WriteStmts(stmt ...Stmt) {
	stmt = ctx.simplifyStmts(stmt)

	var (
		i     int
		sep   = sepNewline
		inTag bool // currently between a `{%` and its `%}`
		last  = len(stmt) - 1
	)

	emitLeadingSep := func() {
		switch sep {
		case sepSpace:
			ctx.WriteString(" ")
		case sepNewline:
			ctx.WriteSemi()
		}
	}

	Stmts(stmt).Each(func(x int, _ bool, s Stmt) {
		if skiper, _ := s.(CodeWriteSkiper); skiper != nil {
			if skiper.SkipCode(ctx) {
				return
			}
		}

		// When transpiling, the mixed segments become ordinary write(...) calls,
		// so they must be separated like normal statements (not glued/inlined as
		// template tags).
		transpiling := ctx.Transpile != nil

		// Own-line comments that precede this statement: each on its own line,
		// then the statement follows on a fresh line.
		if ctx.hasComments() {
			for c := ctx.peekComment(); c != nil && c.Pos() < s.Pos(); c = ctx.peekComment() {
				if i > 0 {
					emitLeadingSep()
				}
				ctx.WriteString(c.Text)
				ctx.commentIdx++
				sep = sepNewline
				i++
			}
		}

		// Leading separator. A `%}` terminator always hugs the preceding code
		// with a single space so the whole `{% … %}` tag stays on one line.
		if _, isEnd := s.(*CodeEndStmt); isEnd && !transpiling {
			ctx.WriteString(" ")
		} else if i > 0 {
			emitLeadingSep()
		}
		s.WriteCode(ctx)
		i++

		// Trailing comment(s) on the same source line as this statement stay
		// glued to it (` // ...`).
		if ctx.hasComments() {
			endLine := ctx.lineOf(s.End())
			for c := ctx.peekComment(); c != nil && ctx.lineOf(c.Pos()) == endLine; c = ctx.peekComment() {
				ctx.WriteString(" ", c.Text)
				ctx.commentIdx++
			}
		}

		// Separator for the NEXT statement.
		switch s.(type) {
		case *CodeBeginStmt:
			// The code after `{%` is kept on the same line, one space away.
			sep = sepSpace
			inTag = true
		case *CodeEndStmt:
			sep = sepGlue
			inTag = false
		case *ConfigStmt, *MixedTextStmt, *MixedValueStmt:
			if transpiling {
				// Transpiled write(...) statements need a real separator.
				sep = sepNewline
			} else {
				// Template segments carry their own (significant) whitespace, so
				// the next statement is glued to them without an inserted
				// separator.
				sep = sepGlue
			}
		case *ExprStmt:
			sep = sepNewline
		default:
			sep = sepNewline
			// Separate block/declaration statements from the next with a blank
			// line, except when inside a `{% … %}` tag (kept inline). Emit a bare
			// newline (no indentation) so the blank line never carries trailing
			// whitespace; the next statement's leading separator writes its own
			// indentation.
			if x < last && ctx.HasPrefix() && !inTag {
				ctx.WriteString("\n")
			}
		}
	})
}

// Column returns the current cursor column (0 when unknown).
func (ctx *CodeWriteContext) Column() int {
	if c, ok := ctx.CodeWriter.(*cw); ok {
		return c.col
	}
	return 0
}

// maxColumns resolves the effective line-width budget.
func (ctx *CodeWriteContext) maxColumns() int {
	if ctx.MaxColumns > 0 {
		return ctx.MaxColumns
	}
	return DefaultMaxColumns
}

// measure renders `do` to a throwaway writer that starts at startCol and
// reports the widest column reached and whether any newline was emitted, while
// leaving the real output untouched.
func (ctx *CodeWriteContext) measure(startCol int, do func()) (width int, multiline bool) {
	saved := ctx.CodeWriter
	m := &cw{Writer: io.Discard, col: startCol, max: startCol}
	ctx.CodeWriter = m
	do()
	ctx.CodeWriter = saved
	return m.max, m.multiline
}

// DecideNewLine reports whether a list construct's items should be written one
// per line. Without NEW_LINE_CALC it honours the per-construct force `flag`.
// With NEW_LINE_CALC it renders the items inline (separated by inlineSep) and
// wraps only when they would overflow MaxColumns (or already contain a
// newline). `closing` is the width of the trailing delimiter (e.g. 1 for `)`).
func (ctx *CodeWriteContext) DecideNewLine(flag CodeWriteContextFlag, count int, inlineSep string, closing int, do func(i int)) bool {
	return ctx.DecideNewLineFunc(flag, count, closing, func() {
		for i := 0; i < count; i++ {
			if i > 0 {
				ctx.WriteString(inlineSep)
			}
			do(i)
		}
	})
}

// DecideNewLineFunc is DecideNewLine for constructs whose inline form does not
// map to a uniform per-item callback: renderInline writes the whole inline body
// (to a throwaway writer during measurement).
func (ctx *CodeWriteContext) DecideNewLineFunc(flag CodeWriteContextFlag, count, closing int, renderInline func()) bool {
	if !ctx.Flags.Has(CodeWriteContextFlagFormatNewLineCalc) {
		return ctx.Flags.Has(flag)
	}
	if count <= 1 {
		return false
	}
	width, multiline := ctx.measure(ctx.Column(), renderInline)
	return multiline || width+closing > ctx.maxColumns()
}

func (ctx *CodeWriteContext) WriteItems(inNewLine bool, count int, do func(i int), done func(newLine bool)) {
	ctx.WriteItemsSep(inNewLine, count, ", ", ",", do, done)
}

func (ctx *CodeWriteContext) WriteItemsSep(inNewLine bool, count int, inlineSep, newLineSep string, do func(i int), done func(newLine bool)) {
	if count == 0 {
		return
	}

	last := count - 1

	if inNewLine {
		ctx.Depth++
		ctx.WriteSecondLine()
		for i := 0; i < count; i++ {
			ctx.WritePrefix()
			do(i)
			if i != last {
				ctx.WriteString(newLineSep)
				ctx.WriteSecondLine()
			}
		}
		if done != nil {
			done(inNewLine)
		}
		ctx.Depth--
	} else {
		for i := 0; i < count; i++ {
			do(i)
			if i != last {
				ctx.WriteString(inlineSep)
			}
		}
		if done != nil {
			done(inNewLine)
		}
	}
}

// WriteGreedy renders count items using the column-aware overflow rule: items
// are packed onto the current line with itemSep between them and continue on a
// new line only when the next item would overflow MaxColumns. breakConnector is
// appended to the line just before a wrap newline (e.g. "" to drop a comma, or
// " |" to keep a union bar). Continuation lines are written at the current
// ctx.Depth, so a caller wanting an extra indent level bumps Depth before
// calling. The caller positions the cursor for item 0 (this writes no leading
// newline).
func (ctx *CodeWriteContext) WriteGreedy(count int, itemSep, breakConnector string, do func(i int)) {
	if count == 0 {
		return
	}
	do(0)
	for i := 1; i < count; i++ {
		w, _ := ctx.measure(0, func() { do(i) })
		if ctx.Column()+len(itemSep)+w > ctx.maxColumns() {
			ctx.WriteString(breakConnector)
			ctx.WriteSecondLine()
			ctx.WritePrefix()
			do(i)
		} else {
			ctx.WriteString(itemSep)
			do(i)
		}
	}
}

func (ctx *CodeWriteContext) WriteExprs(sep string, expr ...Expr) {
	for i, e := range expr {
		if i > 0 {
			ctx.WriteString(sep)
		}
		e.WriteCode(ctx)
	}
}

func Code(n Coder, opt ...CodeOption) string {
	var buf bytes.Buffer
	CodeW(&buf, n, opt...)
	return normalizeMixedEndTags(buf.String())
}

func CodeW(w io.Writer, n Coder, opt ...CodeOption) {
	ctx := NewCodeWriteContext(NewCodeWriter(w), opt...)
	if ctx.srcFile != nil {
		if stmts, ok := n.(Stmts); ok {
			ctx.claimLeadDocs(stmts)
		}
	}
	n.WriteCode(ctx)
	// File-trailing comments (after the last statement) are not flushed by
	// WriteStmts (which only emits comments that precede a statement); emit them
	// here at the top level.
	if ctx.hasComments() {
		ctx.flushRemainingComments()
	}
}
