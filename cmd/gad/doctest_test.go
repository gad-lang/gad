package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStripLineComment(t *testing.T) {
	require.Equal(t, "5", stripLineComment("5 // a result"))
	require.Equal(t, `"a // b"`, stripLineComment(`"a // b" // trailing`))
	require.Equal(t, "[1, 2]", stripLineComment("[1, 2]"))
}

func TestFencedGadBlocks(t *testing.T) {
	md := "intro\n\n```gad\nsum(1, 2)\n>>> 3\n```\n\nmore\n\n```\nnot gad\n```\n"
	blocks := fencedGadBlocks(md)
	require.Len(t, blocks, 1)
	require.Equal(t, "sum(1, 2)\n>>> 3", blocks[0])
}

func TestRunExamplePass(t *testing.T) {
	require.NoError(t, runExample("sum := func(a, b) { return a + b }\nsum(2, 3)\n>>> 5"))
	require.NoError(t, runExample("[1, 2, 3]\n>>> [1, 2, 3]"))
	require.NoError(t, runExample("\"he\" + \"llo\"\n>>> \"hello\" // a string"))
	require.NoError(t, runExample("x := 2\nprintln(x)")) // no checks: just runs
}

func TestRunExampleMismatch(t *testing.T) {
	err := runExample("1 + 1\n>>> 3")
	require.Error(t, err)
	require.Contains(t, err.Error(), "doctest mismatch")
	require.Contains(t, err.Error(), "got 2")
	require.Contains(t, err.Error(), "want 3")
}

func TestRunExampleRunError(t *testing.T) {
	err := runExample("undefinedFunc()")
	require.Error(t, err)
	require.Contains(t, err.Error(), "running example")
}

func TestCheckFileExamples(t *testing.T) {
	src := "/**\n```gad\n1 + 1\n>>> 2\n```\n**/\nconst A = 1\n" +
		"/**\n```gad\n1 + 1\n>>> 99\n```\n**/\nconst B = 2\n"
	results := checkFileExamples("m.gad", []byte(src))
	require.Len(t, results, 2)
	require.NoError(t, results[0].err)
	require.Error(t, results[1].err)
}
