package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// isContractDeclStart reports whether the current `contract` identifier
// begins a declaration — `contract NAME {` — rather than being a name.
func (p *Parser) isContractDeclStart() bool {
	if p.Token.Token != token.Ident || p.Token.Literal != "contract" || p.Peek().Token != token.Ident {
		return false
	}
	return p.PeekC(2)[1].Token == token.LBrace
}

// isContractExprStart reports whether the current `contract` identifier
// begins an anonymous contract — `contract {` — at an operand position.
func (p *Parser) isContractExprStart() bool {
	return p.Token.Token == token.Ident && p.Token.Literal == "contract" && p.Peek().Token == token.LBrace
}

// parseContractStmt parses `contract NAME { *P; target T; meta { … } }`; the
// current token is the contextual `contract`.
func (p *Parser) parseContractStmt() *node.ContractStmt {
	return &node.ContractStmt{ContractExpr: *p.parseContractExpr(true)}
}

// parseContractExpr parses a contract, named (`contract NAME { … }`) or not
// (`contract { … }`, a value); the current token is the contextual `contract`.
func (p *Parser) parseContractExpr(named bool) *node.ContractExpr {
	if p.Trace {
		defer untracep(tracep(p, "ContractExpr"))
	}
	s := &node.ContractExpr{ContractPos: p.Token.Pos, Doc: p.leadComment, Meta: p.takeMeta()}
	p.Next() // consume `contract`
	p.SkipSpace()
	if named {
		s.NameExpr = p.ParseIdent()
		p.SkipSpace()
	}
	s.LBrace = p.Expect(token.LBrace)
	p.ExprLevel++
	for {
		p.skipClassSeps()
		if p.Token.Token == token.RBrace || p.Token.Token == token.EOF || p.Failed() {
			break
		}
		switch {
		case p.Token.Token == token.Mul:
			// `*Parent`
			p.Next()
			p.SkipSpace()
			if e := p.ParsePrimaryExpr(); e != nil {
				s.Parents = append(s.Parents, e)
			}
		case p.Token.Token == token.Ident && p.Token.Literal == "target":
			s.TargetPos = p.Token.Pos
			p.Next()
			p.SkipSpace()
			s.Target = p.ParseTypes()
			if len(s.Target) == 0 {
				p.Error(p.Token.Pos, "expected the types of the target after `target`")
			}
		case p.Token.Token == token.Ident && p.Token.Literal == "meta" && p.Peek().Token == token.LBrace:
			p.Next() // consume `meta`
			s.MetaIface = p.parseInterfaceBody(PToken{}, nil)
		default:
			p.Error(p.Token.Pos, "expected `*Parent`, `target …` or `meta { … }` in a contract")
			p.ExprLevel--
			return s
		}
	}
	p.ExprLevel--
	s.RBrace = p.Expect(token.RBrace)
	return s
}
