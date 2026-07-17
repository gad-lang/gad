package parser

import (
	"bytes"
	"fmt"
	"strings"

	gadparser "github.com/gad-lang/gad/parser"
	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"

	giomnode "github.com/gad-lang/gad/giom/node"
	giomtoken "github.com/gad-lang/gad/giom/token"
)

type bailout struct{}

// Parser parses giom template source into AST nodes.
type Parser struct {
	file      *source.File
	Errors    gadparser.ErrorList
	scanner   *scanner
	Token     gadparser.PToken
	PrevToken gadparser.PToken
	filename  string
	comps     []*giomnode.CompDecl
	compStack []*giomnode.CompDecl
}

// NewParser creates a new Parser for the given source file.
func NewParser(file *source.File) *Parser {
	p := &Parser{
		file:    file,
		scanner: newScanner(file, bytes.NewReader(file.Data.Bytes())),
	}
	p.Next()
	return p
}

// Next advances the scanner and stores the current and previous tokens.
func (p *Parser) Next() {
	p.PrevToken = p.Token
	p.Token = p.scanner.Scan()
}

// expect checks the current token against the given kinds. If exactly one kind
// is provided and the token matches, it advances to the next token. If multiple
// kinds are provided, it checks without advancing. If no kind matches, it panics.
func (p *Parser) expect(kinds ...token.Token) {
	for _, kind := range kinds {
		if p.Token.Token == kind {
			if len(kinds) == 1 {
				p.Next()
			}
			return
		}
	}
	p.Error(p.Token.Pos, fmt.Sprintf("expected %v, got %s (%s)", kinds, giomtoken.String(p.Token.Token), p.Token.Literal))
}

func (p *Parser) Error(pos source.Pos, msg string) {
	filePos := source.MustFilePosition(p.file, pos)
	n := len(p.Errors)
	if n > 0 && p.Errors[n-1].Pos.Line == filePos.Line {
		// discard errors reported on the same line
		return
	}
	if n > 10 {
		// too many errors; terminate early
		panic(bailout{})
	}
	p.Errors.Add(filePos, msg)
}

// currentComp returns the component at the top of the comp stack, or nil.
func (p *Parser) currentComp() *giomnode.CompDecl {
	if len(p.compStack) == 0 {
		return nil
	}
	return p.compStack[len(p.compStack)-1]
}

// ParseFile parses the entire giom source file into a File AST node.
func (p *Parser) ParseFile() (_ *giomnode.File, err error) {
	defer func() {
		if e := recover(); e != nil {
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		p.Errors.Sort()
		err = p.Errors.Err()
	}()

	file := &giomnode.File{
		InputFile: p.scanner.SourceFile(),
	}

	for p.Token.Token != giomtoken.EOF && p.Token.Token != token.Illegal {
		stmt := p.parseStmt()

		if p.Errors.Len() > 0 {
			return nil, p.Errors.Err()
		}

		if stmt != nil {
			file.Stmts = append(file.Stmts, stmt)
		}
	}

	file.Comps = p.comps
	return file, nil
}

// =============================================================================
// Statement parsing
// =============================================================================

// parseStmt dispatches to the appropriate parse function based on the current token.
func (p *Parser) parseStmt() gnode.Stmt {
	switch p.Token.Token {
	case giomtoken.Doctype:
		return p.parseDoctype()
	case giomtoken.Comment:
		return p.parseComment()
	case giomtoken.Text:
		return p.parseText()
	case giomtoken.Html:
		return p.parseHtml()
	case giomtoken.Tag:
		return p.parseTag()
	case giomtoken.Id, giomtoken.ClassName, giomtoken.Attribute:
		return p.parseTag()
	case giomtoken.If:
		return p.parseIf()
	case giomtoken.ElseIf, giomtoken.Else:
		p.Error(p.Token.Pos, fmt.Sprintf("unexpected %s without matching @if", giomtoken.String(p.Token.Token)))
		p.Next()
		return nil
	case giomtoken.For:
		return p.parseFor()
	case giomtoken.Assignment:
		return p.parseAssignment()
	case giomtoken.Code:
		return p.parseCode()
	case giomtoken.ImportModule:
		return p.parseImportModule()
	case giomtoken.Global:
		return p.parseGlobal()
	case giomtoken.Var:
		return p.parseVar()
	case giomtoken.Const:
		return p.parseConst()
	case giomtoken.Enum:
		return p.parseEnum()
	case giomtoken.Func:
		return p.parseFunc()
	case giomtoken.Comp:
		return p.parseComp()
	case giomtoken.Slot:
		return p.parseSlot()
	case giomtoken.SlotPass:
		return p.parseSlotPass()
	case giomtoken.Wrap:
		return p.parseWrap()
	case giomtoken.CompCall:
		return p.parseCompCall()
	case giomtoken.Match:
		return p.parseMatch()
	case giomtoken.Case:
		p.Error(p.Token.Pos, fmt.Sprintf("unexpected %s without matching @match", giomtoken.String(p.Token.Token)))
		p.Next()
		return nil
	case giomtoken.Export:
		return p.parseExport()
	case giomtoken.Blank:
		p.Next()
		return nil
	case giomtoken.Outdent, giomtoken.EOF:
		return nil
	default:
		p.Error(p.Token.Pos, fmt.Sprintf("unexpected token: %s (%s)", giomtoken.String(p.Token.Token), p.Token.Literal))
		p.Next()
		return nil
	}
}

// parseBlock parses an indented block of statements. It expects the current
// token to be Indent, consumes statements until Outdent, and returns them.
func (p *Parser) parseBlock(parent gnode.Stmt) gnode.Stmts {
	p.expect(giomtoken.Indent)

	var stmts gnode.Stmts
	for p.Token.Token != giomtoken.Outdent && p.Token.Token != giomtoken.EOF {
		stmt := p.parseStmt()
		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	p.expect(giomtoken.Outdent)
	return stmts
}

// =============================================================================
// Parse functions for specific constructs
// =============================================================================

func (p *Parser) parseDoctype() *giomnode.DoctypeStmt {
	tok := p.Token
	p.expect(giomtoken.Doctype)

	value := stringData(tok, "value", "html")
	d := &giomnode.DoctypeStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Value:   value,
	}
	return d
}

func (p *Parser) parseComment() *giomnode.CommentStmt {
	tok := p.Token
	p.expect(giomtoken.Comment)

	mode := stringData(tok, "mode", "embed")
	text := stringData(tok, "value", "")

	c := &giomnode.CommentStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Text:    text,
		Silent:  mode == "silent",
	}

	if p.Token.Token == giomtoken.Indent {
		c.Body = p.parseBlock(c)
		if len(c.Body) > 0 {
			c.NodeEnd = c.Body[len(c.Body)-1].End()
		}
	}

	return c
}

