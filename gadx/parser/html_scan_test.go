package parser

import "testing"

// TestHtmlRegionEnd covers the balanced HTML region scanner: nested tags,
// self-closing and void elements, fragments, and `<`/`>` hidden inside quoted
// attribute values and `{ … }` interpolations.
func TestHtmlRegionEnd(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int // expected end offset; -1 means "not ok"
	}{
		{"simple", "<a>x</a>", 8},
		{"nested", "<div><span>x</span></div>", 25},
		{"self-closing", "<br/>", 5},
		{"self-closing space", "<img />", 7},
		{"void element", "<br>", 4},
		{"fragment", "<><b>a</b></>", 13},
		{"gt inside attr string", "<a title=\"a > b\">x</a>", 22},
		{"lt inside interpolation", "<a href={x < y ? p : q}>x</a>", 29},
		{"trailing content", "<a>x</a> tail", 8},
		{"unbalanced", "<a>x", -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			end, ok := htmlRegionEnd(tc.src, 0)
			if tc.want == -1 {
				if ok {
					t.Fatalf("expected not ok, got end=%d", end)
				}
				return
			}
			if !ok || end != tc.want {
				t.Fatalf("htmlRegionEnd = (%d, %v), want (%d, true)", end, ok, tc.want)
			}
		})
	}
}

func TestScanOpenTagEnd(t *testing.T) {
	tests := []struct {
		src           string
		wantEnd       int
		wantSelfClose bool
		wantName      string
	}{
		{"<a>", 3, false, "a"},
		{"<br/>", 5, true, "br"},
		{"<img />rest", 7, true, "img"},
		{`<a href="x>y">`, 14, false, "a"},
		{"<div data-{k}={v}>", 18, false, "div"},
	}
	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			end, self, name := scanOpenTagEnd(tc.src, 0)
			if tc.wantEnd == -1 {
				if end != -1 {
					t.Fatalf("expected end=-1, got %d", end)
				}
				return
			}
			if end != tc.wantEnd || self != tc.wantSelfClose || name != tc.wantName {
				t.Fatalf("scanOpenTagEnd = (%d, %v, %q), want (%d, %v, %q)",
					end, self, name, tc.wantEnd, tc.wantSelfClose, tc.wantName)
			}
		})
	}
}
