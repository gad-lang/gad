package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// ParseClassExpr parses an anonymous class expression `class { … }` (the
// expression form). The statement form with a name is parsed by ParseClassStmt.
func (p *Parser) ParseClassExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "ClassExpr"))
	}
	doc := p.leadComment
	classTok := p.ExpectToken(token.Class)
	cls := p.parseClassBody(classTok, nil)
	if cls != nil {
		cls.Doc = doc
	}
	return cls
}

// ParseClassStmt parses the statement form. `class Name { … }` becomes
// `const Name = <class expression>`; an anonymous `class { … }` used as a
// statement is parsed as an expression statement.
func (p *Parser) ParseClassStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "ClassStmt"))
	}
	doc := p.leadComment
	classTok := p.ExpectToken(token.Class)

	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}

	cls := p.parseClassBody(classTok, name)
	if cls == nil {
		return &node.BadStmt{From: classTok.Pos, To: p.Token.Pos}
	}
	cls.Doc = doc

	if name == nil {
		return &node.ExprStmt{Expr: cls}
	}
	return &node.ClassStmt{ClassExpr: *cls}
}

// parseClassBody parses the `{ … }` body of a class (including the `extends { … }`
// block), shared by the expression and statement forms.
func (p *Parser) parseClassBody(classTok PToken, name node.Expr) *node.ClassExpr {
	cls := &node.ClassExpr{ClassToken: classTok.TokenLit, NameExpr: name}

	p.SkipSpace()
	cls.LBrace = p.Expect(token.LBrace)

	p.ExprLevel++
	for {
		p.skipClassSeps()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
			break
		}
		p.parseClassBodyItem(cls)
		if p.Failed() {
			break
		}
	}
	p.ExprLevel--

	cls.RBrace = p.Expect(token.RBrace)
	return cls
}

// parseClassBodyItem parses one top-level class body item: a `*Parent` spread
// (a parent class, optionally aliased `*Parent: Alias`), a `props {}` /
// `methods {}` / `new` block, or a field.
func (p *Parser) parseClassBodyItem(cls *node.ClassExpr) {
	doc := p.leadComment

	// `*Parent [: Alias]` — a parent class, written as a spread body item.
	if p.Token.Token == token.Mul {
		p.Next()
		p.SkipSpace()
		if parent := p.parseClassParent(); parent != nil {
			if cls.ExtendsDoc == nil {
				cls.ExtendsDoc = doc
			}
			cls.Parents = append(cls.Parents, parent)
		}
		return
	}

	if p.Token.Token == token.Ident {
		switch p.Token.Literal {
		case "props":
			if p.Peek().Token == token.LBrace {
				p.Next()
				cls.PropsDoc = doc
				cls.Props = append(cls.Props, p.parseClassMemberBlock()...)
				return
			}
		case "methods":
			if p.Peek().Token == token.LBrace {
				p.Next()
				cls.MethodsDoc = doc
				cls.Methods = append(cls.Methods, p.parseClassMemberBlock()...)
				return
			}
		case "new":
			if pk := p.Peek().Token; pk == token.LParen || pk == token.LBrace {
				p.Next()
				cls.NewDoc = doc
				cls.New = append(cls.New, p.parseClassConstructors()...)
				return
			}
		}
	}

	if f := p.parseClassField(); f != nil {
		f.Doc = doc
		cls.Fields = append(cls.Fields, f)
	}
}

// parseClassParent parses one `*Parent` entry: a parent type (IdentExpr or
// SelectorExpr) with an optional `: Alias`. The leading `*` is consumed by the
// caller.
func (p *Parser) parseClassParent() *node.ClassParentExpr {
	typ := p.ParsePrimaryExpr()
	if typ == nil {
		return nil
	}
	parent := &node.ClassParentExpr{Type: typ}
	p.SkipSpace()
	if p.Token.Token == token.Colon {
		p.Next()
		p.SkipSpace()
		parent.Alias = p.ParseIdent()
	}
	return parent
}

// parseClassField parses `name`, `name Type`, `name = value`, `name Type =
// value` or a computed default `name = (= expr)`.
func (p *Parser) parseClassField() *node.ClassFieldExpr {
	name := p.ParseTypedIdent()
	if name == nil {
		return nil
	}
	f := &node.ClassFieldExpr{Name: name}
	if p.Token.Token == token.Assign {
		f.Assign = p.Token.Pos
		p.Next()
		f.Value = p.ParseExpr()
	}
	return f
}

// parseClassMemberBlock parses the entries of a `props {}` / `methods {}` block.
func (p *Parser) parseClassMemberBlock() (members []*node.ClassMemberExpr) {
	p.Expect(token.LBrace)
	p.ExprLevel++
	for {
		p.skipClassSeps()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
			break
		}
		m := p.parseClassMember()
		if m == nil || p.Failed() {
			break
		}
		members = append(members, m)
	}
	p.ExprLevel--
	p.Expect(token.RBrace)
	return
}

// parseClassMember parses one `props`/`methods` entry: `name(params) body`,
// `name { overloads }`, or the `name = expr` shortcut (a zero-arg accessor).
func (p *Parser) parseClassMember() *node.ClassMemberExpr {
	doc := p.leadComment
	m := &node.ClassMemberExpr{NameExpr: p.ParseIdent(), Doc: doc}

	switch p.Token.Token {
	case token.LParen:
		fm := p.parsePropMethod()
		if fm == nil {
			return nil
		}
		m.Methods = append(m.Methods, fm)
	case token.LBrace:
		m.Block = true
		m.LBrace = p.Expect(token.LBrace)
		p.ExprLevel++
		for {
			p.skipClassSeps()
			if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
				break
			}
			fm := p.parsePropMethod()
			if fm == nil || p.Failed() {
				break
			}
			m.Methods = append(m.Methods, fm)
		}
		p.ExprLevel--
		m.RBrace = p.Expect(token.RBrace)
	case token.Assign:
		p.Next()
		m.Methods = append(m.Methods, &node.FuncMethod{BodyExpr: p.ParseExpr()})
	default:
		p.ErrorExpectToken(p.Token, token.LParen, token.LBrace, token.Assign)
		return nil
	}
	return m
}

// parseClassConstructors parses the `new` clause: a single `new(params) body`
// or a `new { (params) body … }` overload block.
func (p *Parser) parseClassConstructors() (methods []*node.FuncMethod) {
	switch p.Token.Token {
	case token.LParen:
		if fm := p.parsePropMethod(); fm != nil {
			methods = append(methods, fm)
		}
	case token.LBrace:
		p.Expect(token.LBrace)
		p.ExprLevel++
		for {
			p.skipClassSeps()
			if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
				break
			}
			fm := p.parsePropMethod()
			if fm == nil || p.Failed() {
				break
			}
			methods = append(methods, fm)
		}
		p.ExprLevel--
		p.Expect(token.RBrace)
	}
	return
}

// skipClassSeps consumes class body item separators: whitespace, newlines
// (auto-semicolons) and commas.
func (p *Parser) skipClassSeps() {
	for {
		if p.Token.IsSpace() {
			p.Next()
			continue
		}
		switch p.Token.Token {
		case token.Semicolon, token.Comma:
			p.Next()
		default:
			return
		}
	}
}