func (p *Parser) parseText() *giomnode.TextStmt {
	tok := p.Token
	p.expect(giomtoken.Text)

	content := stringData(tok, "value", "")

	t := &giomnode.TextStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
	}

	if content != "" {
		base := noBase
		if positions, ok := tokenValuePos(tok); ok && len(positions) > 0 {
			base = positions[0]
		}
		stmts, err := parseTextGadAt(content, base)
		if err == nil {
			t.Stmts = stmts
		}
	}

	return t
}

func (p *Parser) parseHtml() *giomnode.HtmlStmt {
	tok := p.Token
	p.expect(giomtoken.Html)

	raw := stringData(tok, "value", tok.Literal)
	base := posData(tok, "htmlPos")
	if base == noBase {
		base = tok.Pos
	}
	return &giomnode.HtmlStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Stmts:   buildHtmlStmts(raw, base),
	}
}

func (p *Parser) parseTag() *giomnode.TagStmt {
	tok := p.Token
	p.expect(giomtoken.Tag)

	name := stringData(tok, "value", tok.Literal)

	tag := &giomnode.TagStmt{
		NodePos: tok.Pos,
		Name:    name,
	}

	// Consume inline attributes (id, class, and `[ … ]` attribute groups).
	for p.Token.Token == giomtoken.Id ||
		p.Token.Token == giomtoken.ClassName ||
		p.Token.Token == giomtoken.Attribute {
		if p.Token.Token == giomtoken.Attribute {
			tok := p.Token
			p.expect(giomtoken.Attribute)
			tag.Attributes = append(tag.Attributes, p.parseAttributeGroup(tok)...)
			continue
		}
		if attr := p.parseInlineAttribute(); attr != nil {
			tag.Attributes = append(tag.Attributes, attr)
		}
	}

	tag.SelfClosing = giomnode.IsSelfClosing(name)

	if p.Token.Token == giomtoken.Indent {
		tag.Body = p.parseBlock(tag)
	} else if p.Token.Token == giomtoken.Text {
		tag.Body = gnode.Stmts{p.parseText()}
	}

	if len(tag.Body) > 0 {
		tag.NodeEnd = tag.Body[len(tag.Body)-1].End()
	} else {
		tag.NodeEnd = tok.Pos + source.Pos(len(tok.Literal))
	}

	return tag
}

func (p *Parser) parseInlineAttribute() *giomnode.TagAttribute {
	tok := p.Token
	switch tok.Token {
	case giomtoken.Id:
		p.expect(giomtoken.Id)
		attr := &giomnode.TagAttribute{
			Name:  "id",
			Value: gnode.Str(stringData(tok, "value", ""), tok.Pos),
		}
		if cond := stringData(tok, "condition", ""); cond != "" {
			attr.Condition = parseExprStr(cond, tok.Pos)
		}
		return attr
	case giomtoken.ClassName:
		p.expect(giomtoken.ClassName)
		attr := &giomnode.TagAttribute{
			Name:  "class",
			Value: gnode.Str(stringData(tok, "value", ""), tok.Pos),
		}
		if cond := stringData(tok, "condition", ""); cond != "" {
			attr.Condition = parseExprStr(cond, tok.Pos)
		}
		return attr
	default:
		// Attribute groups are handled by parseAttributeGroup in parseTag.
		p.expect(giomtoken.Attribute)
		return nil
	}
}

