package parser

import (
	"strings"

	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// buildHtmlStmts turns a raw HTML region into the Gad statements that render it:
// literal markup becomes `write(raw "…")`, a `{expr}` text interpolation becomes
// `giom.write(expr)` (HTML-escaped), and an interpolated attribute becomes
// `write(giom.attr(name, value))` (auto-quoted and escaped). Runs of whitespace
// in text content are collapsed to a single space. base is the absolute source
// position of raw[0], so interpolation expressions keep their source positions.
func buildHtmlStmts(raw string, base source.Pos) gnode.Stmts {
	b := &htmlBuilder{src: raw, base: base}
	b.run()
	b.flush()
	return b.out
}

type htmlBuilder struct {
	src      string
	base     source.Pos
	out      gnode.Stmts
	lit      strings.Builder
	litPos   source.Pos
	litValid bool
}

func (b *htmlBuilder) pos(i int) source.Pos { return b.base + source.Pos(i) }

// emitLit appends verbatim literal markup to the pending `write(raw …)` buffer.
func (b *htmlBuilder) emitLit(text string, i int) {
	if text == "" {
		return
	}
	if !b.litValid {
		b.litPos = b.pos(i)
		b.litValid = true
	}
	b.lit.WriteString(text)
}

func (b *htmlBuilder) flush() {
	if b.lit.Len() == 0 {
		b.litValid = false
		return
	}
	b.out = append(b.out, writeRawStmt(b.lit.String(), b.litPos))
	b.lit.Reset()
	b.litValid = false
}

func (b *htmlBuilder) run() {
	s := b.src
	i := 0
	for i < len(s) {
		if s[i] == '<' {
			i = b.tag(i)
			continue
		}
		i = b.text(i)
	}
}

// text emits a text run (until the next `<`), collapsing whitespace and lowering
// `{expr}` interpolations to escaped writes.
func (b *htmlBuilder) text(start int) int {
	s := b.src
	i := start
	var run strings.Builder
	runStart := i
	flushRun := func() {
		if run.Len() > 0 {
			b.emitLit(collapseWS(run.String()), runStart)
			run.Reset()
		}
	}
	for i < len(s) && s[i] != '<' {
		if s[i] == '{' {
			end := skipBraces(s, i)
			flushRun()
			b.flush()
			expr := parseExprStr(s[i+1:end-1], b.pos(i+1))
			b.out = append(b.out, writeEscStmt(expr))
			i = end
			runStart = i
			continue
		}
		if run.Len() == 0 {
			runStart = i
		}
		run.WriteByte(s[i])
		i++
	}
	flushRun()
	return i
}

// tag emits a single tag beginning at s[i] (`<`): an open/self-closing tag, a
// close tag, or a `<>` / `</>` fragment (which produce no markup).
func (b *htmlBuilder) tag(start int) int {
	s := b.src
	i := start
	if i+1 < len(s) && s[i+1] == '/' {
		gt := strings.IndexByte(s[i:], '>')
		if gt < 0 {
			return len(s)
		}
		gt += i
		name := strings.TrimSpace(s[i+2 : gt])
		if name != "" {
			b.emitLit("</"+name+">", i)
		}
		return gt + 1
	}
	if i+1 < len(s) && s[i+1] == '>' {
		return i + 2 // fragment open: no markup
	}

	tagEnd, selfClose, name := scanOpenTagEnd(s, i)
	if tagEnd < 0 {
		return len(s)
	}
	b.emitLit("<"+name, i)

	// Attribute region: after the tag name, up to the closing `/` (self-close)
	// or `>`.
	attrEnd := tagEnd - 1 // the '>'
	if selfClose {
		k := attrEnd - 1
		for k > i && (s[k] == ' ' || s[k] == '\t' || s[k] == '\n' || s[k] == '\r') {
			k--
		}
		attrEnd = k // the '/'
	}
	b.attributes(i+1+len(name), attrEnd)

	if selfClose {
		b.emitLit(" />", attrEnd)
	} else {
		b.emitLit(">", attrEnd)
	}
	return tagEnd
}

// attributes parses the attribute list in s[start:end] and emits each one:
// fully-literal attributes stay in the `write(raw …)` buffer, while any
// attribute with an interpolated name or value becomes `write(giom$attr(name,
// value))`.
func (b *htmlBuilder) attributes(start, end int) {
	s := b.src
	i := start
	for i < end {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			i++
			continue
		}
		// Attribute name: literal characters and `{expr}` interpolations.
		nameParts, nameLit, ni := b.attrName(i, end)
		i = ni
		// Optional `= value`.
		var (
			valExpr  gnode.Expr
			valLit   string
			hasVal   bool
			valIsLit bool
		)
		if i < end && s[i] == '=' {
			i++
			valExpr, valLit, valIsLit, i = b.attrValue(i, end)
			hasVal = true
		}

		nameInterp := len(nameParts) > 0
		if !nameInterp && (!hasVal || valIsLit) {
			// Fully literal attribute: keep it verbatim.
			b.emitLit(" "+nameLit, start)
			if hasVal {
				b.emitLit("="+valLit, start)
			}
			continue
		}
		// Interpolated name and/or value -> giom$attr(name, value).
		var nameExpr gnode.Expr
		if nameInterp {
			nameExpr = concatExprs(nameParts)
		} else {
			nameExpr = gnode.Str(nameLit, b.pos(start))
		}
		if !hasVal {
			valExpr = gnode.EIdent("true", b.pos(start))
		} else if valIsLit {
			valExpr = gnode.Str(unquoteAttr(valLit), b.pos(start))
		}
		b.emitLit(" ", start)
		b.flush()
		b.out = append(b.out, writeAttrStmt(nameExpr, valExpr))
	}
}

