package giom

import (
	"testing"

	"github.com/gad-lang/gad"
)

// TestSlotProgrammatic covers passing slots to a component from gad source
// (via the `slots` named argument) instead of the `@slot #name` sugar.
//
// A component compiles to a gad function taking a `slots` dict; each entry is a
// slot function whose first parameter is `super` (the component's default for
// that slot), followed by the slot's scope parameters. The `@slot #name` /
// `+super` template sugar just builds this dict and forwards super — done by
// hand here, a raw call must pass super's own super explicitly.
func TestSlotProgrammatic(t *testing.T) {
	const box = "@export comp box(;slots={})\n" +
		"    div\n" +
		"        @slot main\n" +
		"            span default\n"

	const list = "@export comp list(items;slots={})\n" +
		"    ul\n" +
		"        @for i, it in items\n" +
		"            li\n" +
		"                @slot row(i, it)\n" +
		"                    {=i}: {=it}\n"

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			// Pass the `main` slot and render the default after it via super.
			// A raw super() call must supply super's own super (an empty fn).
			name: "main slot with super",
			src: box + "@main\n" +
				"    ~ tag += box(; slots={main: func(super) { tag := giom.Tag(nil); giom.Text(tag, raw \"<b>hi</b>\"); tag += super(func(*_){}); return tag }})\n",
			want: `<div><b>hi</b><span>default</span></div>`,
		},
		{
			// A scoped slot: super is first, then the slot's (i, it) scope.
			name: "scoped slot ignoring super",
			src: list + "@main\n" +
				"    ~ tag += list([\"a\", \"b\"]; slots={row: func(super, i, it) { tag := giom.Tag(nil); giom.Text(tag, raw \"<b>\"+it+\"</b>\"); return tag }})\n",
			want: `<ul><li><b>a</b></li><li><b>b</b></li></ul>`,
		},
		{
			// Forward the scope to super: super(super_of_super, i, it).
			name: "scoped slot forwarding super",
			src: list + "@main\n" +
				"    ~ tag += list([\"a\", \"b\"]; slots={row: func(super, i, it) { tag := giom.Tag(nil); giom.Text(tag, raw \"* \"); tag += super(func(*_){}, i, it); return tag }})\n",
			want: `<ul><li>* 0: a</li><li>* 1: b</li></ul>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderGiom(t, tc.src, gad.Dict{})
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