func (p *Parser) parseIf() *giomnode.IfStmt {
	tok := p.Token
	p.expect(giomtoken.If)

	condStr := stringData(tok, "value", "")

	s := &giomnode.IfStmt{
		NodePos: tok.Pos,
		Cond:    parseExprStr(condStr, tok.Pos),
	}

	if p.Token.Token == giomtoken.Indent {
		s.Body = p.parseBlock(s)
	}

	// Handle optional else-ifs and else
	for p.Token.Token == giomtoken.ElseIf {
		elseIfTok := p.Token
		p.expect(giomtoken.ElseIf)
		clause := &giomnode.ElseIfClause{
			Cond: parseExprStr(stringData(elseIfTok, "value", ""), elseIfTok.Pos),
		}
		if p.Token.Token == giomtoken.Indent {
			clause.Body = p.parseBlock(s)
		}
		s.ElseIfs = append(s.ElseIfs, clause)
	}

	if p.Token.Token == giomtoken.Else {
		p.expect(giomtoken.Else)
		if p.Token.Token == giomtoken.Indent {
			s.Else = p.parseBlock(s)
		}
	}

	s.NodeEnd = bodyEndIf(s)
	return s
}

func (p *Parser) parseFor() *giomnode.ForStmt {
	tok := p.Token
	p.expect(giomtoken.For)

	condStr := stringData(tok, "value", "")

	s := &giomnode.ForStmt{
		NodePos: tok.Pos,
		Cond:    parseExprStr(condStr, tok.Pos),
	}

	if p.Token.Token == giomtoken.Indent {
		s.Body = p.parseBlock(s)
	}

	if p.Token.Token == giomtoken.Else {
		p.expect(giomtoken.Else)
		if p.Token.Token == giomtoken.Indent {
			s.Else = p.parseBlock(s)
		}
	}

	s.NodeEnd = bodyEndFor(s)
	return s
}

func (p *Parser) parseAssignment() *giomnode.AssignStmt {
	tok := p.Token
	p.expect(giomtoken.Assignment)

	x := stringData(tok, "x", "")
	op := stringData(tok, "op", "")
	valueStr := stringData(tok, "value", "")

	s := &giomnode.AssignStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Op:      op,
		RHS:     parseExprStr(valueStr, tok.Pos),
	}

	if x != "" {
		s.LHS = gnode.EIdent(x, tok.Pos)
	}

	return s
}

func (p *Parser) parseCode() *giomnode.CodeStmt {
	tok := p.Token
	p.expect(giomtoken.Code)

	s := &giomnode.CodeStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
	}

	if values, ok := tok.GetOk("values"); ok {
		switch v := values.(type) {
		case []string:
			positions, _ := tokenValuePos(tok)
			// Multi-line code (~~) — join lines and parse as full GAD source.
			// Lines are verbatim (indentation preserved) and joined with the
			// original newlines, so parsing at the first line's base position
			// keeps every statement mapped to its real source line/column.
			if tok.Literal == "" {
				joined := strings.Join(v, "\n")
				if trimmed := strings.TrimSpace(joined); trimmed != "" {
					base := noBase
					if len(positions) > 0 {
						base = positions[0]
					}
					stmts, err := parseGadAt(joined, base, false)
					if err != nil {
						p.Error(tok.Pos, err.Error())
						return s
					}
					s.Stmts = stmts
				}
				if len(s.Stmts) > 0 {
					s.NodeEnd = s.Stmts[len(s.Stmts)-1].End()
				}
				return s
			}
			// Single-line code (~) — parse each value as an individual statement
			// at the base position of its content, offset by any leading
			// whitespace TrimSpace removes.
			for i, line := range v {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				base := noBase
				if i < len(positions) {
					lead := source.Pos(len(line) - len(strings.TrimLeft(line, " \t")))
					base = positions[i] + lead
				}
				stmt, err := parseGadFirstStmtAt(trimmed, base, false)
				if err == nil {
					s.Stmts = append(s.Stmts, stmt)
				}
			}
		}
	}

	return s
}

// tokenValuePos returns the per-value base positions recorded by the scanner
// for a code token, if present.
func tokenValuePos(tok gadparser.PToken) (positions []source.Pos, ok bool) {
	if v, has := tok.GetOk("valuePos"); has {
		positions, ok = v.([]source.Pos)
	}
	return
}

func (p *Parser) parseImportModule() *giomnode.CodeStmt {
	tok := p.Token
	p.expect(giomtoken.ImportModule)

	path := stringData(tok, "value", "")
	ident := stringData(tok, "ident", "")
	destructure := stringData(tok, "destructure", "")

	var gadSrc string
	switch {
	case destructure != "":
		gadSrc = expandImportDestructure(destructure, path, tok.Pos)
	case ident != "":
		gadSrc = fmt.Sprintf("var %s = import(%s)", ident, path)
	default:
		gadSrc = fmt.Sprintf("import(%s)", path)
	}

	stmts, err := parseGad(gadSrc, nil, false)
	s := &giomnode.CodeStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
	}

	if err == nil && stmts != nil {
		s.Stmts = stmts
	}

	return s
}

