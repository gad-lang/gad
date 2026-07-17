package giom

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// TestSlotDynamicName covers interpolated slot names: `@slot (name[{expr}])`
// (declaration) and `@slot #(name[{expr}])` (pass). The `( … )` content is a Gad
// template string, so `{expr}` is evaluated at render time and used as the
// `slots[…]` key. Also covers hoisting call-block `~` code to call scope.
func TestSlotDynamicName(t *testing.T) {
	// A component whose per-iteration slot name is computed from the loop index,
	// giving each row its own overridable slot.
	const list = "@comp list(items;slots={})\n" +
		"\t@for i, it in items\n" +
		"\t\t@slot (item[{i}])(it)\n" +
		"\t\t\t| {=it};\n"

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			// No overrides: every row renders its default.
			name: "declaration defaults",
			src:  list + "@main\n\t+list([\"a\", \"b\", \"c\"])\n",
			want: "a;b;c;",
		},
		{
			// Override a single row by its interpolated name.
			name: "pass overrides one row",
			src: list + "@main\n" +
				"\t+list([\"a\", \"b\", \"c\"])\n" +
				"\t\t@slot #(item[{1}])(super, it)\n" +
				"\t\t\t| X{=it};\n",
			want: "a;Xb;c;",
		},
		{
			// The override renders the row's default via super, forwarding the
			// scope with the `+super(it)` sugar.
			name: "override forwards super",
			src: list + "@main\n" +
				"\t+list([\"a\", \"b\", \"c\"])\n" +
				"\t\t@slot #(item[{2}])(super, it)\n" +
				"\t\t\t| *\n" +
				"\t\t\t+super(it)\n",
			want: "a;b;*c;",
		},
		{
			// Call-block `~` code is hoisted to call scope, so an interpolated
			// slot name may reference it.
			name: "init code before slot pass feeds the name",
			src: list + "@main\n" +
				"\t+list([\"a\", \"b\", \"c\"])\n" +
				"\t\t~ const target = 1\n" +
				"\t\t@slot #(item[{target}])(super, it)\n" +
				"\t\t\t| Y{=it};\n",
			want: "a;Yb;c;",
		},
		{
			// Multiple interleaved `~` code statements are all hoisted; a later
			// slot body references one of them.
			name: "multiple init code statements",
			src: list + "@main\n" +
				"\t+list([\"a\", \"b\", \"c\"])\n" +
				"\t\t~ const target = 0\n" +
				"\t\t@slot #(item[{target}])(super, it)\n" +
				"\t\t\t| Z{=it};\n" +
				"\t\t~ var mark = \"!\"\n" +
				"\t\t@slot #(item[{2}])(super, it)\n" +
				"\t\t\t| {=mark}{=it};\n",
			want: "Za;b;!c;",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGiom(t, tc.src, gad.Dict{}))
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}

// TestSlotDynamicNamePositions verifies that a nil-call inside a slot name's
// `{ … }` interpolation reports the correct source line — i.e. the interpolated
// name preserves source positions.
func TestSlotDynamicNamePositions(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantLine int
		wantCol  int
	}{
		{
			// line 3, col 16: the `{z()}` in a slot declaration name
			name: "declaration name",
			src: "@global z\n" +
				"@comp c(;slots={})\n" +
				"    @slot (k[{z()}])\n" +
				"        | x\n" +
				"@main\n" +
				"    +c\n",
			wantLine: 3,
			wantCol:  16,
		},
		{
			// line 7, col 21: the `{z()}` in a slot pass name
			name: "pass name",
			src: "@global z\n" +
				"@comp c(;slots={})\n" +
				"    @slot k\n" +
				"        | x\n" +
				"@main\n" +
				"    +c\n" +
				"        @slot #(k[{z()}])\n" +
				"            | y\n",
			wantLine: 7,
			wantCol:  21,
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
				t.Fatalf("stack trace position = %d:%d, want %d:%d\ntrace:\n%+v",
					line, col, tc.wantLine, tc.wantCol, re.StackTrace())
			}
		})
	}
}
