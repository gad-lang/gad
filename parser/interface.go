package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// ParseInterfaceExpr parses an anonymous interface expression `interface { … }`
// (or an array interface `interface []{ … }` / `interface []<int|uint>`). The
// statement form with a name is parsed by ParseInterfaceStmt.
func (p *Parser) ParseInterfaceExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "InterfaceExpr"))
	}
	doc := p.leadComment
	meta := p.takeMeta() // before the body: a member must not steal the iface's meta
	tok := p.expectContextualKeyword(token.Interface)
	iface := p.parseInterfaceDecl(tok)
	if iface != nil {
		iface.Doc = doc
		iface.Meta = meta
	}
	return iface
}

// parseInterfaceArrayDepth consumes a run of `[]` pairs (the array depth of an
// array interface, `interface P [][] { … }`) and returns their count, 0 when
// there is none.
func (p *Parser) parseInterfaceArrayDepth() (depth int) {
	for p.Token.Token == token.LBrack {
		p.Next()
		p.Expect(token.RBrack)
		depth++
	}
	return
}

// parseInterfaceDecl parses what follows the `interface` keyword:
//
//	interface [NAME] { … }                 // a plain interface
//	interface [NAME] [][]… { … }           // an array interface (array of the body)
//	interface [NAME] [] interface { … }    // the same, long form
//	interface [NAME] []<int|uint>          // an array-of-types interface (no body)
//	interface [NAME] []int                 // the same, one bare element type
//
// The `[]`s follow the name (`interface NAME [] { … }`); the former
// `interface[] NAME { … }` order is rejected with a hint.
func (p *Parser) parseInterfaceDecl(tok PToken) *node.InterfaceExpr {
	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}
	depth := p.parseInterfaceArrayDepth()

	if depth == 0 {
		return p.parseInterfaceBody(tok, name)
	}

	switch {
	case p.Token.Token == token.Ident && p.Token.Literal == "interface" && p.Peek().Token == token.LBrace:
		// `[] interface { … }` — the long form of `[] { … }`.
		p.Next()
		fallthrough
	case p.Token.Token == token.LBrace:
		iface := p.parseInterfaceBody(tok, name)
		if iface != nil {
			iface.ArrayDepth = depth
		}
		return iface
	case name == nil && p.Token.Token == token.Ident && p.Peek().Token == token.LBrace:
		p.Error(p.Token.Pos, "the `[]` of a named array interface follows its name: "+
			"write `interface "+p.Token.Literal+" [] { … }`")
		return nil
	case p.Token.Token == token.Less || p.Token.Token == token.Shl:
		// `[]<T1|T2>` — the element types in the angle envelope.
		iface := &node.InterfaceExpr{InterfaceToken: tok.TokenLit, NameExpr: name, ArrayDepth: depth}
		p.consumeLess()
		p.SkipSpace()
		iface.ElemTypes = p.ParseTypes()
		if len(iface.ElemTypes) == 0 {
			p.ErrorExpected(p.Token.Pos, "element type")
			return nil
		}
		p.SkipSpace()
		iface.RAngle = p.expectGreater()
		return iface
	case p.isTypeStart():
		// `[]int` — one bare element type.
		iface := &node.InterfaceExpr{InterfaceToken: tok.TokenLit, NameExpr: name, ArrayDepth: depth}
		if t := p.parseType(); t != nil {
			iface.ElemTypes = []*node.TypeExpr{t}
		}
		return iface
	}
	p.ErrorExpected(p.Token.Pos, "'{', 'interface { … }' or an element type after `[]`")
	return nil
}

// ParseInterfaceStmt parses the statement form. `interface Name { … }` (or a
// array interface `interface Name [] { … }` / `interface Name []<int|uint>`)
// becomes `const Name = <interface expression>`; an anonymous `interface { … }`
// used as a statement is parsed as an expression statement.
func (p *Parser) ParseInterfaceStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "InterfaceStmt"))
	}
	doc := p.leadComment
	meta := p.takeMeta() // before the body: a member must not steal the iface's meta
	tok := p.expectContextualKeyword(token.Interface)

	iface := p.parseInterfaceDecl(tok)
	if iface == nil {
		return &node.BadStmt{From: tok.Pos, To: p.Token.Pos}
	}
	iface.Doc = doc
	iface.Meta = meta

	if iface.NameExpr == nil {
		return &node.ExprStmt{Expr: iface}
	}
	return &node.InterfaceStmt{InterfaceExpr: *iface}
}

