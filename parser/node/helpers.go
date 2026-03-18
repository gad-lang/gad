package node

import (
	"strings"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// IdentList represents a list of identifiers.
type IdentList struct {
	LParen  source.Pos
	VarArgs bool
	List    []*IdentExpr
	RParen  source.Pos
}

// Pos returns the position of first character belonging to the node.
func (n *IdentList) Pos() source.Pos {
	if n.LParen.IsValid() {
		return n.LParen
	}
	if len(n.List) > 0 {
		return n.List[0].Pos()
	}
	return source.NoPos
}

// End returns the position of first character immediately after the node.
func (n *IdentList) End() source.Pos {
	if n.RParen.IsValid() {
		return n.RParen + 1
	}
	if l := len(n.List); l > 0 {
		return n.List[l-1].End()
	}
	return source.NoPos
}

// NumFields returns the number of fields.
func (n *IdentList) NumFields() int {
	if n == nil {
		return 0
	}
	return len(n.List)
}

func (n *IdentList) String() string {
	var list []string
	for i, e := range n.List {
		if n.VarArgs && i == len(n.List)-1 {
			list = append(list, "..."+e.String())
		} else {
			list = append(list, e.String())
		}
	}
	return "(" + strings.Join(list, ", ") + ")"
}

type Token struct {
	Pos   source.Pos
	Token token.Token
}

func (t Token) Valid() bool {
	return t.Token != token.Illegal
}

type TokenLit struct {
	Pos     source.Pos
	Token   token.Token
	Literal string
}

func (t TokenLit) Precedence() int {
	return t.Token.Precedence()
}

func (t TokenLit) Is(other ...token.Token) bool {
	return t.Token.Is(other...)
}

func (t TokenLit) Valid() bool {
	return t.Token != token.Illegal || len(t.Literal) > 0
}

func (t TokenLit) String() string {
	if len(t.Literal) > 0 {
		return t.Literal
	}
	return t.Token.String()
}