// attrName parses an attribute name of literal characters and `{expr}`
// interpolations. It returns the interpolation parts (nil when fully literal,
// otherwise the ordered literal/expression fragments), the literal name (when
// there is no interpolation) and the new index.
func (b *htmlBuilder) attrName(start, end int) (parts []gnode.Expr, lit string, next int) {
	s := b.src
	i := start
	var (
		buf     strings.Builder
		interp  bool
		segment strings.Builder
	)
	flushSeg := func() {
		if segment.Len() > 0 {
			parts = append(parts, gnode.Str(segment.String(), b.pos(start)))
			segment.Reset()
		}
	}
	for i < end {
		c := s[i]
		if c == '{' {
			interp = true
			e := skipBraces(s, i)
			flushSeg()
			parts = append(parts, parseExprStr(s[i+1:e-1], b.pos(i+1)))
			i = e
			continue
		}
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '=' || c == '>' || c == '/' {
			break
		}
		buf.WriteByte(c)
		segment.WriteByte(c)
		i++
	}
	if interp {
		flushSeg()
		return parts, "", i
	}
	return nil, buf.String(), i
}

// attrValue parses an attribute value: `"…"`, `'…'`, `{expr}`, or a bareword. It
// returns the value expression (for `{expr}`), the raw literal text (for quoted
// or bareword values), whether it is literal, and the new index.
func (b *htmlBuilder) attrValue(start, end int) (expr gnode.Expr, lit string, isLit bool, next int) {
	s := b.src
	i := start
	if i >= end {
		return nil, "", true, i
	}
	switch s[i] {
	case '"', '\'':
		e := skipString(s, i)
		return nil, s[i:e], true, e
	case '{':
		e := skipBraces(s, i)
		return parseExprStr(s[i+1:e-1], b.pos(i+1)), "", false, e
	default:
		j := i
		for j < end && s[j] != ' ' && s[j] != '\t' && s[j] != '\n' && s[j] != '\r' && s[j] != '>' && s[j] != '/' {
			j++
		}
		return nil, s[i:j], true, j
	}
}

// --- helpers ---

// collapseWS replaces every run of ASCII whitespace with a single space.
func collapseWS(s string) string {
	var b strings.Builder
	space := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteByte(c)
	}
	if space {
		b.WriteByte(' ')
	}
	return b.String()
}

// unquoteAttr strips surrounding quotes from a literal attribute value.
func unquoteAttr(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// concatExprs folds parts into a `+` concatenation, so an interpolated name like
// `data-{key}` becomes `"data-" + key`.
func concatExprs(parts []gnode.Expr) gnode.Expr {
	if len(parts) == 0 {
		return gnode.Str("", 0)
	}
	expr := parts[0]
	for _, p := range parts[1:] {
		expr = gnode.EBinary(expr, p, token.Add, expr.Pos())
	}
	return expr
}

func writeRawStmt(lit string, pos source.Pos) gnode.Stmt {
	call := gnode.ECall(gnode.EIdent("write", pos), pos, pos)
	call.Args.Values = append(call.Args.Values, gnode.EToRaw(pos, gnode.Str(lit, pos)))
	return gnode.SExpr(call)
}

func writeEscStmt(expr gnode.Expr) gnode.Stmt {
	call := gnode.ECall(gnode.ESelector(gnode.EIdent("giom", expr.Pos()), gnode.Str("write", 0)), expr.Pos(), expr.End())
	call.Args.Values = append(call.Args.Values, expr)
	return gnode.SExpr(call)
}

func writeAttrStmt(nameExpr, valExpr gnode.Expr) gnode.Stmt {
	attr := gnode.ECall(gnode.ESelector(gnode.EIdent("giom", nameExpr.Pos()), gnode.Str("attr", 0)), nameExpr.Pos(), valExpr.End())
	attr.Args.Values = append(attr.Args.Values, nameExpr, valExpr)
	call := gnode.ECall(gnode.EIdent("write", nameExpr.Pos()), nameExpr.Pos(), valExpr.End())
	call.Args.Values = append(call.Args.Values, attr)
	return gnode.SExpr(call)
}
