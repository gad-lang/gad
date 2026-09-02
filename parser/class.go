package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// ParseClassExpr parses an anonymous class expression `class { … }` (the
// expression form). The statement form with a name is parsed by ParseClassStmt.
func (p *Parser) ParseClassExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "TypeLitExpr"))
	}
	doc := p.leadComment
	classTok := p.expectContextualKeyword(token.Class)
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
		defer untracep(tracep(p, "TypeDeclStmt"))
	}
	doc := p.leadComment
	classTok := p.expectContextualKeyword(token.Class)

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
	return &node.TypeDeclStmt{TypeLitExpr: *cls}
}

// ParseMixinExpr parses a `mixin [Name] { … }` expression. A mixin shares the
// class body grammar (parents, fields, props, methods) plus an optional `this`
// interface block; it has no `new` clause and lowers to `gad.Mixin(...)`.
func (p *Parser) ParseMixinExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "MixinExpr"))
	}
	doc := p.leadComment
	tok := p.expectContextualKeyword(token.Mixin)
	cls := p.parseClassBody(tok, nil)
	if cls != nil {
		cls.Doc = doc
	}
	return cls
}

// ParseMixinStmt parses the statement form `mixin Name { … }` (a const bind), or
// an anonymous `mixin { … }` as an expression statement.
func (p *Parser) ParseMixinStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "MixinStmt"))
	}
	doc := p.leadComment
	tok := p.expectContextualKeyword(token.Mixin)
	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}
	cls := p.parseClassBody(tok, name)
	if cls == nil {
		return &node.BadStmt{From: tok.Pos, To: p.Token.Pos}
	}
	cls.Doc = doc
	if name == nil {
		return &node.ExprStmt{Expr: cls}
	}
	return &node.TypeDeclStmt{TypeLitExpr: *cls}
}

// ParseStaticTypeExpr parses an anonymous marker-type expression `type { … }`
// (the expression form used in `const X = type { … }`). The current token is the
// contextual `type` identifier.
func (p *Parser) ParseStaticTypeExpr() node.Expr {
	if p.Trace {
		defer untracep(tracep(p, "StaticTypeExpr"))
	}
	doc := p.leadComment
	tok := p.Token // the contextual `type` ident
	p.Next()
	cls := p.parseClassBody(PToken{TokenLit: tok.TokenLit}, nil)
	if cls != nil {
		cls.Static = true
		cls.Doc = doc
	}
	return cls
}

// ParseStaticTypeStmt parses the statement form `type Name { … }` (a const bind),
// or an anonymous `type { … }` as an expression statement. The current token is
// the contextual `type` identifier.
func (p *Parser) ParseStaticTypeStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "StaticTypeStmt"))
	}
	doc := p.leadComment
	tok := p.Token
	p.Next()
	var name node.Expr
	if p.Token.Token == token.Ident {
		name = p.ParseIdent()
	}
	cls := p.parseClassBody(PToken{TokenLit: tok.TokenLit}, name)
	if cls == nil {
		return &node.BadStmt{From: tok.Pos, To: p.Token.Pos}
	}
	cls.Static = true
	cls.Doc = doc
	if name == nil {
		return &node.ExprStmt{Expr: cls}
	}
	return &node.TypeDeclStmt{TypeLitExpr: *cls}
}

// parseClassBody parses the `{ … }` body of a class (including the `extends { … }`
// block), shared by the expression and statement forms.
func (p *Parser) parseClassBody(classTok PToken, name node.Expr) *node.TypeLitExpr {
	cls := &node.TypeLitExpr{ClassToken: classTok.TokenLit, NameExpr: name, Mixin: classTok.Token == token.Mixin}

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
func (p *Parser) parseClassBodyItem(cls *node.TypeLitExpr) {
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
		case "use":
			// `use A, pkg.B` — the mixins a class pulls in (a contextual ident, not a
			// keyword). Each name is an ident or selector expression; names are
			// comma-separated (a comma may precede a newline); the list ends at a
			// newline or `;` with no trailing comma. Only triggers when followed by a
			// name, so a plain `use` ident/value is unaffected.
			if pk := p.Peek().Token; pk == token.Ident {
				p.Next()
				p.SkipSpace()
				if cls.UseDoc == nil {
					cls.UseDoc = doc
				}
				for {
					cls.Use = append(cls.Use, p.ParsePrimaryExpr())
					p.SkipSpace()
					if p.Token.Token != token.Comma {
						break
					}
					p.Next()
					p.skipClassSeps()
				}
				return
			}
		case "this":
			// `this { … }` — the interface the mixin's `this` must satisfy (mixin-only).
			if p.Peek().Token == token.LBrace {
				p.Next()
				cls.ThisDoc = doc
				cls.This = p.parseInterfaceBody(PToken{}, nil)
				return
			}
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
		case "call":
			// `call(…)` — a marker type's factory (Static only), parsed exactly like
			// `new` overloads. Contextual: only when followed by `(`/`{`, so a plain
			// `call` field/name is unaffected.
			if pk := p.Peek().Token; pk == token.LParen || pk == token.LBrace {
				p.Next()
				cls.CallDoc = doc
				cls.Call = append(cls.Call, p.parseClassConstructors()...)
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
// `name { overloads }`, or the zero-arg accessor shortcuts `name = expr` and
// `name => expr` (both a getter `() => expr`).
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
	case token.Assign, token.Lambda:
		// `name = expr` and `name => expr` are the same getter shortcut: a
		// zero-argument accessor `() => expr`.
		p.Next()
		m.Methods = append(m.Methods, &node.FuncMethod{BodyExpr: p.ParseExpr()})
	default:
		p.ErrorExpectToken(p.Token, token.LParen, token.LBrace, token.Assign, token.Lambda)
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
