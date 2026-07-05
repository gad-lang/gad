package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// looksLikeCurlyDestructure reports whether the `{` at the current position
// begins a `{ … } := / = source` destructuring pattern rather than an ordinary
// block. It scans to the matching `}` (balancing braces/parens/brackets) and
// returns true only when an assignment operator (`:=` or `=`) immediately
// follows — a block can never be the left side of an assignment, so this is
// unambiguous.
func (p *Parser) looksLikeCurlyDestructure() bool {
	depth := 1 // the current `{` is already open
	afterBrace := false
	var result bool
	p.PeekCb(func(t PToken) bool {
		if afterBrace {
			if t.IsSpace() {
				return true
			}
			result = t.Token == token.Assign || t.Token == token.Define
			return false
		}
		switch t.Token {
		case token.LBrace, token.LParen, token.LBrack:
			depth++
		case token.RBrace, token.RParen, token.RBrack:
			if depth--; depth == 0 {
				afterBrace = true
			}
		case token.EOF:
			return false
		}
		return true
	})
	return result
}

// ParseCurlyDestructureStmt parses a TypeScript-style named-data destructuring
// statement:
//
//	{ key, key2: target, name = default, **rest } := / = source
//
// Entries use key-on-the-left (like TypeScript): `key` binds the value of key
// `key` to the variable `key`; `key2: target` binds the value of key `key2` to
// the variable `target`; `name = default` supplies a fallback when the key is
// absent; and a trailing `**rest` collects the remaining keys into a dict. It
// builds the same KeyValueArrayLit that the `(; … )` form produces (target:key
// order), so the existing dict-destructuring compiler handles it.
func (p *Parser) ParseCurlyDestructureStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "CurlyDestructureStmt"))
	}

	lbrace := p.Expect(token.LBrace)
	var elements []node.Expr
	for {
		p.SkipSpace()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF {
			break
		}
		if p.Token.Token == token.Pow { // `**rest`, must be last
			pos := p.Expect(token.Pow)
			p.SkipSpace()
			elements = append(elements, &node.NamedArgVarLit{TokenPos: pos, Value: p.ParseExpr()})
			p.SkipSpace()
			break
		}
		elements = append(elements, p.parseCurlyDestructureEntry())
		if !p.AtCommaOrNewLine("destructuring pattern", token.RBrace) {
			break
		}
		p.Next()
	}
	p.SkipSpace()
	rbrace := p.Expect(token.RBrace)
	kva := &node.KeyValueArrayLit{LParen: lbrace, RParen: rbrace, Elements: elements, Curly: true}

	p.SkipSpace()
	pos, tok := p.Token.Pos, p.Token.Token
	if tok != token.Assign && tok != token.Define {
		p.ErrorExpected(pos, "':=' or '='")
		return &node.BadStmt{From: lbrace, To: p.Token.Pos}
	}
	p.Next()
	return &node.AssignStmt{
		LHS:      []node.Expr{kva},
		RHS:      p.ParseExprList(),
		Token:    tok,
		TokenPos: pos,
	}
}

// parseCurlyDestructureEntry parses one `key` / `key: target` / `name = default`
// entry, producing a KeyValuePairLit in key-on-the-left (TypeScript) order: Key
// is the source key, and Value is the target variable (`:`) or the fallback
// default (`=`), or nil for the shorthand. The compiler reads this order when
// KeyValueArrayLit.Curly is set.
func (p *Parser) parseCurlyDestructureEntry() *node.KeyValuePairLit {
	key := p.ParseIdent()
	p.SkipSpace()
	switch p.Token.Token {
	case token.Colon: // `key: target` — bind key to a differently named variable
		p.Next()
		p.SkipSpace()
		return &node.KeyValuePairLit{Key: key, Value: p.ParseIdent(), Colon: true}
	case token.Assign: // `name = default`
		p.Next()
		p.SkipSpace()
		return &node.KeyValuePairLit{Key: key, Value: p.ParseExpr()}
	default: // `key` shorthand
		return &node.KeyValuePairLit{Key: key}
	}
}