func expandImportDestructure(destructure, path string, pos source.Pos) string {
	tmp := fmt.Sprintf("giom_import_%d", pos)
	parts := []string{fmt.Sprintf("var %s = import(%s)", tmp, path)}
	for _, field := range strings.Split(destructure, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if strings.HasPrefix(field, "**") {
			alias := strings.TrimSpace(strings.TrimPrefix(field, "**"))
			if alias != "" {
				parts = append(parts, fmt.Sprintf("var %s = %s", alias, tmp))
			}
			continue
		}
		name := field
		alias := field
		fallback := ""
		if before, after, ok := strings.Cut(field, "="); ok {
			name = strings.TrimSpace(before)
			alias = name
			fallback = strings.TrimSpace(after)
		}
		if before, after, ok := strings.Cut(field, ":"); ok {
			name = strings.TrimSpace(before)
			alias = strings.TrimSpace(after)
		}
		if name == "" || alias == "" {
			continue
		}
		value := fmt.Sprintf("%s.%s", tmp, name)
		if fallback != "" {
			value = fmt.Sprintf("%s ?? %s", value, fallback)
		}
		parts = append(parts, fmt.Sprintf("var %s = %s", alias, value))
	}
	return strings.Join(parts, "\n")
}

// =============================================================================
// Function and component parameter parsing
// =============================================================================

// parseFuncParams parses function parameters from an args string.
func (p *Parser) parseFuncParams(argsStr string, pos source.Pos) *gnode.FuncParams {
	params, err := parseFuncParamsString(argsStr)
	if err != nil {
		panic(fmt.Errorf("parsing func params failed: %v", err))
	}
	return params
}

// parseCompParams parses component parameters from an args string.
func (p *Parser) parseCompParams(argsStr string, pos source.Pos) *gnode.FuncParams {
	params, err := parseFuncParamsString(argsStr)
	if err != nil {
		panic(fmt.Errorf("parsing comp params failed: %v", err))
	}
	return params
}

// =============================================================================
// Surviving content below — preserved exactly as-is
// =============================================================================

// parseAttributeGroup splits the raw inner text of an attribute group into one
// or more attributes. Entries are separated by top-level commas or newlines,
// mirroring GAD KeyValueArray `(; … )`. A trailing `? condition` on the group
// applies to every attribute in it.
func (p *Parser) parseAttributeGroup(tok gadparser.PToken) []*giomnode.TagAttribute {
	inner := stringData(tok, "inner", "")

	var cond gnode.Expr
	if condStr := stringData(tok, "condition", ""); condStr != "" {
		cond = parseExprStr(condStr, tok.Pos)
	}

	base := tok.Pos + 1 // byte after the opening '['
	if v, ok := tok.GetOk("innerPos"); ok {
		if pos, ok := v.(source.Pos); ok {
			base = pos
		}
	}

	var attrs []*giomnode.TagAttribute
	for _, span := range splitAttributeEntries(inner) {
		attr := parseAttributeEntry(inner[span.start:span.end], base+source.Pos(span.start))
		if attr == nil {
			continue
		}
		if cond != nil {
			attr.Condition = cond
		}
		attrs = append(attrs, attr)
	}
	return attrs
}

// isAttrNameChar reports whether c is valid in an attribute name. Names allow
// HTML/framework punctuation such as `xlink:href`, `data-x`, `@click`, `v.on`.
func isAttrNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-' || c == ':' || c == '@' || c == '.'
}

// parseAttributeEntry parses a single `name`, `name=value` or `name="raw"`
// attribute from an entry slice. base is the absolute position of entry[0], so
// the value expression maps back to the original source.
func parseAttributeEntry(entry string, base source.Pos) *giomnode.TagAttribute {
	// Skip leading whitespace, advancing base to keep positions aligned.
	i := 0
	for i < len(entry) && (entry[i] == ' ' || entry[i] == '\t' || entry[i] == '\n' || entry[i] == '\r') {
		i++
	}
	nameStart := i
	for i < len(entry) && isAttrNameChar(entry[i]) {
		i++
	}
	if i == nameStart {
		return nil
	}
	name := entry[nameStart:i]

	j := i
	for j < len(entry) && (entry[j] == ' ' || entry[j] == '\t' || entry[j] == '\n' || entry[j] == '\r') {
		j++
	}
	if j >= len(entry) || entry[j] != '=' {
		// Boolean/flag attribute.
		return &giomnode.TagAttribute{Name: name, IsRaw: true, IsFlag: true}
	}

	// name = value
	valOffset := j + 1
	for valOffset < len(entry) && (entry[valOffset] == ' ' || entry[valOffset] == '\t' || entry[valOffset] == '\n' || entry[valOffset] == '\r') {
		valOffset++
	}
	value := strings.TrimSpace(entry[valOffset:])

	attr := &giomnode.TagAttribute{Name: name}
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		attr.IsRaw = true
		attr.Value = gnode.Str(value[1:len(value)-1], base+source.Pos(valOffset))
	} else if value != "" && value != `""` {
		attr.Value = parseExprStr(value, base+source.Pos(valOffset))
	}
	return attr
}

// attrSpan marks a [start,end) slice of the group's inner text.
type attrSpan struct{ start, end int }

