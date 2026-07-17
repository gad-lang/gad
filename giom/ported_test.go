package giom

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// This file ports the behavioural tests from the upstream giom v1 suite
// (run_test.go / giom_test.go at cb3a3d9) to the current API and syntax:
// interpolation is `{ … }` / `{= … }` (was `#{ … }`), and rendering goes through
// Render. Cases exercising features that changed in the current engine
// (`+@fn` self-recursion, `@switch`/`@case`, raw-text `script`/`style`) are
// intentionally omitted.

// portRun renders tpl (written to a temp .giom file), with optional extra .giom
// modules and globals, returning the trimmed output.
func portRun(t *testing.T, tpl string, globals gad.Dict, modules map[string]string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	for name, src := range modules {
		if err := os.WriteFile(filepath.Join(dir, name+".giom"), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}
	p := filepath.Join(dir, "t.giom")
	if err := os.WriteFile(p, []byte(tpl), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	var buf bytes.Buffer
	err := r.Render(&buf, p, globals)
	return strings.TrimSpace(buf.String()), err
}

// portExpect renders tpl and asserts the trimmed output equals expected.
func portExpect(t *testing.T, tpl, expected string, globals gad.Dict) {
	t.Helper()
	got, err := portRun(t, tpl, globals, nil)
	require.NoErrorf(t, err, "render:\n%s", tpl)
	require.Equal(t, expected, got)
}

func TestPorted_Doctype(t *testing.T) {
	portExpect(t, "!!! 5", "<!DOCTYPE html>", nil)
}

func TestPorted_Nesting(t *testing.T) {
	portExpect(t,
		"@main\n    html\n        head\n            title\n        body\n",
		"<html><head><title></title></head><body></body></html>", nil)
}

func TestPorted_Empty(t *testing.T) {
	portExpect(t, "", "", nil)
}

func TestPorted_Comp(t *testing.T) {
	portExpect(t,
		"@comp a(a)\n    p {= a }\n@main\n    +a(1)\n",
		"<p>1</p>", nil)
}

func TestPorted_CompNoArguments(t *testing.T) {
	portExpect(t,
		"@comp a()\n    p Testing\n@main\n    +a\n",
		"<p>Testing</p>", nil)
}

func TestPorted_CompMultiArguments(t *testing.T) {
	portExpect(t,
		"@comp a($a, $b, $c, $d)\n    p {$a} {$b} {$c} {$d}\n@main\n    +a(\"a\", \"b\", \"c\", 2)\n",
		"<p>a b c 2</p>", nil)
}

func TestPorted_CompNameWithDashes(t *testing.T) {
	portExpect(t,
		"@comp i-am-mixin($a, $b)\n    p {$a} {$b}\n@main\n    +i-am-mixin(\"a\", \"b\")\n",
		"<p>a b</p>", nil)
}

func TestPorted_CompUnknown(t *testing.T) {
	_, err := portRun(t, "@main\n    +bar(1)\n", nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unresolved reference "bar"`)
}

func TestPorted_Id(t *testing.T) {
	portExpect(t,
		"@main\n    div#test\n        p#test1#test2\n",
		`<div id="test"><p id="test2"></p></div>`, nil)
}

func TestPorted_ArithmeticExpression(t *testing.T) {
	portExpect(t, "@main\n    {A + B * C}\n",
		"14", gad.Dict{"A": gad.Int(2), "B": gad.Int(3), "C": gad.Int(4)})
}

func TestPorted_BooleanExpression(t *testing.T) {
	portExpect(t, "@main\n    {C - A < B}\n",
		"true", gad.Dict{"A": gad.Int(2), "B": gad.Int(3), "C": gad.Int(4)})
}

func TestPorted_StructMethodCall(t *testing.T) {
	d := rfDummy{X: "Hello"}
	rv, err := gad.NewReflectValue(d)
	require.NoError(t, err)
	portExpect(t, `@main`+"\n    "+`{ $.MethodWithArg("world") }`+"\n",
		"Hello world", gad.Dict{"$": rv})
}

func TestPorted_DollarInTagAttributes(t *testing.T) {
	portExpect(t, `@main`+"\n    "+`input[placeholder="$ per " + kwh]`+"\n",
		`<input placeholder="$ per kWh" />`, gad.Dict{"kwh": gad.Str("kWh")})
}

func TestPorted_TableComp(t *testing.T) {
	portExpect(t,
		"@comp table(rows, header=nil)\n"+
			"    @slot body(rows)\n"+
			"        tbody\n"+
			"            @for row in rows\n"+
			"                tr\n"+
			"                    @for cel in row\n"+
			"                        td {= cel }\n"+
			"@main\n"+
			"    +table([[1,2]])\n",
		"<tbody><tr><td>1</td><td>2</td></tr></tbody>", nil)
}

func TestPorted_CompOverrideMainSlot(t *testing.T) {
	// caller replaces the slot content
	portExpect(t,
		"@comp message()\n    @slot main\n        | the message\n@main\n    +message\n        | my msg\n",
		"my msg", nil)

	// caller wraps the default via the auto-injected super
	portExpect(t,
		"@comp message()\n"+
			"    @slot main\n"+
			"        | default message\n"+
			"@main\n"+
			"    +message\n"+
			"        @slot #main\n"+
			"            {= \"my msg - \" }\n"+
			"            +super\n",
		"my msg - default message", nil)
}

// TestPorted_Match ports the upstream @switch/@case/@default test to the current
// @match/@case/@else syntax (the default clause is `@else`).
func TestPorted_Match(t *testing.T) {
	tpl := "@main\n" +
		"    @match a\n" +
		"        @case 1\n" +
		"            | v1\n" +
		"        @case 2\n" +
		"            | v2\n" +
		"        @else\n" +
		"            | v0\n"
	portExpect(t, tpl, "v1", gad.Dict{"a": gad.Int(1)})
	portExpect(t, tpl, "v2", gad.Dict{"a": gad.Int(2)})
	portExpect(t, tpl, "v0", gad.Dict{"a": gad.Int(9)})
}

func TestPorted_RunErrorTrace(t *testing.T) {
	// an unresolved comp call surfaces as a compile error at the call site
	_, err := portRun(t, "@main\n    +b\n", nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unresolved reference "b"`)
}

// rfDummy backs TestPorted_StructMethodCall (a reflected Go value with a method).
type rfDummy struct{ X string }

func (d rfDummy) MethodWithArg(s string) string { return d.X + " " + s }
