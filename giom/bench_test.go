package giom

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
)

// benchCases are representative templates exercising the parts of the pipeline
// that differ between the streaming and tree-building render models: plain tags
// and text, attribute-heavy tags, loops that produce many nodes, and
// components with slots.
var benchCases = []struct {
	name    string
	src     string
	globals gad.Dict
}{
	{
		name: "page",
		src: `@global Title
@main
    !!! 5
    html[lang="en"]
        head
            title {= Title}
        body
            main.container
                h1 {= Title}
                p Welcome to Giom.
`,
		globals: gad.Dict{"Title": gad.Str("Hello")},
	},
	{
		name: "attrs",
		src: `@main
    div[id="x", class="a b c", data-role="main", tabindex="0"]
        a[href="/x", title="go", target="_blank"] link
        input[type="text", disabled, value=1+2]
`,
		globals: gad.Dict{},
	},
	{
		name: "loop",
		src: `@global items
@main
    ul
        @for i, it in items
            li[class="row"] {= i }: {= it }
`,
		globals: gad.Dict{"items": benchItems(50)},
	},
	{
		name: "components",
		src: `@export comp card(title; kind="primary")
    article.card[class="card--" + kind]
        h2 {= title}
        @slot main
            p default
@global items
@main
    section
        @for i, it in items
            +card(it; kind="secondary")
                p body {= i }
`,
		globals: gad.Dict{"items": benchItems(20)},
	},
}

func benchItems(n int) gad.Array {
	arr := make(gad.Array, n)
	for i := range arr {
		arr[i] = gad.Str("item")
	}
	return arr
}

// writeSink is an io.Writer that discards output while counting bytes, so the
// render's output work is performed but not measured as allocation.
type writeSink struct{ n int }

func (s *writeSink) Write(p []byte) (int, error) { s.n += len(p); return len(p), nil }

// benchRenderFile writes src to a temp .giom file and returns a warmed Render
// (compiled once) plus the file path.
func benchRenderFile(b *testing.B, name, src string, globals gad.Dict) (*Render, string) {
	b.Helper()
	dir := b.TempDir()
	p := filepath.Join(dir, name+".giom")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		b.Fatal(err)
	}
	r := NewRender(dir)
	// Warm the bytecode cache so BenchmarkRender measures the steady-state
	// render path (VM execution + output), not the first-time compile.
	if err := r.Render(io.Discard, p, globals); err != nil {
		b.Fatalf("warm render: %v", err)
	}
	return r, p
}

// BenchmarkRender measures the steady-state (cache-warm) render throughput:
// reading the template, running its cached bytecode, and producing HTML output.
func BenchmarkRender(b *testing.B) {
	for _, tc := range benchCases {
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			r, p := benchRenderFile(b, tc.name, tc.src, tc.globals)
			var sink writeSink
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := r.Render(&sink, p, tc.globals); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkCompile measures parse + convert + compile of a template to Gad
// bytecode (no execution).
func BenchmarkCompile(b *testing.B) {
	builtins := AppendBuiltins(gad.NewBuiltins())
	for _, tc := range benchCases {
		tc := tc
		src := []byte(tc.src)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				st := gad.NewSymbolTable(builtins.NameSet)
				if _, _, err := Compile(st, src, gad.CompileOptions{}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
