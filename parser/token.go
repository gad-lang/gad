package parser

import (
	"fmt"
	"strings"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/utils"
	"github.com/gad-lang/gad/token"
)

type PToken struct {
	node.TokenLit
	InsertSemi bool
	handled    bool
	Prev       []PToken
	utils.Data
}

var _ fmt.Stringer = PToken{}

func (t PToken) IsSpace() bool {
	return t.Token == token.Semicolon && t.Literal == "\n"
}

func (t PToken) String() string {
	return t.Token.String() + ": " + t.Literal
}

func (t *PToken) LiteralRemoveLinePrefix(prefix string) {
	lines := strings.Split(t.Literal, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, prefix)
	}
	t.Literal = strings.Join(lines, "\n")
}
