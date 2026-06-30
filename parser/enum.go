package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// ParseEnumExpr parses an anonymous enum expression `enum { … }` (the
// expression form). The statement form with a name is parsed by ParseEnumStmt.
func (p *Parser) ParseEnumExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "EnumExpr"))
	}
	doc := p.leadComment
	enumTok := p.ExpectToken(token.Enum)
	e := p.parseEnumBody(enumTok, nil)
	if e != nil {
		e.Doc = doc
	}
	return e
}

// ParseEnumStmt parses the statement form. `enum Name { … }` becomes
// `const Name = <enum expression>`; an anonymous `enum { … }` used as a
// statement is parsed as an expression statement.
func (p *Parser) ParseEnumStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "EnumStmt"))
	}
	doc := p.leadComment
	enumTok := p.ExpectToken(token.Enum)

	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}

	e := p.parseEnumBody(enumTok, name)
	if e == nil {
		return &node.BadStmt{From: enumTok.Pos, To: p.Token.Pos}
	}
	e.Doc = doc

	if name == nil {
		return &node.ExprStmt{Expr: e}
	}
	return &node.EnumStmt{EnumExpr: *e}
}

// parseEnumBody parses the `{ field, … }` body shared by the expression and
// statement forms.
func (p *Parser) parseEnumBody(enumTok PToken, name node.Expr) *node.EnumExpr {
	e := &node.EnumExpr{EnumToken: enumTok.TokenLit, NameExpr: name}

	p.SkipSpace()
	e.LBrace = p.Expect(token.LBrace)

	p.ExprLevel++
	for {
		p.skipClassSeps()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
			break
		}
		f := p.parseEnumField()
		if f == nil || p.Failed() {
			break
		}
		e.Fields = append(e.Fields, f)
	}
	p.ExprLevel--

	e.RBrace = p.Expect(token.RBrace)
	return e
}

// parseEnumField parses one field: `[bit] [+|-] IDENT [= Expr]`.
func (p *Parser) parseEnumField() *node.EnumFieldExpr {
	f := &node.EnumFieldExpr{Doc: p.leadComment}

	// `bit` prefix: only when followed by another field token (sign or ident),
	// so a field may still be named `bit`.
	if p.Token.Token == token.Ident && p.Token.Literal == "bit" {
		if pk := p.Peek().Token; pk == token.Add || pk == token.Sub || pk == token.Ident {
			f.Bit = true
			p.Next()
			p.SkipSpace()
		}
	}

	if p.Token.Token == token.Add || p.Token.Token == token.Sub {
		f.Sign = p.Token.Token
		f.SignPos = p.Token.Pos
		p.Next()
	}

	if p.Token.Token != token.Ident {
		p.ErrorExpectToken(p.Token, token.Ident)
		return nil
	}
	f.Name = p.ParseIdent()

	if p.Token.Token == token.Assign {
		f.Assign = p.Token.Pos
		p.Next()
		f.Value = p.ParseExpr()
	}
	return f
}
