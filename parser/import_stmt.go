package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// ParseImportStmt parses the import statement — the form the templates (gadx)
// write —, lowered to the import expression:
//
//	@import "m"                       // import("m")
//	@import "m" as name               // name := import("m")
//	@import { a, b: c, d = 1 } from "m" // { a, b: c, d = 1 } := import("m")
//
// The names of the last form are those of a curly destructuring
// (ParseCurlyPattern): a key, a key bound to another name, a default. With
// ParseImportMain (the templates, gadx), `@import name from "m"` is
// `@import { main: name } from "m"`.
func (p *Parser) ParseImportStmt() node.Stmt {
	pos := p.Token.Pos
	p.Next()
	p.SkipSpace()

	var pattern *node.KeyValueArrayLit
	mainName := false
	switch {
	case p.Token.Token == token.Ident && p.mode.Has(ParseImportMain) && p.Token.Literal != "as":
		// `@import name from "m"`: the module's main as name
		mainName = true
		ident := p.ParseIdent()
		pattern = &node.KeyValueArrayLit{LParen: ident.Pos(), RParen: ident.End(), Curly: true,
			Elements: []node.Expr{&node.KeyValuePairLit{Key: node.EIdent("main", ident.Pos()), Value: ident, Colon: true}}}
		p.SkipSpace()
		if p.Token.Token != token.Ident || p.Token.Literal != "from" {
			p.ErrorExpected(p.Token.Pos, "'from'")
			p.advance(stmtStart)
			return &node.BadStmt{From: pos, To: p.Token.Pos}
		}
		p.Next()
		p.SkipSpace()
	case p.Token.Token == token.LBrace:
		pattern = p.ParseCurlyPattern()
		p.SkipSpace()
		if p.Token.Token != token.Ident || p.Token.Literal != "from" {
			p.ErrorExpected(p.Token.Pos, "'from'")
			p.advance(stmtStart)
			return &node.BadStmt{From: pos, To: p.Token.Pos}
		}
		p.Next()
		p.SkipSpace()
	}

	if p.Token.Token != token.String {
		p.ErrorExpected(p.Token.Pos, "module name")
		p.advance(stmtStart)
		return &node.BadStmt{From: pos, To: p.Token.Pos}
	}
	name, _ := p.ParseOperand().(*node.StrLit)
	if name == nil {
		p.ErrorExpected(p.Token.Pos, "module name")
		p.advance(stmtStart)
		return &node.BadStmt{From: pos, To: p.Token.Pos}
	}
	imp := &node.ImportExpr{Directive: true, MainName: mainName, CallExpr: node.CallExpr{
		Func: node.EIdent(token.Import.String(), pos),
		CallArgs: node.CallArgs{
			LParen: name.Pos(),
			RParen: name.End(),
			Args:   node.CallExprPositionalArgs{Values: []node.Expr{name}},
		},
	}}

	var stmt node.Stmt
	switch {
	case pattern != nil:
		stmt = &node.AssignStmt{LHS: []node.Expr{pattern}, RHS: []node.Expr{imp}, Token: token.Define, TokenPos: pos}
	case p.Token.Token == token.Ident && p.Token.Literal == "as":
		p.Next()
		ident := p.ParseIdent()
		stmt = &node.AssignStmt{LHS: []node.Expr{ident}, RHS: []node.Expr{imp}, Token: token.Define, TokenPos: pos}
	default:
		stmt = &node.ExprStmt{Expr: imp}
	}
	p.ExpectSemi()
	return stmt
}
