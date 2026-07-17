package giom

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
			src:  "@global uri\n@main\n    <a href={uri}>see {uri}</a>\n",
			want: `<a href="/u">see /u</a>`,
		},
		{
			name: "self-closing element",
			src:  "@main\n    <img src=\"a.png\"/>\n",
			want: `<img src="a.png" />`,
		},
		{
			name: "void element",
			src:  "@main\n    <br>\n",
			want: `<br>`,
		},
		{
			name: "boolean attribute",
			src:  "@main\n    <input disabled>\n",
			want: `<input disabled>`,
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
			name: "multi-line attributes",
			src:  "@global cls\n@main\n    <div\n        class={cls}\n        id=\"main\"\n    >body</div>\n",
			want: `<div class="box" id="main">body</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGiom(t, tc.src, g))
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