// splitAttributeEntries splits inner attribute text on top-level commas and
// newlines, ignoring separators inside strings, parentheses, brackets and
// braces. Empty spans are dropped.
func splitAttributeEntries(inner string) []attrSpan {
	var spans []attrSpan
	var (
		paren, bracket, brace int
		quote                 byte
		escaped               bool
		start                 = 0
	)
	flush := func(end int) {
		if strings.TrimSpace(inner[start:end]) != "" {
			spans = append(spans, attrSpan{start, end})
		}
		start = end + 1
	}
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if quote != 0 {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '(':
			paren++
		case ')':
			if paren > 0 {
				paren--
			}
		case '[':
			bracket++
		case ']':
			if bracket > 0 {
				bracket--
			}
		case '{':
			brace++
		case '}':
			if brace > 0 {
				brace--
			}
		case ',', '\n':
			if paren == 0 && bracket == 0 && brace == 0 {
				flush(i)
			}
		}
	}
	if strings.TrimSpace(inner[start:]) != "" {
		spans = append(spans, attrSpan{start, len(inner)})
	}
	return spans
}

// parseGlobal parses `@global` in any form:
//
//	@global a                      // single
//	@global a, b, c = 1            // comma-separated, with a default
//	@global a, b, d !?= 2          // absent-only default
//	@global x y z                  // legacy space-separated names
//	@global (a                     // parenthesized, may span lines
//	    b, c = 2)
//
// The declaration body is wrapped in a Gad grouped `global (…)` declaration so
// every form (including `=` / `!?=` defaults) is handled by Gad natively. The
// legacy space-separated form is normalized to commas first.
func (p *Parser) parseGlobal() *giomnode.GlobalStmt {
	tok := p.Token
	p.expect(giomtoken.Global)

	s := &giomnode.GlobalStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
	}

	inner := strings.TrimSpace(stringData(tok, "value", ""))
	if inner == "" {
		return s
	}

	// Legacy space-separated names (no comma, `=` or newline) → comma-separated.
	content := inner
	verbatim := true
	if !strings.ContainsAny(inner, ",=\n") {
		content = strings.Join(strings.Fields(inner), ", ")
		verbatim = content == inner
	}

	base := noBase
	if verbatim {
		if v, ok := tok.GetOk("innerPos"); ok {
			if pos, ok := v.(source.Pos); ok {
				if b := pos - source.Pos(len("global (")); b >= 1 {
					base = b
				}
			}
		}
	}

	stmt, err := parseGadFirstStmtAt("global ("+content+")", base, false)
	if err != nil {
		p.Error(tok.Pos, err.Error())
		return s
	}
	if declStmt, ok := stmt.(*gnode.DeclStmt); ok {
		if decl, ok := declStmt.Decl.(*gnode.GenDecl); ok && decl.Tok == token.Global {
			s.Decl = decl
		}
	}
	return s
}

func (p *Parser) parseVar() *giomnode.VarStmt {
	tok := p.Token
	p.expect(giomtoken.Var)

	decl, decls := p.parseGadDeclDirective(tok, token.Var)

	s := &giomnode.VarStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Decl:    decl,
		Decls:   decls,
	}
	return s
}

func (p *Parser) parseConst() *giomnode.ConstStmt {
	tok := p.Token
	p.expect(giomtoken.Const)

	decl, decls := p.parseGadDeclDirective(tok, token.Const)

	s := &giomnode.ConstStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Decl:    decl,
		Decls:   decls,
	}
	return s
}

// parseEnum parses `@enum IDENT ( … )`. The parenthesized body holds the enum
// fields (same syntax as a `@var` declaration, plus the Gad enum extras `bit`
// and `+`/`-`). It is rewritten into a Gad `enum IDENT { … }` statement and
// parsed by Gad, so every field form is handled natively. The body is parsed at
// its original source position so field positions are preserved.
func (p *Parser) parseEnum() *giomnode.EnumStmt {
	tok := p.Token
	p.expect(giomtoken.Enum)

	name := stringData(tok, "name", "")
	s := &giomnode.EnumStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Name:    name,
	}

	inner := stringData(tok, "value", "")
	prefix := "enum " + name + " { "

	base := noBase
	if v, ok := tok.GetOk("innerPos"); ok {
		if pos, ok := v.(source.Pos); ok {
			if b := pos - source.Pos(len(prefix)); b >= 1 {
				base = b
			}
		}
	}

	stmt, err := parseGadFirstStmtAt(prefix+inner+" }", base, false)
	if err != nil {
		p.Error(tok.Pos, err.Error())
		return s
	}
	if enumStmt, ok := stmt.(*gnode.EnumStmt); ok {
		s.Decl = enumStmt
	} else {
		p.Error(tok.Pos, "expected enum declaration")
	}
	return s
}

