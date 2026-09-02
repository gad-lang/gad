package gadx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
)

// renderGadx writes src to a temp .gadx file and renders it.
func renderGadx(t *testing.T, src string, globals gad.Dict) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "t.gadx")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	out, err := renderString(r, p, globals)
	if err != nil {
		t.Fatalf("render: %v\nsrc:\n%s", err, src)
	}
	return out
}

// TestAttributeGroupRendering verifies that single, multi-value and multi-line
// attribute groups all render, and that separators inside values are ignored.
func TestAttributeGroupRendering(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "single",
			src:  "@main\n    div[class=\"a\"] hi\n",
			want: `<div class="a">hi</div>`,
		},
		{
			name: "multi-value comma",
			src:  "@main\n    a[href=\"/x\", title=\"go\"] link\n",
			want: `<a href="/x" title="go">link</a>`,
		},
		{
			name: "multi-line",
			src: "@main\n" +
				"    a[\n" +
				"        href=\"/x\"\n" +
				"        title=\"go\"\n" +
				"    ] link\n",
			want: `<a href="/x" title="go">link</a>`,
		},
		{
			name: "flag and expression",
			src:  "@main\n    input[type=\"text\", disabled, value=1+2]\n",
			want: `<input type="text" disabled value="3" />`,
		},
		{
			name: "comma inside value not split",
			src:  "@main\n    div[title=[1, 2][0], class=\"c\"] x\n",
			want: `<div title="1" class="c">x</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderGadx(t, tc.src, gad.Dict{})
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestClassAttributeForms exercises every accepted shape of the `class`
// attribute: plain string, static (dotted) tokens merged with a class attr,
// arrays (with falsy filtering), the JSX/Vue object form {name: condition}
// (truthy keys only, emitted sorted for determinism), and multiple class
// groups merging.
func TestClassAttributeForms(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		globals gad.Dict
		want    string
	}{
		{
			name: "string",
			src:  "@main\n    div[class=\"a\"] hi\n",
			want: `<div class="a">hi</div>`,
		},
		{
			name: "dotted merged with class attr",
			src:  "@main\n    div.a.b[class=\"c\"] hi\n",
			want: `<div class="a b c">hi</div>`,
		},
		{
			name: "array",
			src:  "@main\n    div[class=[\"x\", \"y\", \"z\"]] hi\n",
			want: `<div class="x y z">hi</div>`,
		},
		{
			name: "array with falsy filtered",
			src:  "@main\n    div[class=[nil, \"base\", false, \"\"]] hi\n",
			want: `<div class="base">hi</div>`,
		},
		{
			name: "dict truthy keys only",
			src:  "@main\n    div[class={active: true, off: false}] hi\n",
			want: `<div class="active">hi</div>`,
		},
		{
			name: "dict sorted for determinism",
			src:  "@main\n    div[class={zeta: true, alpha: true, beta: false}] hi\n",
			want: `<div class="alpha zeta">hi</div>`,
		},
		{
			name:    "dict driven by globals",
			src:     "@global on\n@global off\n@main\n    div[class={active: on, muted: off}] hi\n",
			globals: gad.Dict{"on": gad.True, "off": gad.False},
			want:    `<div class="active">hi</div>`,
		},
		{
			name: "static tokens plus dict",
			src:  "@main\n    div.base[class={active: true}] hi\n",
			want: `<div class="base active">hi</div>`,
		},
		{
			name: "two class groups merge",
			src:  "@main\n    div[class=[\"p\"]][class=[\"q\"]] hi\n",
			want: `<div class="p q">hi</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			globals := tc.globals
			if globals == nil {
				globals = gad.Dict{}
			}
			got := renderGadx(t, tc.src, globals)
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestBooleanAttributeForms verifies attribute presence vs. value semantics:
// the flag type (yes/no, and a valueless attribute) controls presence and
// renders bare, while a bool (true/false) always renders its literal value.
func TestBooleanAttributeForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "valueless is bare flag",
			src:  "@main\n    div[data-value] x\n",
			want: `<div data-value>x</div>`,
		},
		{
			name: "flag yes is bare",
			src:  "@main\n    div[data-value=yes] x\n",
			want: `<div data-value>x</div>`,
		},
		{
			name: "flag no is omitted",
			src:  "@main\n    div[data-value=no] x\n",
			want: `<div>x</div>`,
		},
		{
			name: "bool true renders value",
			src:  "@main\n    div[data-value=true] x\n",
			want: `<div data-value="true">x</div>`,
		},
		{
			name: "bool false renders value",
			src:  "@main\n    div[data-value=false] x\n",
			want: `<div data-value="false">x</div>`,
		},
		{
			name: "number renders value",
			src:  "@main\n    div[data-value=1] x\n",
			want: `<div data-value="1">x</div>`,
		},
		{
			name: "nil is omitted",
			src:  "@main\n    div[data-value=nil] x\n",
			want: `<div>x</div>`,
		},
		{
			name: "empty string is omitted",
			src:  "@main\n    div[data-value=\"\"] x\n",
			want: `<div>x</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderGadx(t, tc.src, gad.Dict{})
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestClassAttrsBuiltinDict verifies the gadx.attrs builtin (the programmatic
// attribute formatter, distinct from the render-tree path) accepts the JSX/Vue
// dict form for class, keeping truthy keys in sorted order.
func TestClassAttrsBuiltinDict(t *testing.T) {
	src := "@main\n" +
		"    div{= gadx.attrs(; class={active: true, off: false, base: true}) }\n"
	got := renderGadx(t, src, gad.Dict{})
	want := `<div> class="active base"</div>`
	if got != want {
		t.Fatalf("render mismatch\n got: %s\nwant: %s", got, want)
	}
}

// TestAttributeExpressionPosition verifies a runtime error inside a multi-line
// attribute value reports the correct source line and column.
func TestAttributeExpressionPosition(t *testing.T) {
	src := "@global bad\n" +
		"@main\n" +
		"    div[\n" +
		"        class=\"a\"\n" +
		"        title=bad()\n" + // line 5, col 18 (the call `(`)
		"    ] hi\n"
	re := runForError(t, src)
	line, col := firstTraceLineCol(re)
	if line != 5 || col != 18 {
		t.Fatalf("attribute nil-call resolved to %d:%d, want 5:18\ntrace:\n%+v", line, col, re.StackTrace())
	}
}
