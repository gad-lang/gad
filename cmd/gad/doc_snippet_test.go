// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad/web/gadbridge"
	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

// snippetSrc is a module whose doc references two examples: a value check, an
// output check after the `}` (all the example printed) and a context (`name`)
// from an earlier example.
const snippetSrc = `/***
# m

@snippet greet

@snippet loud
***/

example base {
    name := "Gad"
}

example greet {
    greet := "hi " + name
    greet
    = "hi Gad"
}

example loud {
    println("A")
    println("B")
}
<<<
A
B
>>>
`

func TestExtractSnippets(t *testing.T) {
	snips := extractSnippets([]byte(snippetSrc))
	require.Len(t, snips, 3)
	require.Equal(t, `name := "Gad"`, snips["base"].code)
	require.Equal(t, "greet := \"hi \" + name\ngreet\n= \"hi Gad\"", snips["greet"].code)
	// the output check after the `}` is part of the snippet
	require.Equal(t, "println(\"A\")\nprintln(\"B\")\n<<<\nA\nB\n>>>", snips["loud"].code)
	// in source order
	var names []string
	for _, s := range collectSnippets([]byte(snippetSrc)) {
		names = append(names, s.Name)
	}
	require.Equal(t, []string{"base", "greet", "loud"}, names)
}

func TestExpandSnippets(t *testing.T) {
	out := expandSnippets("before\n@snippet greet\nafter\n@snippet missing", extractSnippets([]byte(snippetSrc)), "gad")
	require.Equal(t, "before\n```gad\ngreet := \"hi \" + name\ngreet\n= \"hi Gad\"\n```\nafter\n@snippet missing", out)
}

// The examples are verified by running the file once with checks on; only a
// file with an example or a check in its code runs.
func TestVerifyExamples(t *testing.T) {
	require.NoError(t, verifyExamples("m.gad", []byte(snippetSrc), "gad"))
	err := verifyExamples("m.gad", []byte("example x {\n    1 + 1\n    = 3\n}\n"), "gad")
	require.ErrorContains(t, err, "m.gad: CheckError: got 2, want 3")
	err = verifyExamples("m.gad", []byte("example x {\n    println(1)\n}\n<<< 2\n"), "gad")
	require.ErrorContains(t, err, `output: got "1", want "2"`)
	// no check in the code (a doc fence's are the doctest's): not run
	require.NoError(t, verifyExamples("m.gad", []byte("/**\n```gad\n1\n= 2\n```\n**/\nundefinedFunc()\n"), "gad"))
	require.False(t, hasChecks([]byte("/**\n= 1\n**/\nx := 1\n")))
	require.True(t, hasChecks([]byte("x := 1\n= 1\n")))
}

func TestExampleSourceDropsModuleDoc(t *testing.T) {
	src := exampleSource([]byte(snippetSrc))
	require.NotContains(t, src, "@snippet")
	require.Contains(t, src, "example greet {")
}

func TestDocCommandExpandsAndVerifiesSnippets(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.gad"), []byte(snippetSrc), 0o644))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	var out, errBuf bytes.Buffer
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{"--no-config", "m.gad"}}
	runCtx, err := docCommand().Parse(inCtx)
	require.NoError(t, err)
	require.NoError(t, runCtx.Run())

	md, err := os.ReadFile(filepath.Join(dir, "doc", "m.md"))
	require.NoError(t, err)
	s := string(md)
	require.Contains(t, s, "# m")
	require.Contains(t, s, `= "hi Gad"`)
	require.Contains(t, s, "<<<\nA\nB\n>>>")
}

func TestDocCommandFailingCheck(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.gad"), []byte("/***\n@snippet x\n***/\nexample x {\n    2 * 2\n    = 5\n}\n"), 0o644))
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()
	var out, errBuf bytes.Buffer
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{"--no-config", "m.gad"}}
	runCtx, err := docCommand().Parse(inCtx)
	require.NoError(t, err)
	err = runCtx.Run()
	require.Error(t, err)
	require.Contains(t, err.Error()+errBuf.String(), "got 4, want 5")
}

// A snippet's code runs as the file does: relative `import("./…")` / `embed`
// paths resolve against the file's directory.
func TestSnippetRunsInFileDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lib.gad"), []byte("export answer = 42\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("hi"), 0o644))
	src := []byte("/***\nDemo.\n\n@snippet rel\n***/\n\n" +
		"example rel {\n" +
		"    [import(\"./lib.gad\").answer, str(embed(\"data.txt\").data)]\n" +
		"    = [42, \"hi\"]\n" +
		"}\n")
	path := filepath.Join(dir, "main.gad")
	require.NoError(t, os.WriteFile(path, src, 0o644))

	doc, err := gadbridge.ExtractDoc(string(src), "gad")
	require.NoError(t, err)
	_, err = buildDocDict(doc, path, src, sourceTypeFor(path), true)
	require.NoError(t, err)
	require.Equal(t, ".", exampleWorkDir, "the work dir is restored after the file")
}

// A snippet holding a ``` fence (in a string) gets a wider fence.
func TestSnippetEmbeddedFence(t *testing.T) {
	src := "example f {\n    s := \"```\"\n}\n"
	out := expandSnippets("@snippet f", extractSnippets([]byte(src)), "gad")
	require.Contains(t, out, "````gad\n")
}
