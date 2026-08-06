// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

// snippetSrc is a runnable .gad module whose prose references two snippets: one
// asserting a value (`/**= … **/`), one asserting STDOUT (`/**< … **/`).
const snippetSrc = "/***\n" +
	"Greeting demo.\n\n" +
	"@snippet greet\n\n" +
	"@snippet shout\n" +
	"***/\n\n" +
	"//snippet greet\n" +
	"greet := \"hi \" + \"Gad\"\n" +
	"greet\n" +
	"/**= \"hi Gad\" **/\n" +
	"//endsnippet\n\n" +
	"//snippet shout\n" +
	"println(\"HELLO\")\n" +
	"/**< HELLO **/\n" +
	"//endsnippet\n"

func TestExtractSnippets(t *testing.T) {
	snips := extractSnippets([]byte(snippetSrc))
	require.Len(t, snips, 2)

	require.Equal(t, "greet := \"hi \" + \"Gad\"\ngreet", snips["greet"].code)
	require.Equal(t, snippetValue, snips["greet"].kind)
	require.Equal(t, `"hi Gad"`, snips["greet"].expected)

	require.Equal(t, "println(\"HELLO\")", snips["shout"].code)
	require.Equal(t, snippetOutput, snips["shout"].kind)
	require.Equal(t, "HELLO", snips["shout"].expected)
}

func TestExpandSnippetsRunsAndVerifies(t *testing.T) {
	snips := extractSnippets([]byte(snippetSrc))

	// Value snippet: run + verify, result inline as `// => value`.
	got, err := expandSnippets("@snippet greet", snips, "gad", true)
	require.NoError(t, err)
	require.Contains(t, got, "```gad")
	require.Contains(t, got, `greet := "hi " + "Gad"`)
	require.Contains(t, got, "// => hi Gad")

	// Output snippet: run + verify, stdout in an Output text block.
	got, err = expandSnippets("@snippet shout", snips, "gad", true)
	require.NoError(t, err)
	require.Contains(t, got, `println("HELLO")`)
	require.Contains(t, got, "Output:")
	require.Contains(t, got, "```text\nHELLO\n```")
}

func TestExpandSnippetsResultMismatchFails(t *testing.T) {
	bad := strings.Replace(snippetSrc, `/**= "hi Gad" **/`, `/**= "wrong" **/`, 1)
	snips := extractSnippets([]byte(bad))
	_, err := expandSnippets("@snippet greet", snips, "gad", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "result mismatch")

	bad = strings.Replace(snippetSrc, "/**< HELLO **/", "/**< BYE **/", 1)
	snips = extractSnippets([]byte(bad))
	_, err = expandSnippets("@snippet shout", snips, "gad", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "output mismatch")
}

func TestExpandSnippetsNoRunShowsExpected(t *testing.T) {
	snips := extractSnippets([]byte(snippetSrc))
	got, err := expandSnippets("@snippet greet", snips, "gad", false)
	require.NoError(t, err)
	// Without running, the marker's raw expected text is shown in a gad fence.
	require.Contains(t, got, `// => "hi Gad"`)
}

func TestExampleSourceStripsMarkersAndModuleDoc(t *testing.T) {
	ex := exampleSource([]byte(snippetSrc))
	require.NotContains(t, ex, "//snippet")
	require.NotContains(t, ex, "//endsnippet")
	require.NotContains(t, ex, "/**=")
	require.NotContains(t, ex, "/**<")
	require.NotContains(t, ex, "Greeting demo.") // leading module doc dropped
	require.Contains(t, ex, `greet := "hi " + "Gad"`)
	require.Contains(t, ex, `println("HELLO")`)
}

// TestDocCommandExpandsAndVerifiesSnippets runs `gad doc` end to end on a module
// with snippets and result markers, and checks the generated Markdown embeds the
// expanded, verified snippets.
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
	require.Contains(t, s, "// => hi Gad")
	require.Contains(t, s, "Output:")
}