// parseGadDeclDirective parses the body of a `@var`/`@const` directive. The body
// (bare or parenthesized) is wrapped in a Gad grouped declaration `kw ( … )` so
// both single and comma/newline-separated forms parse uniformly and `const`
// initializer requirements are enforced by Gad. The body is parsed at its
// original source position so declaration positions are preserved.
func (p *Parser) parseGadDeclDirective(tok gadparser.PToken, want token.Token) (*gnode.GenDecl, []giomnode.VarDecl) {
	inner := stringData(tok, "value", "")
	keyword := "var"
	if want == token.Const {
		keyword = "const"
	}
	prefix := keyword + " ("

	base := noBase
	if v, ok := tok.GetOk("innerPos"); ok {
		if pos, ok := v.(source.Pos); ok {
			if b := pos - source.Pos(len(prefix)); b >= 1 {
				base = b
			}
		}
	}

	stmt, err := parseGadFirstStmtAt(prefix+inner+")", base, false)
	if err != nil {
		p.Error(tok.Pos, err.Error())
		return nil, nil
	}
	declStmt, ok := stmt.(*gnode.DeclStmt)
	if !ok {
		p.Error(tok.Pos, fmt.Sprintf("expected %s declaration", want.String()))
		return nil, nil
	}
	decl, ok := declStmt.Decl.(*gnode.GenDecl)
	if !ok || decl.Tok != want {
		p.Error(tok.Pos, fmt.Sprintf("expected %s declaration", want.String()))
		return nil, nil
	}
	normalizeValueSpecValues(decl)
	return decl, varDeclsFromGenDecl(decl)
}

func normalizeValueSpecValues(decl *gnode.GenDecl) {
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*gnode.ValueSpec)
		if !ok || len(valueSpec.Values) >= len(valueSpec.Idents) {
			continue
		}
		values := make([]gnode.Expr, len(valueSpec.Idents))
		copy(values, valueSpec.Values)
		valueSpec.Values = values
	}
}

func varDeclsFromGenDecl(decl *gnode.GenDecl) []giomnode.VarDecl {
	var decls []giomnode.VarDecl
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*gnode.ValueSpec)
		if !ok {
			continue
		}
		for i, ident := range valueSpec.Idents {
			var init gnode.Expr
			if i < len(valueSpec.Values) {
				init = valueSpec.Values[i]
			}
			decls = append(decls, giomnode.VarDecl{Name: ident.Name, Init: init})
		}
	}
	return decls
}

func (p *Parser) parseFunc() *giomnode.FuncDecl {
	tok := p.Token
	p.expect(giomtoken.Func)

	argsStr := stringData(tok, "args", "")
	exported := stringData(tok, "exported", "") == "true"
	name := stringData(tok, "value", tok.Literal)
	params := p.parseFuncParams(argsStr, tok.Pos)

	f := &giomnode.FuncDecl{
		NodePos:   tok.Pos,
		Name:      name,
		Params:    params,
		ParamsRaw: strings.TrimSpace(argsStr),
		Exported:  exported,
	}

	if p.Token.Token == giomtoken.Indent {
		f.Body = p.parseBlock(f)
	}

	if len(f.Body) > 0 {
		f.NodeEnd = f.Body[len(f.Body)-1].End()
	} else {
		f.NodeEnd = tok.Pos + source.Pos(len(tok.Literal))
	}
	return f
}

func (p *Parser) parseComp() *giomnode.CompDecl {
	tok := p.Token
	p.expect(giomtoken.Comp)

	name := stringData(tok, "value", tok.Literal)
	argsStr := stringData(tok, "args", "")
	exported := stringData(tok, "exported", "") == "true"
	main := stringData(tok, "main", "") == "true"
	params := p.parseCompParams(argsStr, tok.Pos)

	comp := &giomnode.CompDecl{
		NodePos:   tok.Pos,
		Name:      name,
		ID:        strings.ReplaceAll(name, "-", "__"),
		Params:    params,
		ParamsRaw: strings.TrimSpace(argsStr),
		Exported:  exported,
		Main:      main,
	}

	p.compStack = append(p.compStack, comp)
	if p.Token.Token == giomtoken.Indent {
		comp.Body = p.parseBlock(comp)
	}
	p.compStack = p.compStack[:len(p.compStack)-1]

	if len(comp.Body) > 0 {
		comp.NodeEnd = comp.Body[len(comp.Body)-1].End()
	} else {
		comp.NodeEnd = tok.Pos + source.Pos(len(tok.Literal))
	}
	p.comps = append(p.comps, comp)
	return comp
}

// posData reads a source.Pos stored on a scanner token (e.g. "namePos"),
// returning noBase when absent.
func posData(tok gadparser.PToken, key string) source.Pos {
	if v, ok := tok.GetOk(key); ok {
		if p, ok := v.(source.Pos); ok {
			return p
		}
	}
	return noBase
}

// parseSlotNameExpr parses an interpolated slot name (the content of `@slot (…)`
// / `@slot #(…)`) as a Gad template string `#"…"`, so `{expr}` interpolations
// are evaluated at render time. namePos is the absolute position of the content;
// it is offset by the synthetic `#"` prefix so the parsed expression preserves
// the original source positions.
func parseSlotNameExpr(content string, namePos source.Pos) gnode.Expr {
	delim := `"`
	if strings.Contains(content, `"`) {
		delim = "`"
	}
	pos := namePos
	if pos != noBase {
		pos -= 2 // account for the synthetic `#"` (or "#`") prefix
	}
	return parseExprStr("#"+delim+content+delim, pos)
}

