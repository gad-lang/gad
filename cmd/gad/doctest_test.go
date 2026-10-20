package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFencedGadBlocks(t *testing.T) {
	md := "intro\n\n```gad\nsum(1, 2)\n= 3\n```\n\nmore\n\n```\nnot gad\n```\n"
	blocks := fencedGadBlocks(md)
	require.Len(t, blocks, 1)
	require.Equal(t, "sum(1, 2)\n= 3", blocks[0])
}

func TestRunnableGadFence(t *testing.T) {
	// Runnable: exactly-three-backtick gad fences, with or without benign flags.
	require.True(t, runnableGadFence("```gad"))
	require.True(t, runnableGadFence("```Gad"))
	// Not runnable: other languages, wider fences, or an ignore/no-run modifier.
	require.False(t, runnableGadFence("```"))
	require.False(t, runnableGadFence("```gadx"))
	require.False(t, runnableGadFence("```go"))
	require.False(t, runnableGadFence("``````gad"))     // wider fence
	require.False(t, runnableGadFence("```gad ignore")) // illustrative fragment
	require.False(t, runnableGadFence("```gad no-run"))
}

func TestFencedGadBlocksIgnore(t *testing.T) {
	// A ```gad ignore fence highlights as Gad on the site but is never run.
	md := "```gad\nrun := 1\n```\n\n```gad ignore\nimport(\"./missing.gad\")\n```\n"
	blocks := fencedGadBlocks(md)
	require.Len(t, blocks, 1)
	require.Equal(t, "run := 1", blocks[0])
}

func TestRunExamplePass(t *testing.T) {
	require.NoError(t, runExample("sum := func(a, b) { return a + b }\nsum(2, 3)\n= 5"))
	require.NoError(t, runExample("[1, 2, 3]\n= [1, 2, 3]"))
	require.NoError(t, runExample("\"he\" + \"llo\"\n= \"hello\" // a string"))
	require.NoError(t, runExample("println(\"a\")\n<<< a\nprintln(1, 2)\n<<<\n1 2\n>>>"))
	require.NoError(t, runExample("x := 2\nprintln(x)")) // no checks: just runs
}

func TestRunExampleMismatch(t *testing.T) {
	err := runExample("1 + 1\n= 3")
	require.Error(t, err)
	require.Contains(t, err.Error(), "CheckError: got 2, want 3")
	err = runExample("println(1)\n<<< 2")
	require.Error(t, err)
	require.Contains(t, err.Error(), `CheckError: output: got "1", want "2"`)
}

func TestRunExampleRunError(t *testing.T) {
	err := runExample("undefinedFunc()")
	require.Error(t, err)
	require.Contains(t, err.Error(), "running example")
}

func TestCheckFileExamples(t *testing.T) {
	src := "/**\n```gad\n1 + 1\n= 2\n```\n**/\nconst A = 1\n" +
		"/**\n```gad\n1 + 1\n= 99\n```\n**/\nconst B = 2\n"
	results := checkFileExamples("m.gad", []byte(src))
	require.Len(t, results, 2)
	require.NoError(t, results[0].err)
	require.Error(t, results[1].err)
}
