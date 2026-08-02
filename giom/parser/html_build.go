package parser

import (
	"strings"

	giomnode "github.com/gad-lang/gad/giom/node"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// buildHtmlNodes parses a (balanced) raw HTML region into giom Tag/Text AST
// nodes, so the region compiles to giom.Tag/giom.Text elements — like the
// pug-style tag syntax — instead of being written as raw HTML markup, and
// transpiles back to pug-style giom. Source positions of interpolation
// expressions (`{expr}`) are preserved.
func buildHtmlNodes(raw string, base source.Pos) (gnode.Stmts, []htmlSubError) {
	rewritten, blocks, errs := rewriteGiomBlocks(raw, base)
	b := &htmlBuilder{src: rewritten, base: base, blocks: blocks}
	nodes, _ := b.parseNodes(0)
	return nodes, errs
}

type htmlBuilder struct {
	src      string
	base     source.Pos
	blocks   []gnode.Stmts // inline giom blocks, in source order (see rewriteGiomBlocks)
	blockCur int           // next block to splice at a sentinel
}

func (b *htmlBuilder) pos(i int) source.Pos { return b.base + source.Pos(i) }

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// parseNodes parses sibling HTML nodes from i until a close tag (`</…>`) or the
// end of the region. It returns the giom child nodes and the index at the close
// tag's `<` (or len(src)); the caller consumes the close tag.
func (b *htmlBuilder) parseNodes(i int) (gnode.Stmts, int) {
	s := b.src
	var out gnode.Stmts
	for i < len(s) {
		if s[i] == '<' {
			if i+1 < len(s) && s[i+1] == '/' {
				return out, i // close tag — stop here
			}
			if i+1 < len(s) && s[i+1] == '>' {
				// `<>…</>` fragment: inline its children with no wrapper element.
				children, ci := b.parseNodes(i + 2)
				out = append(out, children...)
				i = b.skipCloseTag(ci)
				continue
			}
			elem, ni := b.parseElement(i)
			if elem != nil {
				out = append(out, elem)
			}
			i = ni
			continue
		}
		// Text run up to the next `<`.
		end := len(s)
		if j := strings.IndexByte(s[i:], '<'); j >= 0 {
			end = i + j
		}
		b.emitTextRun(i, end, &out)
		i = end
	}
	return out, i
}

// skipCloseTag skips a close tag (`</name>` or `</>`) at i (which points at `<`).
func (b *htmlBuilder) skipCloseTag(i int) int {
	s := b.src
	if i >= len(s) || s[i] != '<' {
		return i
	}
	if gt := strings.IndexByte(s[i:], '>'); gt >= 0 {
		return i + gt + 1
	}
	return len(s)
}

// emitTextRun appends the nodes for the text run src[start:end]. The run may
// contain sentinels marking interleaved giom blocks; each is spliced in as
// sibling statements (see rewriteGiomBlocks). The literal spans between
// sentinels become TextStmts, except whitespace-only spans (the padding a
// collapsed block leaves behind), which are dropped.
func (b *htmlBuilder) emitTextRun(start, end int, out *gnode.Stmts) {
	s := b.src
	if strings.IndexByte(s[start:end], sentinel) < 0 {
		// No interleaved block: preserve the normal text/whitespace behavior.
		if txt := b.textNode(start, end); txt != nil {
			*out = append(*out, txt)
		}
		return
	}
	seg := start
	for i := start; i < end; i++ {
		if s[i] != sentinel {
			continue
		}
		if strings.TrimSpace(s[seg:i]) != "" {
			if txt := b.textNode(seg, i); txt != nil {
				*out = append(*out, txt)
			}
		}
		if b.blockCur < len(b.blocks) {
			*out = append(*out, b.blocks[b.blockCur]...)
			b.blockCur++
		}
		seg = i + 1
	}
	if strings.TrimSpace(s[seg:end]) != "" {
		if txt := b.textNode(seg, end); txt != nil {
			*out = append(*out, txt)
		}
	}
}

// parseElement parses `<name attrs>children</name>` or a self-closing element,
// returning the giom TagStmt and the index just past its close.
func (b *htmlBuilder) parseElement(i int) (gnode.Stmt, int) {
	s := b.src
	tagEnd, selfClose, name := scanOpenTagEnd(s, i)
	if tagEnd < 0 {
		return nil, len(s)
	}

	attrStart := i + 1 + len(name)
	attrEnd := tagEnd - 1 // the '>'
	if selfClose {
		k := attrEnd - 1
		for k > attrStart && isSpace(s[k]) {
			k--
		}
		attrEnd = k // the '/'
	}
	attrs := b.parseAttrs(attrStart, attrEnd)

	tag := &giomnode.TagStmt{
		NodePos:     b.pos(i),
		NodeEnd:     b.pos(tagEnd),
		Name:        name,
		Attributes:  attrs,
		SelfClosing: selfClose,
	}
	if selfClose {
		return tag, tagEnd
	}
	children, ci := b.parseNodes(tagEnd)
	tag.Body = children
	tag.NodeEnd = b.pos(ci)
	return tag, b.skipCloseTag(ci)
}

// textNode builds a giom TextStmt from the text run src[start:end]: literal
// parts are whitespace-collapsed MixedText segments, and each `{expr}` becomes a
// MixedValue segment (its expression keeps its source position). Returns nil for
// an empty run.
func (b *htmlBuilder) textNode(start, end int) gnode.Stmt {
	s := b.src
	var stmts gnode.Stmts
	var lit strings.Builder
	litStart := start
	flushLit := func() {
		if lit.Len() == 0 {
			return
		}
		if collapsed := collapseWS(lit.String()); collapsed != "" {
			stmts = append(stmts, gnode.SMixedText(b.pos(litStart), collapsed))
		}
		lit.Reset()
	}
	i := start
	for i < end {
		if s[i] == '{' {
			e := skipBraces(s, i)
			flushLit()
			expr := parseExprStr(s[i+1:e-1], b.pos(i+1))
			stmts = append(stmts, gnode.SMixedValue(
				gnode.Lit("{", b.pos(i)), gnode.Lit("}", b.pos(e-1)), expr))
			i = e
			litStart = i
			continue
		}
		if lit.Len() == 0 {
			litStart = i
		}
		lit.WriteByte(s[i])
		i++
	}
	flushLit()
	if len(stmts) == 0 {
		return nil
	}
	return &giomnode.TextStmt{NodePos: b.pos(start), NodeEnd: b.pos(end), Stmts: stmts}
}

// parseAttrs parses the attribute list in src[start:end] into giom TagAttributes.
func (b *htmlBuilder) parseAttrs(start, end int) []*giomnode.TagAttribute {
	s := b.src
	var attrs []*giomnode.TagAttribute
	i := start
	for i < end {
		if isSpace(s[i]) {
			i++
			continue
		}
		nameParts, nameLit, ni := b.attrName(i, end)
		i = ni
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
		attrs = append(attrs, b.makeAttr(nameParts, nameLit, valExpr, valLit, hasVal, valIsLit))
	}
	return attrs
}

// makeAttr builds one TagAttribute. A fully-literal name yields a normal
// attribute (flag, literal value, or interpolated value); an interpolated name
// (`data-{key}`) yields a computed attribute group (`**{[name]: value}`) so it
// still lowers onto the giom.Tag call.
func (b *htmlBuilder) makeAttr(nameParts []gnode.Expr, nameLit string, valExpr gnode.Expr, valLit string, hasVal, valIsLit bool) *giomnode.TagAttribute {
	if len(nameParts) > 0 { // interpolated name -> `**{[name]: value}` spread
		nameExpr := concatExprs(nameParts)
		v := valExpr
		switch {
		case !hasVal:
			v = gnode.EIdent("true", nameExpr.Pos())
		case valIsLit:
			v = gnode.Str(unquoteAttr(valLit), nameExpr.Pos())
		}
		return &giomnode.TagAttribute{Spread: computedAttrDict(nameExpr, v)}
	}

	attr := &giomnode.TagAttribute{Name: nameLit}
	if !hasVal {
		attr.IsFlag = true
		return attr
	}
	if valIsLit {
		// A plain string value: stored as a StrLit whose giom-source form is
		// already quoted (`[href="/x"]`), so IsRaw (which would re-quote) is left
		// unset.
		attr.Value = gnode.Str(unquoteAttr(valLit), b.pos(0))
		return attr
	}
	attr.Value = valExpr // interpolated value expression
	return attr
}

// computedAttrDict builds a `{[name]: value}` dict literal for a computed
// (interpolated-name) attribute. The key is wrapped in parentheses so it is
// evaluated at runtime (a bare identifier key would otherwise be a static name).
func computedAttrDict(nameExpr, valExpr gnode.Expr) gnode.Expr {
	key := gnode.EParen(nameExpr, nameExpr.Pos(), nameExpr.End())
	return &gnode.DictExpr{
		Elements: []*gnode.DictElementLit{{Key: key, Value: valExpr}},
	}
}

// --- attribute-part parsing (position-preserving) ---

// attrName parses an attribute name of literal characters and `{expr}`
// interpolations. It returns the interpolation parts (nil when fully literal),
// the literal name (when there is no interpolation) and the new index.
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
		if isSpace(c) || c == '=' || c == '>' || c == '/' {
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
// returns the value expression (for `{expr}`), the raw literal text (quoted or
// bareword), whether it is literal, and the new index.
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
		for j < end && !isSpace(s[j]) && s[j] != '>' && s[j] != '/' {
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
		if isSpace(s[i]) {
			space = true
			continue
		}
		if space && b.Len() > 0 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteByte(s[i])
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
