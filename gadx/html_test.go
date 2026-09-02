package gadx

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// TestHtmlRegions covers the raw-HTML region syntax: literal and interpolated
// attributes (value and name), text interpolation, self-closing/void elements,
// `<>…</>` fragments, nested elements and whitespace collapsing.
func TestHtmlRegions(t *testing.T) {
	g := gad.Dict{
		"uri":  gad.Str("/u"),
		"key":  gad.Str("id"),
		"val":  gad.Str("x1"),
		"name": gad.Str("<b>Ann</b>"),
		"cls":  gad.Str("box"),
	}
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "literal attrs and text",
			src:  "@main\n    <a href=\"/x\" title=\"hi\">hello</a>\n",
			want: `<a href="/x" title="hi">hello</a>`,
		},
		{
			name: "interpolated attribute value",
			src:  "@global uri\n@main\n    <a href={uri}>go</a>\n",
			want: `<a href="/u">go</a>`,
		},
		{
			name: "interpolated attribute name and value",
			src:  "@global key\n@global val\n@main\n    <div data-{key}={val}>x</div>\n",
			want: `<div data-id="x1">x</div>`,
		},
		{
			name: "text interpolation",
			src:  "@global uri\n@main\n    <a href={uri}>see {=uri}</a>\n",
			want: `<a href="/u">see /u</a>`,
		},
		{
			name: "self-closing element",
			src:  "@main\n    <img src=\"a.png\"/>\n",
			want: `<img src="a.png" />`,
		},
		{
			// The HTML region compiles to a gadx.Tag, so a void element renders
			// self-closed (`<br />`) like the equivalent pug-style `br`.
			name: "void element",
			src:  "@main\n    <br>\n",
			want: `<br />`,
		},
		{
			// A valueless attribute is the flag `yes`, rendered as a bare
			// boolean attribute (no value); `input` is void, so it self-closes.
			name: "boolean attribute",
			src:  "@main\n    <input disabled>\n",
			want: `<input disabled />`,
		},
		{
			name: "nested elements collapse whitespace",
			src:  "@main\n    <ul>\n        <li>a</li>\n        <li>b</li>\n    </ul>\n",
			want: `<ul> <li>a</li> <li>b</li> </ul>`,
		},
		{
			name: "fragment produces no wrapper",
			src:  "@main\n    <><span>a</span><span>b</span></>\n",
			want: `<span>a</span><span>b</span>`,
		},
		{
			// gadx.Tag classifies attributes (regular first, then the class list),
			// so `class`/`id` render in that order regardless of source order.
			name: "multi-line attributes",
			src:  "@global cls\n@main\n    <div\n        class={cls}\n        id=\"main\"\n    >body</div>\n",
			want: `<div id="main" class="box">body</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGadx(t, tc.src, g))
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestFragmentLowersToElements verifies a `<>…</>` fragment lowers to a real
// gadx.Elements() node (not a nameless tag): its children build into the
// fragment, which is then spliced into the enclosing parent.
func TestFragmentLowersToElements(t *testing.T) {
	src := "@main\n    <><span>a</span><span>b</span></>\n"
	out, err := gad.TranspileGadxSource("frag.gadx", []byte(src))
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	code := string(out)
	if !strings.Contains(code, "gadx.Elements()") {
		t.Fatalf("fragment did not lower to gadx.Elements():\n%s", code)
	}
	// It must not resurrect a nameless tag for the fragment.
	if strings.Contains(code, `gadx.Tag(`+"\n\t\t\t\ttag\n\t\t\t\t\"\"") {
		t.Fatalf("fragment lowered to a nameless gadx.Tag:\n%s", code)
	}
	// And it still renders as its children with no wrapper.
	if got := strings.TrimSpace(renderGadx(t, src, gad.Dict{})); got != "<span>a</span><span>b</span>" {
		t.Fatalf("render = %q", got)
	}
}

// TestHtmlInterleave covers block-level gadx statements (@if/@for/@else)
// interleaved inside an HTML region by indentation: the directive line and its
// more-indented body (which may itself contain HTML) render as children of the
// enclosing element, in source order alongside sibling HTML.
func TestHtmlInterleave(t *testing.T) {
	g := gad.Dict{
		"items": gad.Array{gad.Str("a"), gad.Str("b")},
		"on":    gad.True,
	}
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "if inside element",
			src:  "@main\n    <div>\n        @if 1\n            <hr>\n    </div>\n",
			want: `<div><hr /></div>`,
		},
		{
			name: "for inside element",
			src:  "@global items\n@main\n    <ul>\n        @for x in items\n            <li>{=x}</li>\n    </ul>\n",
			want: `<ul><li>a</li><li>b</li></ul>`,
		},
		{
			name: "if/else inside element",
			src:  "@global on\n@main\n    <div>\n        @if on\n            <b>Y</b>\n        @else\n            <i>N</i>\n    </div>\n",
			want: `<div><b>Y</b></div>`,
		},
		{
			// The @for block renders between sibling HTML in source order.
			name: "interleaved with sibling HTML",
			src:  "@global items\n@main\n    <ul><li>head</li>\n        @for x in items\n            <li>{=x}</li>\n    </ul>\n",
			want: `<ul><li>head</li><li>a</li><li>b</li></ul>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGadx(t, tc.src, g))
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestHtmlInterpolationPosition verifies a nil-call inside an HTML interpolation
// reports the correct source line and column.
func TestHtmlInterpolationPosition(t *testing.T) {
	tests := []struct {
		name              string
		src               string
		wantLine, wantCol int
	}{
		{
			// line 3, col 17: the `(` of bad() in the attribute value
			name:     "attribute value",
			src:      "@global bad\n@main\n    <a href={bad()}>x</a>\n",
			wantLine: 3,
			wantCol:  17,
		},
		{
			// line 3, col 12: bad() in text content
			name:     "text content",
			src:      "@global bad\n@main\n    <p>{bad()}</p>\n",
			wantLine: 3,
			wantCol:  12,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			re := runForError(t, tc.src)
			if !strings.Contains(re.Error(), "NotCallableError") {
				t.Fatalf("expected NotCallableError, got: %v", re.Error())
			}
			line, col := firstTraceLineCol(re)
			if line != tc.wantLine || col != tc.wantCol {
				t.Fatalf("position = %d:%d, want %d:%d\ntrace:\n%+v",
					line, col, tc.wantLine, tc.wantCol, re.StackTrace())
			}
		})
	}
}
