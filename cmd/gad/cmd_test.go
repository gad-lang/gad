//go:build !js
// +build !js

package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/gad-lang/gad/parser/node"
	"github.com/stretchr/testify/require"
)

// writeFile is a tiny helper that writes content to dir/name, creating parent
// directories as needed.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func relNames(t *testing.T, base string, paths []string) []string {
	t.Helper()
	out := make([]string, len(paths))
	for i, p := range paths {
		r, err := filepath.Rel(base, p)
		require.NoError(t, err)
		out[i] = filepath.ToSlash(r)
	}
	sort.Strings(out)
	return out
}

func TestCollectFmtTargets(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.gad", "a:=1\n")
	writeFile(t, dir, "b.gad", "b:=2\n")
	writeFile(t, dir, "notes.txt", "ignore me\n")
	writeFile(t, dir, ".hidden.gad", "h:=1\n")
	writeFile(t, dir, "sub/c.gad", "c:=3\n")
	writeFile(t, dir, ".hiddendir/d.gad", "d:=4\n")

	t.Run("non-recursive dir", func(t *testing.T) {
		got, err := collectFmtTargets([]string{dir}, &fileFilter{})
		require.NoError(t, err)
		require.Equal(t, []string{"a.gad", "b.gad"}, relNames(t, dir, got))
	})

	t.Run("recursive dir skips hidden", func(t *testing.T) {
		got, err := collectFmtTargets([]string{dir + "/..."}, &fileFilter{})
		require.NoError(t, err)
		require.Equal(t, []string{"a.gad", "b.gad", "sub/c.gad"}, relNames(t, dir, got))
	})

	t.Run("explicit file always included", func(t *testing.T) {
		got, err := collectFmtTargets([]string{filepath.Join(dir, "notes.txt")}, &fileFilter{})
		require.NoError(t, err)
		require.Equal(t, []string{"notes.txt"}, relNames(t, dir, got))
	})

	t.Run("exclude glob", func(t *testing.T) {
		got, err := collectFmtTargets([]string{dir}, &fileFilter{excludeGlobs: globList{"b.gad"}})
		require.NoError(t, err)
		require.Equal(t, []string{"a.gad"}, relNames(t, dir, got))
	})

	t.Run("include overrides exclude", func(t *testing.T) {
		got, err := collectFmtTargets([]string{dir},
			&fileFilter{excludeGlobs: globList{"*.gad"}, includeGlobs: globList{"b.gad"}})
		require.NoError(t, err)
		require.Equal(t, []string{"b.gad"}, relNames(t, dir, got))
	})

	t.Run("exclude regex", func(t *testing.T) {
		re := reList{}
		require.NoError(t, re.Set(`^b\.gad$`))
		got, err := collectFmtTargets([]string{dir}, &fileFilter{excludeRe: re})
		require.NoError(t, err)
		require.Equal(t, []string{"a.gad"}, relNames(t, dir, got))
	})

	t.Run("exclude regex on full path (recursive)", func(t *testing.T) {
		re := reList{}
		require.NoError(t, re.Set(`/sub/`))
		got, err := collectFmtTargets([]string{dir + "/..."}, &fileFilter{excludeRe: re})
		require.NoError(t, err)
		require.Equal(t, []string{"a.gad", "b.gad"}, relNames(t, dir, got))
	})
}

func TestFormatTargetInPlace(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "messy.gad", "x:=1\nif x>0{println(x)}\n")

	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out := bytes.NewBuffer(nil)
	var mu sync.Mutex
	tgt := fmtTarget{path: p}

	require.NoError(t, o.formatTarget(tgt, false, &mu, out))

	formatted, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Contains(t, string(formatted), "if (x > 0) {\n")
	require.Contains(t, out.String(), p)

	// Idempotent: a second pass reports no change.
	out2 := bytes.NewBuffer(nil)
	require.NoError(t, o.formatTarget(tgt, false, &mu, out2))
	require.Empty(t, out2.String())

	again, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, string(formatted), string(again))
}

func TestFormatTargetBackup(t *testing.T) {
	dir := t.TempDir()
	const orig = "y:=2\nif y>0{println(y)}\n"
	p := writeFile(t, dir, "bk.gad", orig)

	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	var mu sync.Mutex
	tgt := fmtTarget{path: p, backup: true, backupFormat: "BASE_NAME.backup.gad"}

	require.NoError(t, o.formatTarget(tgt, false, &mu, bytes.NewBuffer(nil)))

	backup, err := os.ReadFile(filepath.Join(dir, "bk.backup.gad"))
	require.NoError(t, err)
	require.Equal(t, orig, string(backup), "backup keeps the original source")
}

