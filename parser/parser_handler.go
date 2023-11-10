package parser

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

type ParseListHandler func(start token.Token, ends ...BlockWrap) (list []node.Stmt, end *BlockEnd)
