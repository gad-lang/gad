package node

import (
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// The inline checks of issue #22, statements of their own, written where a
// statement starts: what a test, an example or a doc fence verifies as it runs
// (the compile option Checks); elsewhere they compile to nothing.

// ValueCheckStmt is `= EXPR`: the value of the statement before it, compared as
// `str(…)` with `str(EXPR)`.
type ValueCheckStmt struct {
	EqPos source.Pos
	Expr  Expr
}

func (s *ValueCheckStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *ValueCheckStmt) Pos() source.Pos { return s.EqPos }

// End returns the position of first character immediately after the node.
func (s *ValueCheckStmt) End() source.Pos { return s.Expr.End() }

func (s *ValueCheckStmt) String() string { return "= " + s.Expr.String() }

func (s *ValueCheckStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("= ")
	s.Expr.WriteCode(ctx)
}

// OutputCheckStmt is `<<< text` (one line, `\n`, `\t`, `\\` escaped) or
// `<<<⏎ lines ⏎>>>`: what was written to stdout since the previous check
// (or the start) must be Text.
type OutputCheckStmt struct {
	TokenPos source.Pos
	// Text is the expected output, unescaped.
	Text string
	// Multi is the multi-line form, as written.
	Multi bool
	// EndPos is the position right after the check.
	EndPos source.Pos
	// Target, when set, is what is checked instead of stdout: what was
	// rendered into it since the previous check (gadx's `<<<`, its render tree
	// `$el`: a value implementing gad.RenderedChecker).
	Target Expr
}

func (s *OutputCheckStmt) StmtNode() {}

// Pos returns the position of first character belonging to the node.
func (s *OutputCheckStmt) Pos() source.Pos { return s.TokenPos }

// End returns the position of first character immediately after the node.
func (s *OutputCheckStmt) End() source.Pos { return s.EndPos }

func (s *OutputCheckStmt) String() string { return "<<< " + EscapeOutputCheck(s.Text) }

// WriteCode writes the multi-line form when it was written so and the code
// is laid out in lines, else the one-line form — always ending the line, its
// text running to the end of it.
func (s *OutputCheckStmt) WriteCode(ctx *CodeWriteContext) {
	if s.Multi && ctx.HasPrefix() {
		// the lines under `<<<`, at its indentation
		ctx.WriteString("<<<")
		for _, line := range strings.Split(s.Text, "\n") {
			ctx.WriteString("\n")
			if line != "" {
				ctx.WriteString(ctx.CurrentPrefix() + line)
			}
		}
		ctx.WriteString("\n" + ctx.CurrentPrefix() + ">>>")
		return
	}
	ctx.WriteString("<<< " + EscapeOutputCheck(s.Text))
	if !ctx.HasPrefix() {
		// inline, the next separator is `; `: the text ends at the line's end
		ctx.WriteString("\n")
	}
}

// isCheckStmt reports whether s is an inline check.
func isCheckStmt(s Stmt) bool {
	switch s.(type) {
	case *ValueCheckStmt, *OutputCheckStmt:
		return true
	}
	return false
}

// EscapeOutputCheck is text in the one-line form: `\`, line breaks and tabs
// escaped.
func EscapeOutputCheck(text string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\t", `\t`).Replace(text)
}

// UnescapeOutputCheck is the text of the one-line form: `\n`, `\t`, `\\`
// unescaped (any other `\x` kept as written).
func UnescapeOutputCheck(text string) string {
	if !strings.ContainsRune(text, '\\') {
		return text
	}
	var b strings.Builder
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c == '\\' && i+1 < len(text) {
			switch text[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case 't':
				b.WriteByte('\t')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			}
		}
		b.WriteByte(c)
	}
	return b.String()
}
