// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestDocCommandJSONYAML runs `gad doc --json --yaml` on a module with a snippet
// and checks the encoded doc structure carries the name/file/lang, the expanded
// prose and the example source, and that `<` is kept literal in JSON.
func TestDocCommandJSONYAML(t *testing.T) {
	dir := t.TempDir()
	src := "/***\n# Demo\n\nprose <b> here.\n\n@snippet greet\n***/\n\n" +
		"//snippet greet\ngreet := \"hi \" + \"Gad\"\ngreet\n/**= \"hi Gad\" **/\n//endsnippet\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.gad"), []byte(src), 0o644))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	var out, errBuf bytes.Buffer
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{
		"--no-config", "--json", "--yaml", "m.gad",
	}}
	runCtx, err := docCommand().Parse(inCtx)
	require.NoError(t, err)
	require.NoError(t, runCtx.Run())

	// JSON: unmarshal and assert the structure.
	jb, err := os.ReadFile(filepath.Join(dir, "doc", "m.json"))
	require.NoError(t, err)
	require.Contains(t, string(jb), "prose <b> here.") // `<` kept literal, not <
	var got docEncoded
	require.NoError(t, json.Unmarshal(jb, &got))
	require.Equal(t, "m", got.Name)
	require.Equal(t, "m.gad", got.File)
	require.Equal(t, "gad", got.Lang)
	require.Contains(t, got.Prose, "# Demo")
	require.Contains(t, got.Prose, "```gad")       // the snippet was expanded
	require.Contains(t, got.Prose, "// => hi Gad") // its verified result
	require.Contains(t, got.Source, `greet := "hi " + "Gad"`)

	// YAML: unmarshal and assert the same core fields.
	yb, err := os.ReadFile(filepath.Join(dir, "doc", "m.yaml"))
	require.NoError(t, err)
	var gy docEncoded
	require.NoError(t, yaml.Unmarshal(yb, &gy))
	require.Equal(t, "m", gy.Name)
	require.Equal(t, "gad", gy.Lang)
	require.Contains(t, gy.Prose, "# Demo")
}
