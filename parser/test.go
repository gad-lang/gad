package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// isTestStmtStart reports whether the current `test`/`bench` identifier begins a
// test/bench statement — i.e. it is immediately followed by a NAME (identifier
// or string literal) and then a `{`. The lookahead keeps `test`/`bench` as
// ordinary identifiers everywhere else (e.g. `test := import("test")`).
func (p *Parser) isTestStmtStart() bool {
	if lit := p.Token.Literal; lit != "test" && lit != "bench" {
		return false
	}
	var (
		seen int
		ok   bool
	)
	p.PeekCb(func(t PToken) bool {
		if t.IsSpace() {
			return true
		}
		seen++
		switch seen {
		case 1:
			// NAME must be an identifier or a string literal.
			return t.Token == token.Ident || t.Token == token.String
		default:
			ok = t.Token == token.LBrace
			return false
		}
	})
	return ok
}

// ParseTestStmt parses `test NAME { … }` / `bench NAME { … }`. NAME is an
// identifier or a string literal; the body is a block in which the injected `t`
// test context is available. A preceding doc comment is attached.
func (p *Parser) ParseTestStmt() node.Stmt {
	if p.Trace {
		defer untracep(tracep(p, "TestStmt"))
	}
	doc := p.leadComment

	s := &node.TestStmt{Doc: doc, KwPos: p.Token.Pos}
	if p.Token.Literal == "bench" {
		s.Kind = node.TestKindBench
	}
	p.Next()
	p.SkipSpace()

	s.NamePos = p.Token.Pos
	switch p.Token.Token {
	case token.Ident:
		s.Name = p.Token.Literal
		p.Next()
	case token.String:
		s.Name = p.ParseStrLit().Value()
		s.Quoted = true
	default:
		p.ErrorExpected(p.Token.Pos, "test name (identifier or string)")
		return &node.BadStmt{From: s.KwPos, To: p.Token.Pos}
	}

	p.SkipSpace()
	s.Body = p.ParseScopedBlockStmt()
	return s
}
