package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// isDeleteStmtStart reports whether the current `delete` identifier begins a
// delete statement — i.e. it is immediately followed by an operand (the target
// object). The lookahead keeps `delete` an ordinary identifier everywhere else
// (e.g. `obj.delete(k)` member calls, which do not start with `delete`).
func (p *Parser) isDeleteStmtStart() bool {
	if p.Token.Literal != "delete" {
		return false
	}
	var ok bool
	p.PeekCb(func(t PToken) bool {
		if t.IsSpace() {
			return true
		}
		// The target of a delete is always an identifier-rooted expression
		// (`env`, a variable, `this`, …) or a parenthesised expression.
		ok = t.Token == token.Ident || t.Token == token.LParen
		return false
	})
	return ok
}

// ParseDeleteStmt parses `delete Target[.field] [keys]`:
//
//	delete env.PATH            // selector form: delete the single key "PATH"
//	delete obj.field           // selector form: delete the single key "field"
//	delete env [a, b, *rest]   // keys form: delete each evaluated key
//	delete obj [k1, k2]        // keys form
//
// The target is parsed as an operand plus `.field` selectors, stopping at `[`;
// a following expression (typically an array literal) is the keys.
func (p *Parser) ParseDeleteStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "DeleteStmt"))
	}
	pos := p.Token.Pos
	p.Next() // consume `delete`

	target := p.parseDeleteTarget()

	var keys node.Expr
	if p.Token.Token == token.LBrack {
		keys = p.ParseExpr()
	}
	return &node.DeleteStmt{DeletePos: pos, Target: target, Keys: keys}
}

// parseDeleteTarget parses the delete target: an operand followed by any number
// of `.field` selectors. It deliberately stops at `[` so that a trailing
// `[keys]` array is parsed as the keys, not as an index of the target.
func (p *Parser) parseDeleteTarget() node.Expr {
	x := p.ParseOperand()
	for p.Token.Token == token.Period {
		p.Next()
		switch {
		case p.Token.Token == token.Ident || p.Token.Token == token.Else || p.Token.Token.IsKeyword():
			x = p.ParseSelector(x)
		default:
			p.ErrorExpected(p.Token.Pos, "selector")
			return x
		}
	}
	return x
}
