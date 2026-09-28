package gadx

import (
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
)

// An expression keeps its place in the source wherever it is written — the
// arguments of a component call, positional or named, and what a directive
// takes after its keyword (@for, @if, @match, @case, `~ x = …`) —: a call in it
// is where it is written, not at the start of a fragment parsed apart, nor at
// the directive's own `@`.
func TestExpressionPositions(t *testing.T) {
	src := "@main\n" +
		"    div\n" +
		"        +brand(;ariaLabel=t(\"brand\"), href=\"/\")\n" + // 3
		"        +alert(t(\"warn\"), 1)\n" + // 4
		"        @for $c in t(\"list\")\n" + // 5
		"            p {$c}\n" +
		"        @if t(\"cond\")\n" + // 7
		"            p yes\n" +
		"        ~ x := t(\"assign\")\n" + // 9
		"        @match t(\"tag\")\n" + // 10
		"            @case t(\"case\")\n" + // 11
		"                p c\n"
	file, err := gad.ParseSource("page.gadx", []byte(src), gad.SourceKindGadx)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][2]int{}
	node.Walk(file, func(n ast.Node) bool {
		if c, ok := n.(*node.CallExpr); ok {
			if id, ok := c.Func.(*node.IdentExpr); ok && id.Name == "t" {
				key := c.Args.Values[0].(*node.StrLit).Value()
				p := file.InputFile.Set().Position(c.Pos())
				got[key] = [2]int{p.Line, p.Column}
			}
		}
		return true
	})
	for key, want := range map[string][2]int{
		"brand":  {3, 27},
		"warn":   {4, 16},
		"list":   {5, 20},
		"cond":   {7, 13},
		"assign": {9, 16},
		"tag":    {10, 16},
		"case":   {11, 19},
	} {
		if got[key] != want {
			t.Errorf("t(%q) at %v, want %v", key, got[key], want)
		}
	}
}