func TestFormatTargetOutDir(t *testing.T) {
	dir := t.TempDir()
	const orig = "z:=3\nif z>0{println(z)}\n"
	p := writeFile(t, dir, "src/a.gad", orig)
	outDir := filepath.Join(dir, "out")

	o := &fmtOptions{codeFlags: fmtFormatFlag(), out: outDir}
	var mu sync.Mutex
	tgt := fmtTarget{path: p, root: filepath.Join(dir, "src")}

	require.NoError(t, o.formatTarget(tgt, false, &mu, bytes.NewBuffer(nil)))

	// input unchanged
	in, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, orig, string(in))

	// formatted copy written under out, mirroring the path relative to root
	got, err := os.ReadFile(filepath.Join(outDir, "a.gad"))
	require.NoError(t, err)
	require.Contains(t, string(got), "if (z > 0) {\n")
}

// newFmtFlagSet registers the fmt flags on a fresh FlagSet bound to o, mirroring
// the command's New callback.
func newFmtFlagSet(o *fmtOptions) *flag.FlagSet {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	o.registerFlags(fs)
	return fs
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".gad.yaml")
	writeFile(t, dir, ".gad.yaml", strings.Join([]string{
		"fmt:",
		"  exclude:",
		"    - '*_gen.gad'",
		"  backup-format: 'BASE_NAME.bak.gad'",
		"  report-format: json",
		"  input_dirs:",
		"    - path: src",
		"      backup: true",
		"      report: src.json",
		"other:",
		"  ignored: true",
	}, "\n"))

	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	fs := newFmtFlagSet(o)
	require.NoError(t, fs.Parse([]string{"--config", cfg}))
	require.NoError(t, o.loadConfig(fs))

	require.Equal(t, globList{"*_gen.gad"}, o.exclude)
	require.Equal(t, "BASE_NAME.bak.gad", o.backupFormat)
	require.Equal(t, "json", o.reportFormat)
	require.Len(t, o.inputDirs, 1)
	require.Equal(t, "src", o.inputDirs[0].Path)
	require.True(t, o.inputDirs[0].Backup)
	require.Equal(t, "src.json", o.inputDirs[0].Report)
}

func TestLoadConfigCLIOverrides(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, ".gad.yaml")
	writeFile(t, dir, ".gad.yaml", "fmt:\n  backup-format: from-config\n")

	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	fs := newFmtFlagSet(o)
	require.NoError(t, fs.Parse([]string{"--config", cfg, "--backup-format", "from-cli"}))
	require.NoError(t, o.loadConfig(fs))

	require.Equal(t, "from-cli", o.backupFormat, "command line wins over config")
}

func TestNoFormatFlag(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	fs := newFmtFlagSet(o)
	require.NoError(t, fs.Parse([]string{"--no-format"}))
	require.Equal(t, node.CodeWriteContextFlag(0), o.codeFlags&node.CodeWriteContextFlagFormat)
}

func TestTranspileFlags(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	fs := newFmtFlagSet(o)

	// The flag name is derived from the TranspileOptions.WriteFunc field.
	require.NotNil(t, fs.Lookup("transpile-write-func"))
	require.False(t, o.transpileSet)

	require.NoError(t, fs.Parse([]string{"--transpile-write-func", "out.Write"}))
	require.True(t, o.transpileSet)
	require.Equal(t, "out.Write", o.transpile.WriteFunc)
}

func TestMarshalReport(t *testing.T) {
	r := fmtReport{
		Files: []fmtReportFile{{Path: "a.gad", Error: nil}, {Path: "b.gad", Error: "boom"}},
	}

	yamlOut, err := marshalReport("yaml", r)
	require.NoError(t, err)
	require.Contains(t, string(yamlOut), "error: null")
	require.Contains(t, string(yamlOut), "error: boom")

	jsonOut, err := marshalReport("json", r)
	require.NoError(t, err)
	require.Contains(t, string(jsonOut), `"error": null`)
	require.Contains(t, string(jsonOut), `"error": "boom"`)
}

func TestValidateReportFormat(t *testing.T) {
	require.NoError(t, validateReportFormat(""))
	require.NoError(t, validateReportFormat("yaml"))
	require.NoError(t, validateReportFormat("json"))
	require.Error(t, validateReportFormat("xml"))
}
