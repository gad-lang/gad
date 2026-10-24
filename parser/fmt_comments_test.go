package parser_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad/parser"
	. "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// The comments inside a block stay with their item — above it, at the end of
// its line — when the formatter writes it (#27): a class's `props`, `new`,
// `methods` and a member's overloads (in an exported class too), a func's
// overloads, a dict, an array, a key-value array, call arguments, a match.
func TestFormatBlockComments(t *testing.T) {
	for _, src := range []string{
		`export class Shape {
	name str = "?"
	methods {
		// how big
		area() => 0 // none
		/// what the shape is
		describe() => "shape " + this.name
	}
}`,
		`class C {
	// before x
	x int // x trail
	props {
		// before p
		p() => 1 // p trail
	}
	new {
		// before new 1
		(this) => this() // new trail
		// before new 2
		(this, x) => this(; x=x)
	}
	methods {
		n {
			// before n 1
			(a) => a // n1 trail
			// before n 2
			(a, b) => a + b
		}
	}
}`,
		`func F {
	// before F1
	(a int) => a // F1 trail

	// before F2
	(a str) => a
}`,
		`d := {
	// before k
	k: 1 // k trail
	// before j
	j: 2
}`,
		`arr := [
	// before 1
	1 // one trail
	// before 2
	2
]`,
		`kv := (;
	// before a
	a=1 // a trail
	// before b
	b=2
)`,
		`f(
	// before arg
	1; // arg trail
	// before named
	n=2
)`,
		`g(;
	// a
	a=1,
	b // flag
)`,
		`r := match x {
	// before 1
	1: "a" // trail
	// before else
	else: "b"
}`,
	} {
		fs := source.NewFileSet()
		f := fs.AddFileData("fmt", -1, []byte(src))
		file, err := NewParserWithOptions(f, &ParserOptions{Mode: ParseComments}, nil).ParseFile()
		require.NoError(t, err, src)
		out := Code(file.Stmts, CodeWithComments(f, file.Comments), CodeWithPrefix("\t"), CodeWithMinimalParens())
		require.Equal(t, src, out)
	}
}

// try/catch: a catch without a name writes none (not `catch ‹null›`).
func TestFormatTryCatch(t *testing.T) {
	for _, src := range []string{
		"try {\n\tx()\n} catch e {\n\ty()\n} finally {\n\tz()\n}",
		"try {\n\tx()\n} catch {\n\ty()\n}",
	} {
		fs := source.NewFileSet()
		f := fs.AddFileData("fmt", -1, []byte(src))
		file, err := NewParserWithOptions(f, &ParserOptions{Mode: ParseComments}, nil).ParseFile()
		require.NoError(t, err, src)
		require.Equal(t, src, Code(file.Stmts, CodeWithComments(f, file.Comments), CodeWithPrefix("\t")))
	}
}

// A group (`var (…)`) whose spec reads nothing — after enough errors the
// parser stops advancing — is an error, not a loop that grows until the
// memory runs out (it killed `gad fmt`, the output of an old formatter:
// testdata/gendecl_no_progress.gad).
func TestParseGenDeclNoProgress(t *testing.T) {
	src, err := os.ReadFile("testdata/gendecl_no_progress.gad")
	require.NoError(t, err)
	fs := source.NewFileSet()
	f := fs.AddFileData("decl", -1, src)
	_, err = NewParserWithOptions(f, &ParserOptions{Mode: ParseComments}, nil).ParseFile()
	require.Error(t, err)
}
