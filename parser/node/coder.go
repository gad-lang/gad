package node

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
)

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
	Stack     []ast.Node
	Depth     int
	Prefix    string
	Flags     CodeWriteContextFlag
	Transpile *TranspileOptions
	CodeWriter
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

func CodeTranspile(v *TranspileOptions) CodeOption {
	return func(ctx *CodeWriteContext) {
		ctx.Transpile = v
	}
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

		// Leading separator. A `%}` terminator always hugs the preceding code
		// with a single space so the whole `{% … %}` tag stays on one line.
		if _, isEnd := s.(*CodeEndStmt); isEnd && !transpiling {
			ctx.WriteString(" ")
		} else if i > 0 {
			switch sep {
			case sepSpace:
				ctx.WriteString(" ")
			case sepNewline:
				ctx.WriteSemi()
			}
		}
		s.WriteCode(ctx)
		i++

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
	n.WriteCode(NewCodeWriteContext(NewCodeWriter(w), opt...))
}
