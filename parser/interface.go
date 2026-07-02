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
	if p.Token.Token == token.Ident && p.Token.Literal != "extends" {
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
	if p.Token.Token == token.Ident && p.Token.Literal != "extends" {
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

// parseInterfaceBodyItem parses one interface body item: an `extends {}` /
// `parse {}` block, a `get`/`set`/`prop` accessor, a method (`name(params)
// <return>`) or a typed field (`name [Type]`).
func (p *Parser) parseInterfaceBodyItem(iface *node.InterfaceExpr) {
	doc := p.leadComment

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
		case "extends":
			if p.Peek().Token == token.LBrace {
				p.Next()
				iface.ExtendsDoc = doc
				iface.Parents = append(iface.Parents, p.parseInterfaceExtendsBlock()...)
				return
			}
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
		iface.Members = append(iface.Members, &node.InterfaceMemberExpr{
			Kind: node.IfaceField,
			Name: &node.TypedIdentExpr{Ident: name, Type: p.ParseTypes()},
			Doc:  doc,
		})
	}
}

// parseInterfaceExtendsBlock parses `extends { Parent, … }` — parent interfaces
// (IdentExpr or SelectorExpr) separated by commas or newlines, without alias.
func (p *Parser) parseInterfaceExtendsBlock() (parents []node.Expr) {
	p.Expect(token.LBrace)
	p.ExprLevel++
	for {
		p.skipClassSeps()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
			break
		}
		typ := p.ParsePrimaryExpr()
		if typ == nil || p.Failed() {
			break
		}
		parents = append(parents, typ)
	}
	p.ExprLevel--
	p.Expect(token.RBrace)
	return
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
