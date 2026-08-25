package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// ParseInterfaceExpr parses an anonymous interface expression `interface { … }`.
// The statement form with a name is parsed by ParseInterfaceStmt.
func (p *Parser) ParseInterfaceExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "InterfaceExpr"))
	}
	doc := p.leadComment
	tok := p.ExpectToken(token.Interface)
	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}
	iface := p.parseInterfaceBody(tok, name)
	if iface != nil {
		iface.Doc = doc
	}
	return iface
}

// ParseInterfaceStmt parses the statement form. `interface Name { … }` becomes
// `const Name = <interface expression>`; an anonymous `interface { … }` used as
// a statement is parsed as an expression statement.
func (p *Parser) ParseInterfaceStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "InterfaceStmt"))
	}
	doc := p.leadComment
	tok := p.ExpectToken(token.Interface)

	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}

	iface := p.parseInterfaceBody(tok, name)
	if iface == nil {
		return &node.BadStmt{From: tok.Pos, To: p.Token.Pos}
	}
	iface.Doc = doc

	if name == nil {
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
	doc := p.leadComment

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
			Kind: node.IfaceProp, KwPos: kw, Name: p.ParseTypedIdent(), Doc: doc,
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
					Kind: kind, KwPos: kw, Name: p.ParseTypedIdent(), Doc: doc,
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
	switch p.Token.Token {
	case token.LParen:
		h := p.parseInterfaceMethodHeader()
		if h == nil {
			return
		}
		iface.Methods = append(iface.Methods, &node.InterfaceMethodExpr{
			NameExpr: name, Headers: []*node.FuncHeaderExpr{h}, Doc: doc,
		})
	case token.LBrace:
		m := &node.InterfaceMethodExpr{NameExpr: name, Block: true, Doc: doc}
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
		// A `?` right after the name marks the field nullable (`x? int`).
		var nullable bool
		if p.Token.Is(token.Question) {
			nullable = true
			p.Next()
			p.SkipSpace()
		}
		iface.Members = append(iface.Members, &node.InterfaceMemberExpr{
			Kind: node.IfaceField,
			Name: &node.TypedIdentExpr{Ident: name, Type: p.ParseTypes(), Nullable: nullable},
			Doc:  doc,
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