// parseInterfaceBody parses the `{ … }` body of an interface, shared by the
// expression and statement forms.
func (p *Parser) parseInterfaceBody(tok PToken, name node.Expr) *node.InterfaceExpr {
	iface := &node.InterfaceExpr{InterfaceToken: tok.TokenLit, NameExpr: name}

	p.SkipSpace()
	iface.LBrace = p.Expect(token.LBrace)

	p.ExprLevel++
	for {
		p.skipClassSeps()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
			break
		}
		p.parseInterfaceBodyItem(iface)
		if p.Failed() {
			break
		}
	}
	p.ExprLevel--

	iface.RBrace = p.Expect(token.RBrace)
	return iface
}

// parseInterfaceBodyItem parses one interface body item: a `*Parent` spread
// (a parent interface), a `get`/`set`/`prop` accessor, a method (`name(params)
// <return>`) or a typed field (`name [Type]`).
func (p *Parser) parseInterfaceBodyItem(iface *node.InterfaceExpr) {
	p.parseMemberMeta()
	doc := p.leadComment
	meta := p.takeMeta()

	// `funcs { FnExpr <header>; … }` — the context-function section: each entry is
	// a free function (captured by value where the interface is declared) that must
	// handle the interface's object, with `@self` standing for the interface.
	if p.Token.Token == token.Ident && p.Token.Literal == "funcs" && p.Peek().Token == token.LBrace {
		p.Next() // consume `funcs`
		p.SkipSpace()
		p.Expect(token.LBrace)
		p.ExprLevel++
		for {
			p.skipClassSeps()
			if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
				break
			}
			itemDoc := p.leadComment
			fn := p.ParsePrimaryExpr()
			if fn == nil || p.Failed() {
				break
			}
			cf := &node.InterfaceContextFuncExpr{FnExpr: fn, Doc: itemDoc}
			if !p.parseContextFuncHeaders(cf) {
				break
			}
			iface.ContextFuncs = append(iface.ContextFuncs, cf)
		}
		p.ExprLevel--
		p.Expect(token.RBrace)
		return
	}

	// `**name` — a rest-capture field: on a dict cast (`d :: I`) the keys not named
	// by the interface are collected into a dict bound to `name` in the result.
	if p.Token.Token == token.Pow {
		p.Next()
		p.SkipSpace()
		if name := p.ParseIdent(); name != nil {
			iface.Rest = name
			iface.RestDoc = doc
		}
		return
	}

	// `*Parent` — a parent interface, written as a spread body item.
	if p.Token.Token == token.Mul {
		p.Next()
		p.SkipSpace()
		if typ := p.ParsePrimaryExpr(); typ != nil {
			if iface.ExtendsDoc == nil {
				iface.ExtendsDoc = doc
			}
			iface.Parents = append(iface.Parents, typ)
		}
		return
	}

	// `prop name [Type]` — prop is a reserved keyword.
	if p.Token.Token == token.Prop {
		kw := p.Token.Pos
		p.Next()
		p.SkipSpace()
		iface.Members = append(iface.Members, &node.InterfaceMemberExpr{
			Kind: node.IfaceProp, KwPos: kw, Name: p.ParseTypedIdent(), Doc: doc, Meta: meta,
		})
		return
	}

	if p.Token.Token == token.Ident {
		switch p.Token.Literal {
		case "get", "set":
			if p.Peek().Token == token.Ident {
				kind := node.IfaceGet
				if p.Token.Literal == "set" {
					kind = node.IfaceSet
				}
				kw := p.Token.Pos
				p.Next()
				p.SkipSpace()
				iface.Members = append(iface.Members, &node.InterfaceMemberExpr{
					Kind: kind, KwPos: kw, Name: p.ParseTypedIdent(), Doc: doc, Meta: meta,
				})
				return
			}
		}
	}

	// A method (single `name(...)` or block `name { (…), … }`) or a typed field
	// (`name [Type]`).
	name := p.ParseIdent()
	if name == nil {
		return
	}

	// A `?` right after the name marks a field nullable (may be nil): `x? int`,
	// `a?: { … }`. It comes after the name and before the type / `:` (the current
	// convention for the space form).
	var nullable bool
	if p.Token.Is(token.Question) {
		nullable = true
		p.Next()
		p.SkipSpace()
	}

	switch p.Token.Token {
	case token.Colon:
		// `name: { … }` — shorthand for a nested-interface field
		// (`name interface { … }`). A leading `[]` makes it an array interface,
		// `name: []{ … }` == `name interface [] { … }` (each element must satisfy
		// the body); `[][]` nests deeper. The colon form is ONLY for a nested
		// interface: after the optional `[]`s it must be followed by `{`. (The
		// brace form without a colon, `name { … }`, stays a block method; a plain
		// typed field is `name Type`.)
		p.Next()
		p.SkipSpace()
		depth := p.parseInterfaceArrayDepth()
		p.SkipSpace()
		if p.Token.Token != token.LBrace {
			p.ErrorExpected(p.Token.Pos, "'{' (`name: []… { … }` is only for a nested interface)")
			return
		}
		nested := p.parseInterfaceBody(PToken{}, nil)
		if nested != nil {
			nested.ArrayDepth = depth
		}
		iface.Members = append(iface.Members, &node.InterfaceMemberExpr{
			Kind: node.IfaceField,
			Name: &node.TypedIdentExpr{Ident: name, Type: []*node.TypeExpr{{Expr: nested}}, Nullable: nullable},
			Doc:  doc,
			Meta: meta,
		})
	case token.LParen:
		h := p.parseInterfaceMethodHeader()
		if h == nil {
			return
		}
		iface.Methods = append(iface.Methods, &node.InterfaceMethodExpr{
			NameExpr: name, Headers: []*node.FuncHeaderExpr{h}, Doc: doc, Meta: meta,
		})
	case token.LBrace:
		m := &node.InterfaceMethodExpr{NameExpr: name, Block: true, Doc: doc, Meta: meta}
		m.LBrace = p.Expect(token.LBrace)
		p.ExprLevel++
		for {
			p.skipClassSeps()
			if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
				break
			}
			h := p.parseInterfaceMethodHeader()
			if h == nil || p.Failed() {
				break
			}
			m.Headers = append(m.Headers, h)
		}
		p.ExprLevel--
		m.RBrace = p.Expect(token.RBrace)
		iface.Methods = append(iface.Methods, m)
	default:
		iface.Members = append(iface.Members, &node.InterfaceMemberExpr{
			Kind: node.IfaceField,
			Name: &node.TypedIdentExpr{Ident: name, Type: p.ParseTypes(), Nullable: nullable},
			Doc:  doc,
			Meta: meta,
		})
	}
}

