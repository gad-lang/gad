package parser

import (
	"fmt"
	"strings"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/utils"
	"github.com/gad-lang/gad/token"
)

type Token struct {
	Pos        source.Pos
	Token      token.Token
	Literal    string
	InsertSemi bool
	handled    bool
	Prev       []Token
	utils.Data
}

var _ fmt.Stringer = Token{}

func (t Token) String() string {
	return t.Token.String() + ": " + t.Literal
}

func (t *Token) LiteralRemoveLinePrefix(prefix string) {
	lines := strings.Split(t.Literal, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimPrefix(line, prefix)
	}
	t.Literal = strings.Join(lines, "\n")
}
