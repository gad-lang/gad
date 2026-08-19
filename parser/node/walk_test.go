package node

import (
	"testing"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

func TestWalkVisitsAllNodes(t *testing.T) {
	inner := &ArrayExpr{Elements: []Expr{&IdentExpr{Name: "b"}, &IdentExpr{Name: "c"}}}
	root := &ArrayExpr{Elements: []Expr{&IdentExpr{Name: "a"}, inner}}

	var count int
	Walk(root, func(ast.Node) bool { count++; return true })
	// root array + a + inner array + b + c
	if count != 5 {
		t.Fatalf("visited %d nodes, want 5", count)
	}

	// Returning false prunes children: only the root is visited.
	count = 0
	Walk(root, func(ast.Node) bool { count++; return false })
	if count != 1 {
		t.Fatalf("with prune, visited %d nodes, want 1", count)
	}
}

func TestIdentNames(t *testing.T) {
	inner := &ArrayExpr{Elements: []Expr{&IdentExpr{Name: "b"}, &IdentExpr{Name: "c"}}}
	root := &ArrayExpr{Elements: []Expr{&IdentExpr{Name: "a"}, inner, &IdentExpr{Empty: true}}}

	got := IdentNames(root)
	for _, n := range []string{"a", "b", "c"} {
		if _, ok := got[n]; !ok {
			t.Errorf("IdentNames missing %q; got %v", n, got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("IdentNames = %v, want exactly {a,b,c} (empty ident excluded)", got)
	}
}

func TestWalkNilSafe(t *testing.T) {
	Walk(nil, func(ast.Node) bool { t.Fatal("visitor called for nil root"); return true })
	// A struct with a nil interface field must not panic.
	Walk(&ArrayExpr{Elements: []Expr{nil, &IdentExpr{Name: "a"}}}, func(ast.Node) bool { return true })
}

// customNode is a minimal ast.Node implemented outside the node package (as the
// gadx AST is): it must still be visited and descended into.
type customNode struct {
	ast.NodeData
	Child Expr
}

func (customNode) Pos() source.Pos { return 0 }
func (customNode) End() source.Pos { return 0 }
func (customNode) String() string  { return "customNode" }

func TestWalkCustomNode(t *testing.T) {
	root := &customNode{Child: &IdentExpr{Name: "z"}}
	var sawCustom, sawChild bool
	Walk(root, func(n ast.Node) bool {
		switch v := n.(type) {
		case *customNode:
			sawCustom = true
		case *IdentExpr:
			if v.Name == "z" {
				sawChild = true
			}
		}
		return true
	})
	if !sawCustom {
		t.Error("custom node was not visited")
	}
	if !sawChild {
		t.Error("child of custom node was not visited")
	}
}
