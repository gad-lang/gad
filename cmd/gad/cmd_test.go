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

	_, ferr := o.formatTarget(tgt, false, &mu, out)
	require.NoError(t, ferr)

	formatted, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Contains(t, string(formatted), "if (x > 0) {\n")
	require.Contains(t, out.String(), p)

	// Idempotent: a second pass reports no change.
	out2 := bytes.NewBuffer(nil)
	_, ferr = o.formatTarget(tgt, false, &mu, out2)
	require.NoError(t, ferr)
	require.Empty(t, out2.String())

	again, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, string(formatted), string(again))
}

func TestFormatPreservesComments(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "// leading comment\n" +
		"x := 1 // trailing on x\n\n" +
		"/* block comment */\n" +
		"func f() {\n" +
		"\t// inside block\n" +
		"\treturn 1\n" +
		"}\n" +
		"// final comment\n"

	out, err := o.formatSource("c.gad", []byte(src), false)
	require.NoError(t, err)

	for _, want := range []string{
		"// leading comment",
		"x := 1 // trailing on x",
		"/* block comment */",
		"// inside block",
		"// final comment",
	} {
		require.Contains(t, out, want)
	}

	// Idempotent: formatting the formatted output yields the same result.
	out2, err := o.formatSource("c.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

func TestFormatPreservesDocComments(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "/? the pi value\n" +
		"pi := 3.14\n\n" +
		"const x = 1 /? inline doc on x\n\n" +
		"func f() {\n" +
		"\t/? local doc\n" +
		"\treturn 1\n" +
		"}\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)

	for _, want := range []string{
		"/? the pi value",
		"const x = 1 /? inline doc on x",
		"/? local doc",
	} {
		require.Contains(t, out, want)
	}

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

func TestFormatPreservesDocBlockComments(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "/??\n" +
		"a block doc\n" +
		"spanning lines\n" +
		"??\n" +
		"x := 1\n\n" +
		"/???\n" +
		"a root block doc\n" +
		"???\n" +
		"y := 2\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)

	for _, want := range []string{
		"/??\na block doc\nspanning lines\n??",
		"/???\na root block doc\n???",
	} {
		require.Contains(t, out, want)
	}

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocTravelsWithMergedDecl verifies a lead doc on a declaration that
// gets merged into the previous declaration group travels onto its spec instead
// of being misplaced by position-based comment preservation.
func TestFormatDocTravelsWithMergedDecl(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "const a = 1\n" +
		"/? the b value\n" +
		"const b = 2\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)

	// The two const decls merge; the doc stays attached to `b` inside the group.
	require.Equal(t, "const (\n\ta = 1\n\t/? the b value\n\tb = 2\n)", strings.TrimSpace(out))

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocLeadOnGenDeclAndFunc verifies lead docs on a top-level decl and a
// func statement are emitted by their node (preserved, idempotent).
func TestFormatDocLeadOnGenDeclAndFunc(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "/? this is the server addr\n" +
		"const ServerAddr = \":0\"\n\n" +
		"/? sum values\n" +
		"func sum(a, b) {\n" +
		"\treturn a + b\n" +
		"}\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)

	require.Contains(t, out, "/? this is the server addr\nconst ServerAddr = \":0\"")
	require.Contains(t, out, "/? sum values\nfunc sum")

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocSingleBlockConversion verifies a short BLOCK doc collapses to a
// SINGLE doc and a long SINGLE doc expands to a BLOCK, both idempotently.
func TestFormatDocSingleBlockConversion(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}

	// short block -> single
	short := "/??\nthe pi value\n??\nconst pi = 3.14\n"
	out, err := o.formatSource("d.gad", []byte(short), false)
	require.NoError(t, err)
	require.Contains(t, out, "/? the pi value\nconst pi = 3.14")
	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)

	// long single -> block (content wrapped, fenced)
	long := "/? " + strings.Repeat("word ", 30) + "\nconst x = 1\n"
	out, err = o.formatSource("d.gad", []byte(long), false)
	require.NoError(t, err)
	require.Contains(t, out, "/??\n")
	require.Contains(t, out, "\n??\nconst x = 1")
	out2, err = o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocInGroupTrailing verifies an inline trailing doc inside a `( … )`
// group stays attached to its spec on the same line.
func TestFormatDocInGroupTrailing(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "var (\n\tc = 1\n\td = 2 /? the value of d\n\te = 3\n)\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)
	require.Contains(t, out, "d = 2 /? the value of d")

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocPerMethod verifies a func-with-methods keeps its func-level doc
// and per-method docs in place.
func TestFormatDocPerMethod(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "/??\nthis is a difference calculator.\nsee all methods here.\n??\n" +
		"func diff {\n" +
		"\t/? compute diff of b and a\n" +
		"\t(a, b) => b - a\n\n" +
		"\t/? compute diff with floats\n" +
		"\t(a, b) => b - a\n" +
		"}\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)
	require.Contains(t, out, "/? this is a difference calculator. see all methods here.\nfunc diff")
	require.Contains(t, out, "/? compute diff of b and a")
	require.Contains(t, out, "/? compute diff with floats")

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocPropAndMeti verifies prop and meti statements keep their docs.
func TestFormatDocPropAndMeti(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "/? the name property\n" +
		"prop name {\n\t/? getter\n\t() => \"x\"\n}\n\n" +
		"/? a stringer interface\n" +
		"meti stringer {\n\t() <str>\n}\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)
	require.Contains(t, out, "/? the name property\nprop name")
	require.Contains(t, out, "/? getter")
	require.Contains(t, out, "/? a stringer interface\nmeti stringer")

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocMetiPerHeader verifies a doc preceding a meti header is kept on
// its own line above the header (forcing the one-per-line layout).
func TestFormatDocMetiPerHeader(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	src := "/? a calculator interface\n" +
		"meti calc {\n" +
		"\t/? add two ints\n" +
		"\t(a int, b int) <int>\n\n" +
		"\t/? subtract\n" +
		"\t(a int, b int) <int>\n" +
		"}\n"

	out, err := o.formatSource("d.gad", []byte(src), false)
	require.NoError(t, err)
	require.Contains(t, out, "/? a calculator interface\nmeti calc")
	require.Contains(t, out, "/? add two ints")
	require.Contains(t, out, "/? subtract")

	out2, err := o.formatSource("d.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, out2)
}

// TestFormatDocCommaIdentError verifies a doc trailing a comma-separated
// valueless identifier is a parse error.
func TestFormatDocCommaIdentError(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	_, err := o.formatSource("d.gad", []byte("var (\n\tf, g /? f and g\n)\n"), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "comma-separated identifier")
}

func TestFormatTargetTranspileGadt(t *testing.T) {
	dir := t.TempDir()
	p := writeFile(t, dir, "page.gadt", "Hi {%= name %}!\n{% x := 1 %}")

	o := &fmtOptions{codeFlags: fmtFormatFlag(), transpileOn: true}
	o.finalizeTranspile()
	var mu sync.Mutex

	out := bytes.NewBuffer(nil)
	_, ferr := o.formatTarget(fmtTarget{path: p, transpile: true}, false, &mu, out)
	require.NoError(t, ferr)

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

func TestFormatTargetNoSave(t *testing.T) {
	dir := t.TempDir()
	const orig = "x:=1\nif x>0{println(x)}\n"
	p := writeFile(t, dir, "a.gad", orig)

	o := &fmtOptions{codeFlags: fmtFormatFlag(), noSave: true}
	var mu sync.Mutex
	out := bytes.NewBuffer(nil)

	formatted, ferr := o.formatTarget(fmtTarget{path: p}, false, &mu, out)
	require.NoError(t, ferr)

	// the formatted result is returned (for --report-contents) but nothing is
	// written or echoed.
	require.Contains(t, formatted, "if (x > 0) {\n")
	require.Empty(t, out.String())
	in, err := os.ReadFile(p)
	require.NoError(t, err)
	require.Equal(t, orig, string(in), "--no-save leaves the file untouched")
}

func TestFormatTargetBackup(t *testing.T) {
	dir := t.TempDir()
	const orig = "y:=2\nif y>0{println(y)}\n"
	p := writeFile(t, dir, "bk.gad", orig)

	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	var mu sync.Mutex
	tgt := fmtTarget{path: p, backup: true, backupFormat: "BASE_NAME.backup.gad"}

	_, ferr := o.formatTarget(tgt, false, &mu, bytes.NewBuffer(nil))
	require.NoError(t, ferr)

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

	_, ferr := o.formatTarget(tgt, false, &mu, bytes.NewBuffer(nil))
	require.NoError(t, ferr)

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
