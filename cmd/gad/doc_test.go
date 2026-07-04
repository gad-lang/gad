package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocGeneratorFromContentAndFromFile(t *testing.T) {
	gen := &DocGenerator{MustExported: true, NoTest: true}

	src := []byte("/// pi\nexport Pi = 3.14\n")

	// FromContent renders the Markdown (must-exported flat layout).
	md, err := gen.FromContent("m.gad", src)
	require.NoError(t, err)
	require.Contains(t, md, "# m\n")
	require.Contains(t, md, "### const **Pi**")

	// FromFile returns the same Markdown plus the mirrored output path, without
	// touching the filesystem.
	res, err := gen.FromFile(src, filepath.Join("src", "m.gad"), "doc", "src")
	require.NoError(t, err)
	require.Equal(t, md, res.Markdown)
	require.Equal(t, filepath.Join("doc", "m.md"), res.OutPath)
	require.Equal(t, 0, res.ExamplesFailed)
}

// TestDocGeneratorGroupsTestsAndBenchs verifies that `test`/`bench` statements
// are documented in their own Tests and Benchs sections (not mixed into the API
// sections), with their names and doc comments, in source order.
func TestDocGeneratorGroupsTestsAndBenchs(t *testing.T) {
	src := []byte("/// Add two numbers.\n" +
		"func add(a, b) => a + b\n\n" +
		"/// add is commutative\n" +
		"test addCommutes { t.equal(add(1, 2), add(2, 1)) }\n\n" +
		"test \"fib of ten\" { t.equal(55, fib(10)) }\n\n" +
		"/// the fib benchmark\n" +
		"bench \"fib 15\" { for i := 0; i < t.n; i++ {} }\n")

	gen := &DocGenerator{NoTest: true}
	md, err := gen.FromContent("m.gad", src)
	require.NoError(t, err)

	// the real declaration is still documented in the API section
	require.Contains(t, md, "func **add**")

	// dedicated Tests / Benchs sections (heading + TOC bullets)
	require.Contains(t, md, "\n## Tests\n")
	require.Contains(t, md, "\n## Benchs\n")
	require.Contains(t, md, "[Tests](#tests)")
	require.Contains(t, md, "[Benchs](#benchs)")

	// test entries: name, code line and doc comment
	require.Contains(t, md, "### test **addCommutes**")
	require.Contains(t, md, "test addCommutes { … }")
	require.Contains(t, md, "add is commutative")
	require.Contains(t, md, "### test **fib of ten**") // string name
	require.Contains(t, md, `test "fib of ten" { … }`) // rendered quoted
	require.Contains(t, md, "### bench **fib 15**")
	require.Contains(t, md, "the fib benchmark")

	// a bench is not listed among the tests
	tests := md[strings.Index(md, "## Tests"):strings.Index(md, "## Benchs")]
	require.NotContains(t, tests, "fib 15")
}

// TestDocGeneratorNestedTests verifies that nested `test` statements (subtests)
// are documented with parent/child qualified names and their own doc comments.
func TestDocGeneratorNestedTests(t *testing.T) {
	src := []byte("/// top add tests\n" +
		"test sum {\n" +
		"    /// integer case\n" +
		"    test ints { t.equal(3, add(1, 2)) }\n" +
		"    /// float case\n" +
		"    test floats {\n" +
		"        /// deep case\n" +
		"        test other { t.equal(5.0, 2.5 + 2.5) }\n" +
		"    }\n" +
		"}\n")

	gen := &DocGenerator{NoTest: true}
	md, err := gen.FromContent("m.gad", src)
	require.NoError(t, err)

	require.Contains(t, md, "### test **sum**")
	require.Contains(t, md, "top add tests")
	require.Contains(t, md, "### test **sum/ints**")
	require.Contains(t, md, "integer case")
	require.Contains(t, md, "### test **sum/floats**")
	require.Contains(t, md, "### test **sum/floats/other**") // deep qualified name
	require.Contains(t, md, "deep case")
}

func TestDocGeneratorFromFileReportsFailingExample(t *testing.T) {
	// A doc-comment example whose result does not match its `>>>` assertion.
	bad := []byte("/**\n```gad\n1\n>>> 2\n```\n**/\nexport A = 1\n")

	var errs []string
	gen := &DocGenerator{OnError: func(msg string) { errs = append(errs, msg) }}
	res, err := gen.FromFile(bad, "b.gad", "doc", ".")
	require.NoError(t, err)
	require.Equal(t, 1, res.ExamplesFailed)
	require.Len(t, errs, 1)
	require.Contains(t, errs[0], "example failed")

	// With NoTest, the failing example is neither run nor reported.
	errs = nil
	genNoTest := &DocGenerator{NoTest: true, OnError: func(msg string) { errs = append(errs, msg) }}
	res, err = genNoTest.FromFile(bad, "b.gad", "doc", ".")
	require.NoError(t, err)
	require.Equal(t, 0, res.ExamplesFailed)
	require.Empty(t, errs)
}
