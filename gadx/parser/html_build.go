package parser

import (
	"strings"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// buildHTMLNodes parses a (balanced) raw HTML region into gadx Tag/Text AST
// nodes, so the region compiles to gadx.Tag/gadx.Text elements — like the
// pug-style tag syntax — instead of being written as raw HTML markup, and
// transpiles back to pug-style gadx. Source positions of interpolation
// expressions (`{expr}`) are preserved.
func buildHTMLNodes(raw string, base source.Pos) (gnode.Stmts, []htmlSubError) {
	rewritten, blocks, errs := rewriteGadxBlocks(raw, base)
	b := &htmlBuilder{src: rewritten, base: base, blocks: blocks}
	nodes, _ := b.parseNodes(0)
	return nodes, errs
}

type htmlBuilder struct {
	src      string
	base     source.Pos
	blocks   []gnode.Stmts // inline gadx blocks, in source order (see rewriteGadxBlocks)
	blockCur int           // next block to splice at a sentinel
}

func (b *htmlBuilder) pos(i int) source.Pos { return b.base + source.Pos(i) }

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// parseNodes parses sibling HTML nodes from i until a close tag (`</…>`) or the
// end of the region. It returns the gadx child nodes and the index at the close
// tag's `<` (or len(src)); the caller consumes the close tag.
func (b *htmlBuilder) parseNodes(i int) (gnode.Stmts, int) {
	s := b.src
	var out gnode.Stmts
	for i < len(s) {
		if s[i] == '<' {
			if i+1 < len(s) && s[i+1] == '/' {
				return out, i // close tag — stop here
			}
			if i+1 < len(s) && s[i+1] == '!' {
				// A comment, doctype or CDATA section. It is not an element, and
				// letting parseElement read it as one made a `/` in the comment
				// look like a self-closing tag and the scan stop advancing.
				next, ok := skipMarkupDeclaration(s, i)
				if !ok || next <= i {
					return out, len(s)
				}
				// A comment is content — it is how a template hands a note to
				// whatever reads the rendered page — so it is kept. A doctype or
				// a CDATA section carries nothing to render and is dropped; the
				// doctype has the `!!! 5` statement of its own.
				if strings.HasPrefix(s[i:], "<!--") && next-i >= 7 {
					out = append(out, &gadxnode.HTMLCommentStmt{
						NodePos: b.pos(i),
						NodeEnd: b.pos(next),
						Text:    s[i+4 : next-3],
					})
				}
				i = next
				continue
			}
			if i+1 < len(s) && s[i+1] == '>' {
				// `<>…</>` fragment: a wrapper-less node lowering to gadx.Elements()
				// (its children are spliced into the enclosing parent on append).
				open := i
				children, ci := b.parseNodes(i + 2)
				end := b.skipCloseTag(ci)
				out = append(out, &gadxnode.TagStmt{
					NodePos:  b.pos(open),
					NodeEnd:  b.pos(end),
					Fragment: true,
					Body:     children,
				})
				i = end
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
// contain sentinels marking interleaved gadx blocks; each is spliced in as
// sibling statements (see rewriteGadxBlocks). The literal spans between
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
// returning the gadx TagStmt and the index just past its close.
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

	tag := &gadxnode.TagStmt{
		NodePos:     b.pos(i),
		NodeEnd:     b.pos(tagEnd),
		Name:        name,
		Attributes:  hoistID(attrs),
		SelfClosing: selfClose,
	}
	if selfClose {
		return tag, tagEnd
	}
	if gadxnode.IsSelfClosing(name) {
		// A void element has no close tag, so it has no children either. Reading
		// on for them made a `<meta>` adopt its following siblings and then eat
		// the close tag of whatever element actually enclosed it, which cut the
		// rest of the document off the tree.
		tag.SelfClosing = true
		return tag, tagEnd
	}
	if lower := strings.ToLower(name); htmlRawTextElements[lower] {
		contentEnd, after := skipRawTextElement(s, tagEnd, lower)
		if contentEnd < 0 {
			contentEnd, after = len(s), len(s)
		}
		if txt := b.rawTextNode(tagEnd, contentEnd); txt != nil {
			tag.Body = gnode.Stmts{txt}
		}
		tag.NodeEnd = b.pos(contentEnd)
		return tag, after
	}
	children, ci := b.parseNodes(tagEnd)
	tag.Body = children
	tag.NodeEnd = b.pos(ci)
	return tag, b.skipCloseTag(ci)
}

// textNode builds a gadx TextStmt from the text run src[start:end]: literal
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
		// Backslash escapes a literal brace or backslash (`\{`, `\}`, `\\`), so
		// content braces — e.g. Markdown lowered to HTML — are not mistaken for an
		// interpolation. Any other `\x` stays literal.
		if s[i] == '\\' && i+1 < end && (s[i+1] == '{' || s[i+1] == '}' || s[i+1] == '\\') {
			if lit.Len() == 0 {
				litStart = i
			}
			lit.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '{' {
			e := skipBraces(s, i)
			flushLit()
			// `{= expr }` emits its value; a bare `{ expr }` is a control statement
			// that runs but emits nothing (same rule as pug-style tag bodies).
			if i+1 < e-1 && s[i+1] == '=' {
				expr := parseExprStr(s[i+2:e-1], b.pos(i+2))
				stmts = append(stmts, gnode.SMixedValue(
					gnode.Lit("{=", b.pos(i)), gnode.Lit("}", b.pos(e-1)), expr))
			} else {
				expr := parseExprStr(s[i+1:e-1], b.pos(i+1))
				stmts = append(stmts, gnode.SExpr(expr))
			}
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
	return &gadxnode.TextStmt{NodePos: b.pos(start), NodeEnd: b.pos(end), Stmts: stmts}
}

// rawTextNode builds the content of a raw-text element — a script or a
// stylesheet — from src[start:end]. That content is code, not markup: it is
// written verbatim, so braces are literal (a CSS rule is not an interpolation),
// whitespace survives, and nothing is HTML-escaped, which would corrupt a
// string in either language.
//
// `#{= expr }#` writes a value and `#{ expr }#` is a control statement that
// runs and writes nothing — the same pair as `{= … }` and `{ … }` in ordinary
// text. The closing `}#` is what makes either unambiguous: a lone `}` belongs
// to the CSS or the JS. A written value goes out verbatim like the rest, so
// whatever it carries must already be valid in the language it lands in.
func (b *htmlBuilder) rawTextNode(start, end int) gnode.Stmt {
	stmts := rawTextStmts(b.src[start:end], b.pos(start))
	if len(stmts) == 0 {
		return nil
	}
	return &gadxnode.TextStmt{NodePos: b.pos(start), NodeEnd: b.pos(end), Stmts: stmts}
}

// rawTextStmts parses raw text — the content of a script, a stylesheet or a
// `@raw_text` block — into the statements of a text run. Everything is literal
// except `#{= expr }#`, which writes a value, and `#{ expr }#`, which runs and
// writes nothing. The literal spans are carried as raw strings so that rendering
// writes them verbatim instead of HTML-escaping code.
//
// base is the source position of s[0], so every expression maps back onto the
// original file.
func rawTextStmts(s string, base source.Pos) gnode.Stmts {
	pos := func(i int) source.Pos { return base + source.Pos(i) }

	var stmts gnode.Stmts
	litStart := 0
	flushLit := func(to int) {
		if to <= litStart {
			return
		}
		stmts = append(stmts, gnode.SMixedValue(
			gnode.Lit("", pos(litStart)), gnode.Lit("", pos(to)),
			gnode.RawStr(s[litStart:to], pos(litStart))))
	}
	for i := 0; i < len(s); {
		if s[i] != '#' || i+1 >= len(s) || s[i+1] != '{' {
			i++
			continue
		}
		close := strings.Index(s[i+2:], "}#")
		if close < 0 {
			break
		}
		close += i + 2
		flushLit(i)
		if i+2 < close && s[i+2] == '=' {
			expr := parseExprStr(s[i+3:close], pos(i+3))
			stmts = append(stmts, gnode.SMixedValue(
				gnode.Lit("#{=", pos(i)), gnode.Lit("}#", pos(close)), expr))
		} else {
			stmts = append(stmts, gnode.SExpr(parseExprStr(s[i+2:close], pos(i+2))))
		}
		i = close + 2
		litStart = i
	}
	flushLit(len(s))
	return stmts
}

// parseAttrs parses the attribute list in src[start:end] into gadx TagAttributes.
func (b *htmlBuilder) parseAttrs(start, end int) []*gadxnode.TagAttribute {
	s := b.src
	var attrs []*gadxnode.TagAttribute
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
// still lowers onto the gadx.Tag call.
func (b *htmlBuilder) makeAttr(nameParts []gnode.Expr, nameLit string, valExpr gnode.Expr, valLit string, hasVal, valIsLit bool) *gadxnode.TagAttribute {
	if len(nameParts) > 0 { // interpolated name -> `**{[name]: value}` spread
		nameExpr := concatExprs(nameParts)
		v := valExpr
		switch {
		case !hasVal:
			v = gnode.EIdent("true", nameExpr.Pos())
		case valIsLit:
			v = gnode.Str(unquoteAttr(valLit), nameExpr.Pos())
		}
		return &gadxnode.TagAttribute{Spread: computedAttrDict(nameExpr, v)}
	}

	attr := &gadxnode.TagAttribute{Name: nameLit}
	if !hasVal {
		attr.IsFlag = true
		return attr
	}
	if valIsLit {
		if unquoteAttr(valLit) == "" {
			// `value=""` is present and empty, which a plain "" cannot express:
			// falsy attribute values are dropped so conditional attributes work.
			// gadx.EMPTY carries the distinction through to the renderer.
			attr.Value = emptyAttrValue(b.pos(0))
			return attr
		}
		// A plain string value: stored as a StrLit whose gadx-source form is
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
// collapseWS collapses runs of whitespace to a single space, keeping a single
// leading and trailing space when the text has content — HTML inline whitespace
// between elements/interpolations is significant (e.g. `{x} y` and
// `<strong>x</strong> y` must keep the space). A whitespace-only run collapses
// to a single space (the padding between block elements).
func collapseWS(s string) string {
	var b strings.Builder
	space := false
	hasContent := false
	for i := 0; i < len(s); i++ {
		if isSpace(s[i]) {
			space = true
			continue
		}
		if space {
			b.WriteByte(' ')
		}
		space = false
		hasContent = true
		b.WriteByte(s[i])
	}
	if space && (hasContent || b.Len() == 0) {
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

// emptyAttrValue is the expression for an attribute that is present and empty:
// the gadx.EMPTY constant, which the attribute renderer writes as `name=""`.
func emptyAttrValue(pos source.Pos) gnode.Expr {
	return gnode.ESelector(gnode.EIdent("gadx", pos), gnode.Str("EMPTY", pos))
}
