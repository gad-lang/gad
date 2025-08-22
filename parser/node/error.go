package node

import (
	"fmt"
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

func NewExpectedError(got Node, expected ...Node) *NodeError {
	var names = make([]string, len(expected))
	for i, e := range expected {
		names[i] = fmt.Sprintf("%T", e)
	}
	return &NodeError{
		Node: got,
		Err:  fmt.Sprintf("expected %s, but got %s (%[2]T)", strings.Join(names, "|"), got),
	}
}

func NewUnExpectedError(got Node) *NodeError {
	return &NodeError{
		Node: got,
		Err:  fmt.Sprintf("unexpected node %s (%[1]T)", got),
	}
}

type NodeError struct {
	Node Node
	Err  string
}

func (e *NodeError) Pos() source.Pos {
	return e.Node.Pos()
}

func (e *NodeError) Error() string {
	return e.Err
}