// parseContextFuncHeaders parses the signature part of a context-function member
// (cf.FnExpr already set): a shortcut `<(params)>` or a brace block
// `{ (params); … }`. It reports whether parsing succeeded.
func (p *Parser) parseContextFuncHeaders(cf *node.InterfaceContextFuncExpr) bool {
	switch p.Token.Token {
	case token.Less:
		h, _ := p.ParseFuncHeaderExpr().(*node.FuncHeaderExpr)
		if h == nil || p.Failed() {
			return false
		}
		cf.Headers = []*node.FuncHeaderExpr{h}
	case token.LBrace:
		cf.Block = true
		cf.LBrace = p.Expect(token.LBrace)
		p.ExprLevel++
		for {
			p.skipClassSeps()
			if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
				break
			}
			h := p.parseInterfaceMethodHeader()
			if h == nil || p.Failed() {
				break
			}
			cf.Headers = append(cf.Headers, h)
		}
		p.ExprLevel--
		cf.RBrace = p.Expect(token.RBrace)
	default:
		p.Error(p.Token.Pos, "expected a function header `<(...)>` or `{ (...) }` after `"+cf.FnExpr.String()+"`")
		return false
	}
	return true
}

// parseInterfaceMethodHeader parses one anonymous method signature `(params)
// <return>`. Bare positional entries are types (`(int)` -> `(_ int)`), like
// `meti`. The method name is carried by the enclosing InterfaceMethodExpr.
func (p *Parser) parseInterfaceMethodHeader() *node.FuncHeaderExpr {
	paren := p.ParseParemExpr(token.LParen, token.RParen)
	if paren == nil || p.Errors.Len() != 0 {
		return nil
	}
	params, err := paren.ToMultiParenExpr().ToFuncHeaderParams()
	if err != nil {
		p.Error(err.Pos(), err.Error())
		return nil
	}
	return &node.FuncHeaderExpr{
		FuncHeader: node.FuncHeader{Params: params, Return: p.ParseFuncReturnTypes()},
	}
}
