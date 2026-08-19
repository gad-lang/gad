package node

import "testing"

func TestInspectVisitsAllNodes(t *testing.T) {
	inner := &ArrayExpr{Elements: []Expr{&IdentExpr{Name: "b"}, &IdentExpr{Name: "c"}}}
	root := &ArrayExpr{Elements: []Expr{&IdentExpr{Name: "a"}, inner}}

	var count int
	Inspect(root, func(Node) bool { count++; return true })
	// root array + a + inner array + b + c
	if count != 5 {
		t.Fatalf("visited %d nodes, want 5", count)
	}

	// Returning false prunes children: only the root is visited.
	count = 0
	Inspect(root, func(Node) bool { count++; return false })
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

func TestInspectNilSafe(t *testing.T) {
	Inspect(nil, func(Node) bool { t.Fatal("visitor called for nil root"); return true })
	// A struct with a nil interface field must not panic.
	Inspect(&ArrayExpr{Elements: []Expr{nil, &IdentExpr{Name: "a"}}}, func(Node) bool { return true })
}