func (p *Parser) parseSlot() *giomnode.SlotDecl {
	tok := p.Token
	p.expect(giomtoken.Slot)

	name := stringData(tok, "value", tok.Literal)
	argsStr := stringData(tok, "args", "")
	scope := p.parseFuncParams(argsStr, tok.Pos)

	s := &giomnode.SlotDecl{
		NodePos:  tok.Pos,
		Name:     name,
		ID:       strings.ReplaceAll(name, "-", "__"),
		Scope:    scope,
		ScopeRaw: strings.TrimSpace(argsStr),
	}

	if _, ok := tok.GetOk("nameExpr"); ok {
		s.NameExpr = parseSlotNameExpr(name, posData(tok, "namePos"))
		// The interpolated name is not a valid identifier; use a synthetic id
		// (unique per source position) for the generated local variables.
		s.ID = fmt.Sprintf("d%d", tok.Pos)
	}

	if comp := p.currentComp(); comp != nil {
		comp.Slots = append(comp.Slots, s)
	}

	if p.Token.Token == giomtoken.Indent {
		s.Body = p.parseBlock(s)
		if len(s.Body) > 0 {
			if w, ok := s.Body[0].(*giomnode.WrapStmt); ok {
				s.Wrap = w
				s.Body = s.Body[1:]
			}
		}
	}

	if len(s.Body) > 0 {
		s.NodeEnd = s.Body[len(s.Body)-1].End()
	} else {
		s.NodeEnd = tok.Pos + source.Pos(len(tok.Literal))
	}
	return s
}

func (p *Parser) parseSlotPass() *giomnode.SlotPassStmt {
	tok := p.Token
	p.expect(giomtoken.SlotPass)

	var (
		name     string
		argsStr  string
		nameExpr gnode.Expr
	)
	if _, ok := tok.GetOk("nameExpr"); ok {
		name = stringData(tok, "name", "")
		argsStr = stringData(tok, "args", "")
		nameExpr = parseSlotNameExpr(name, posData(tok, "namePos"))
	} else {
		header := strings.TrimSpace(stringData(tok, "header", stringData(tok, "value", "")))
		name = header
		if i := strings.Index(header, "("); i >= 0 && strings.HasSuffix(header, ")") {
			name = strings.TrimSpace(header[:i])
			argsStr = strings.TrimSpace(header[i+1 : len(header)-1])
		}
	}

	params, err := parseFuncParamsString(argsStr)
	if err != nil {
		panic(fmt.Errorf("parsing slot pass failed: %v", err))
	}

	s := &giomnode.SlotPassStmt{
		NodePos:  tok.Pos,
		Name:     gnode.EIdent(name, tok.Pos),
		NameExpr: nameExpr,
		FuncType: &gnode.FuncType{
			FuncHeader: gnode.FuncHeader{Params: *params},
		},
	}

	if p.Token.Token == giomtoken.Indent {
		s.Body = p.parseBlock(s)
	}

	if len(s.Body) > 0 {
		s.NodeEnd = s.Body[len(s.Body)-1].End()
	} else {
		s.NodeEnd = tok.Pos + source.Pos(len(tok.Literal))
	}
	return s
}

func (p *Parser) parseWrap() *giomnode.WrapStmt {
	tok := p.Token
	p.expect(giomtoken.Wrap)

	w := &giomnode.WrapStmt{
		NodePos: tok.Pos,
	}
	w.Body = p.parseBlock(w)

	if len(w.Body) > 0 {
		w.NodeEnd = w.Body[len(w.Body)-1].End()
	}
	return w
}

func (p *Parser) parseCompCall() *giomnode.CompCallStmt {
	tok := p.Token
	p.expect(giomtoken.CompCall)

	name := stringData(tok, "value", tok.Literal)
	call := &giomnode.CompCallStmt{
		NodePos: tok.Pos,
		Name:    name,
		Func:    compCallFuncExpr(name, tok.Pos),
	}

	if header := stringData(tok, "args", ""); header != "" {
		args, err := parseCallArgsString(header)
		if err != nil {
			panic(err)
		}
		call.Args = *args
	}

	if p.Token.Token == giomtoken.Indent {
		block := p.parseBlock(call)
		var lastMainSlot *giomnode.SlotPassStmt
		for _, child := range block {
			switch t := child.(type) {
			case *giomnode.SlotPassStmt:
				call.SlotPass = append(call.SlotPass, t)
				lastMainSlot = nil
			case *giomnode.CodeStmt:
				// Call-scope code: hoisted before the slot-pass declarations so
				// interpolated slot names and slot bodies can reference it.
				call.InitStmts = append(call.InitStmts, t)
			default:
				if t != nil {
					if lastMainSlot != nil {
						lastMainSlot.Body.Append(t)
						lastMainSlot.NodeEnd = t.End()
					} else {
						lastMainSlot = &giomnode.SlotPassStmt{
							NodePos:  t.Pos(),
							NodeEnd:  t.End(),
							Name:     gnode.EIdent("main", 0),
							FuncType: gnode.ProxyFuncType(),
							Body:     gnode.Stmts{t},
						}
						call.SlotPass = append(call.SlotPass, lastMainSlot)
					}
				}
			}
		}
	}

	call.NodeEnd = callEnd(call)
	return call
}

