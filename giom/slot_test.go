package giom

import (
	"testing"

	"github.com/gad-lang/gad"
)

// TestSlotRendering covers slot compilation: default content, overriding,
// optional (no-default) slots that render only when provided, scoped slots that
// pass data, and `$super` used to render the default from an override.
func TestSlotRendering(t *testing.T) {
	box := "@export comp box(;slots={})\n" +
		"    div\n" +
		"        @slot main\n" +
		"            span default\n"

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "default content",
			src:  box + "@main\n    +box\n",
			want: `<div><span>default</span></div>`,
		},
		{
			name: "override",
			src: box + "@main\n" +
				"    +box\n" +
				"        @slot #main\n" +
				"            b over\n",
			want: `<div><b>over</b></div>`,
		},
		{
			name: "optional slot not provided renders nothing",
			src: "@export comp box(;slots={})\n" +
				"    div\n" +
				"        @slot extra\n" +
				"@main\n    +box\n",
			want: `<div></div>`,
		},
		{
			name: "optional slot provided",
			src: "@export comp box(;slots={})\n" +
				"    div\n" +
				"        @slot extra\n" +
				"@main\n" +
				"    +box\n" +
				"        @slot #extra\n" +
				"            i hi\n",
			want: `<div><i>hi</i></div>`,
		},
		{
			name: "super renders default from override",
			src: box + "@main\n" +
				"    +box\n" +
				"        @slot #main\n" +
				"            b before\n" +
				"            +super\n",
			want: `<div><b>before</b><span>default</span></div>`,
		},
		{
			name: "super with explicit first param not double-injected",
			src: box + "@main\n" +
				"    +box\n" +
				"        @slot #main(super)\n" +
				"            b before\n" +
				"            +super\n" +
				"            b after\n",
			want: `<div><b>before</b><span>default</span><b>after</b></div>`,
		},
		{
			name: "optional slot override may call empty super safely",
			src: "@export comp box(;slots={})\n" +
				"    div\n" +
				"        @slot extra\n" +
				"@main\n" +
				"    +box\n" +
				"        @slot #extra\n" +
				"            i hi\n" +
				"            +super\n",
			want: `<div><i>hi</i></div>`,
		},
		{
			name: "scoped slot passes data",
			src: "@export comp list(;slots={})\n" +
				"    ~ item := \"X\"\n" +
				"    ul\n" +
				"        @slot row(item)\n" +
				"            li {= item }\n" +
				"@main\n" +
				"    +list\n" +
				"        @slot #row(item)\n" +
				"            li.custom {= item }\n",
			want: `<ul><li class="custom">X</li></ul>`,
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
