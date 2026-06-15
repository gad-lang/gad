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

func TestFormatTargetTranspileGadt(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "page.gadt", "Hi {%= name %}!\n{% x := 1 %}")

	o := &fmtOptions{codeFlags: fmtFormatFlag(), transpileOn: true}
	o.finalizeTranspile()
	var mu sync.Mutex

	out := bytes.NewBuffer(nil)
	require.NoError(t, o.formatTarget(fmtTarget{path: p, transpile: true}, false, &mu, out))

	// A .gadt is transpiled to a sibling .gad; the original template is kept.
	gadPath := filepath.Join(dir, "page.gad")
	require.FileExists(t, gadPath)
	require.FileExists(t, p)
	require.Contains(t, out.String(), gadPath)

	got, err := os.ReadFile(gadPath)
	require.NoError(t, err)
	require.Contains(t, string(got), `write(raw "Hi ")`)
	require.Contains(t, string(got), "write(name)")
	require.Contains(t, string(got), "x := 1")
}

func TestFormatTargetToStdout(t *testing.T) {
	dir := t.TempDir()
	const orig = "x:=1\nif x>0{println(x)}\n"
	p := writeFile(t, dir, "src/a.gad", orig)

	o := &fmtOptions{codeFlags: fmtFormatFlag(), toStdout: true, boundary: "BND"}
	var mu sync.Mutex
	out := bytes.NewBuffer(nil)
	tgt := fmtTarget{path: p, root: filepath.Join(dir, "src"), index: 7}

	require.NoError(t, o.formatTarget(tgt, false, &mu, out))

	got := out.String()
	// Header carries the input dir (bracketed) and the file relative to it.
	require.Contains(t, got, "-- BND #7 ["+filepath.Join(dir, "src")+"] a.gad\n")
	require.Contains(t, got, "if (x > 0) {\n")
	require.True(t, strings.HasSuffix(got, "-- BND #7\n"), "trailer closes the frame")

	// The input file is left untouched (streamed, not written).
	in, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, orig, string(in))
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
		"  input_dirs:",
		"    - path: src",
		"      backup: true",
		"      report: src.ndjson",
		"other:",
		"  ignored: true",
	}, "\n"))

	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	fs := newFmtFlagSet(o)
	require.NoError(t, fs.Parse([]string{"--config", cfg}))
	require.NoError(t, o.loadConfig(fs))

	require.Equal(t, globList{"*_gen.gad"}, o.exclude)
	require.Equal(t, "BASE_NAME.bak.gad", o.backupFormat)
	require.Len(t, o.inputDirs, 1)
	require.Equal(t, "src", o.inputDirs[0].Path)
	require.True(t, o.inputDirs[0].Backup)
	require.Equal(t, "src.ndjson", o.inputDirs[0].Report)
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

func TestMarshalReportLine(t *testing.T) {
	// A successful file in an input dir: input_dir + relative file, no error.
	ok := marshalReportLine(fmtReportRecord{InputDir: "src", File: "a.gad"})
	require.Equal(t, `{"input_dir":"src","file":"a.gad"}`+"\n", string(ok))

	// A failed explicit file: no input_dir, error present.
	bad := marshalReportLine(fmtReportRecord{File: "b.gad", Error: "boom"})
	require.Equal(t, `{"file":"b.gad","error":"boom"}`+"\n", string(bad))
}

func TestWriteReportNDJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "report.ndjson")
	records := []fmtReportRecord{
		{File: "a.gad"},
		{InputDir: "src", File: "b.gad", Error: "boom"},
	}
	require.NoError(t, writeReport(path, records))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t,
		`{"file":"a.gad"}`+"\n"+`{"input_dir":"src","file":"b.gad","error":"boom"}`+"\n",
		string(data))
}

func TestNewBoundaryUnique(t *testing.T) {
	a, b := newBoundary(), newBoundary()
	require.NotEqual(t, a, b)
	require.Len(t, a, 36) // canonical UUID length
}