func (p *Parser) parseMatch() *giomnode.MatchStmt {
	tok := p.Token
	p.expect(giomtoken.Match)

	tagExpr := parseExprStr(stringData(tok, "value", ""), tok.Pos)
	s := &giomnode.MatchStmt{
		NodePos: tok.Pos,
		Tag:     tagExpr,
	}

	if p.Token.Token == giomtoken.Indent {
		p.Next()
	next:
		switch p.Token.Token {
		case giomtoken.Case:
			caseTok := p.Token
			p.expect(giomtoken.Case)
			cc := &giomnode.CaseClause{
				Expr: parseExprStr(stringData(caseTok, "value", ""), caseTok.Pos),
			}
			if p.Token.Token == giomtoken.Indent {
				cc.Body = p.parseBlock(s)
			}
			s.Cases = append(s.Cases, cc)
			goto next
		case giomtoken.Else:
			p.expect(giomtoken.Else)
			if p.Token.Token == giomtoken.Indent {
				s.Default = p.parseBlock(s)
			}
		default:
			p.expect(giomtoken.Case, giomtoken.Else, giomtoken.Outdent)
		}
		p.expect(giomtoken.Outdent)
	}

	s.NodeEnd = matchEnd(s)
	return s
}

func (p *Parser) parseExport() *giomnode.ExportStmt {
	tok := p.Token
	p.expect(giomtoken.Export)

	name := stringData(tok, "name", stringData(tok, "value", ""))
	valueStr := stringData(tok, "value", "")

	e := &giomnode.ExportStmt{
		NodePos: tok.Pos,
		NodeEnd: tok.Pos + source.Pos(len(tok.Literal)),
		Name:    name,
	}
	if valueStr != "" {
		e.Value = gnode.EIdent(valueStr, tok.Pos)
	}
	return e
}

// =============================================================================
// Helpers
// =============================================================================

func stringData(pt gadparser.PToken, key, def string) string {
	if v, ok := pt.GetOk(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// exprReturnPrefix wraps expression fragments so gad parses them as an
// expression (handling {} dict literals etc.). Its length shifts the fragment
// base so parsed positions still map onto the original source.
const exprReturnPrefix = "return "

func parseExprStr(s string, pos source.Pos) gnode.Expr {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return gnode.EIdent("", pos)
	}
	// Map the fragment back onto the original source: pos marks where s begins,
	// lead skips whitespace TrimSpace removed, and the "return " prefix is
	// compensated so the expression itself lands at pos+lead.
	base := noBase
	if pos != source.NoPos {
		lead := source.Pos(len(s) - len(strings.TrimLeft(s, " \t\r\n")))
		if b := pos + lead - source.Pos(len(exprReturnPrefix)); b >= 1 {
			base = b
		}
	}
	stmt, err := parseGadFirstStmtAt(exprReturnPrefix+trimmed, base, false)
	if err != nil {
		return gnode.EIdent(trimmed, pos)
	}
	if rs, ok := stmt.(*gnode.ReturnStmt); ok && rs.Result != nil {
		return rs.Result
	}
	return gnode.EIdent(trimmed, pos)
}

func compCallFuncExpr(name string, pos source.Pos) gnode.Expr {
	if strings.Contains(name, ".") {
		return parseExprStr(name, pos)
	}
	return gnode.EIdent(strings.ReplaceAll(name, "-", "__"), pos)
}

func splitTopLevelArgs(s string) []string {
	var (
		parts   []string
		start   int
		depth   int
		quote   byte
		escaped bool
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ',', ';':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func topLevelAssignIndex(s string) int {
	var (
		depth   int
		quote   byte
		escaped bool
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"', '`':
			quote = c
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '=':
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func callEnd(call *giomnode.CompCallStmt) source.Pos {
	if len(call.SlotPass) > 0 {
		return call.SlotPass[len(call.SlotPass)-1].End()
	}
	return call.Pos()
}

func matchEnd(s *giomnode.MatchStmt) source.Pos {
	if len(s.Default) > 0 {
		return s.Default[len(s.Default)-1].End()
	}
	if len(s.Cases) > 0 {
		last := s.Cases[len(s.Cases)-1]
		if len(last.Body) > 0 {
			return last.Body[len(last.Body)-1].End()
		}
	}
	return s.Pos()
}

func bodyEndIf(s *giomnode.IfStmt) source.Pos {
	if len(s.Else) > 0 {
		return s.Else[len(s.Else)-1].End()
	}
	if len(s.ElseIfs) > 0 {
		last := s.ElseIfs[len(s.ElseIfs)-1]
		if len(last.Body) > 0 {
			return last.Body[len(last.Body)-1].End()
		}
	}
	if len(s.Body) > 0 {
		return s.Body[len(s.Body)-1].End()
	}
	return s.Pos()
}

func bodyEndFor(s *giomnode.ForStmt) source.Pos {
	if len(s.Else) > 0 {
		return s.Else[len(s.Else)-1].End()
	}
	if len(s.Body) > 0 {
		return s.Body[len(s.Body)-1].End()
	}
	return s.Pos()
}
