// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

//go:build !js
// +build !js

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	cc "github.com/moisespsena-go/command-context"
	"gopkg.in/yaml.v3"
)

// ctxKey is the type of the context keys used to pass parsed options between a
// subcommand's lifecycle callbacks.
type ctxKey string

const (
	runFlagsKey    ctxKey = "runFlags"
	fmtOptionsKey  ctxKey = "fmtOptions"
	docOptionsKey  ctxKey = "docOptions"
	defaultCfgFile        = ".gad.yaml"
	// docConfigKey is the root mapping key in the config file holding the `doc`
	// subcommand settings.
	docConfigKey = "doc"
	// fmtConfigKey is the root mapping key in the config file holding the `fmt`
	// subcommand settings.
	fmtConfigKey = "fmt"
	// templateConfigKey is the root mapping key holding template/mixed-mode
	// settings (the start/end delimiters).
	templateConfigKey = "template"
)

// templateConfig is the `.gad.yaml` `template:` section.
type templateConfig struct {
	StartDelimiter string `yaml:"start_delimiter"`
	EndDelimiter   string `yaml:"end_delimiter"`
}

// loadTemplateConfig reads the `template:` section of `<dir>/.gad.yaml` and
// fills the template delimiter globals that were not already set on the command
// line (CLI flags win). Missing file/section is not an error.
func loadTemplateConfig(dir string) {
	data, err := os.ReadFile(filepath.Join(dir, defaultCfgFile))
	if err != nil {
		return
	}
	var top map[string]any
	if yaml.Unmarshal(data, &top) != nil {
		return
	}
	section, ok := top[templateConfigKey]
	if !ok {
		return
	}
	b, _ := yaml.Marshal(section)
	var tc templateConfig
	if yaml.Unmarshal(b, &tc) != nil {
		return
	}
	if templateStartDelim == "" {
		templateStartDelim = tc.StartDelimiter
	}
	if templateEndDelim == "" {
		templateEndDelim = tc.EndDelimiter
	}
}

// optionalCommands holds subcommand factories registered by build-tagged files
// (e.g. `debug` and `ide`), so they can be excluded with `-tags nodebug,noide`.
var optionalCommands []func() *cc.Command

// registerCommand registers an optional subcommand factory. Called from
// build-tagged init functions; the name is carried by the factory's Command.
func registerCommand(_ string, factory func() *cc.Command) {
	optionalCommands = append(optionalCommands, factory)
}

// buildRootCommand assembles the `gad` command tree: a root that prints help
// plus the `run` and `fmt` subcommands and any optional (build-tagged) ones.
func buildRootCommand() *cc.Command {
	root := &cc.Command{
		Name:        filepath.Base(os.Args[0]),
		Usage:       "[flags] [SCRIPT_FILE [ARGS...]]",
		Description: "Run a Gad script file (or stdin with -), or start the REPL when no file is given.",
		New: func(ctx *cc.CommandContext) error {
			ctx.WithValue(runFlagsKey, registerRunFlags(ctx.Flags()))
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			rf := ctx.Value(runFlagsKey).(*runFlags)
			filePath, params := rf.apply(ctx.Flags())
			if filePath != "" && filePath != "-" {
				if _, err := os.Stat(filePath); os.IsNotExist(err) && rf.module {
					for _, p := range sourcePath {
						if _, err2 := os.Stat(p); err2 == nil {
							filePath = filepath.Join(p, filePath)
							break
						}
					}
				}
			}
			runScriptOrREPL(ctx.Context, filePath, rf.timeout, params)
			return nil
		},
	}
	root.Sub(fmtCommand())
	root.Sub(docCommand())
	root.Sub(doctestCommand())
	for _, f := range optionalCommands {
		root.Sub(f())
	}
	return root
}

// globList is a repeatable flag whose values may also be comma-separated, e.g.
// `--exclude a,b --exclude c` collects [a, b, c].
type globList []string

func (g *globList) String() string { return strings.Join(*g, ",") }

func (g *globList) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			*g = append(*g, p)
		}
	}
	return nil
}

// reList is a repeatable regex flag. Unlike globList it is not comma-split,
// since a regular expression may legitimately contain commas; each flag
// occurrence (or config list element) is one full pattern.
type reList []*regexp.Regexp

func (r *reList) String() string {
	parts := make([]string, len(*r))
	for i, re := range *r {
		parts[i] = re.String()
	}
	return strings.Join(parts, ", ")
}

func (r *reList) Set(v string) error {
	re, err := regexp.Compile(v)
	if err != nil {
		return err
	}
	*r = append(*r, re)
	return nil
}

// fileFilter decides whether a discovered file is formatted. Globs and regexes
// are tested against both the full path and the base name, so either form may
// be used in a pattern. A file is included when it matches any include
// glob/regex; otherwise it is skipped when it matches any exclude glob/regex.
// Includes win over excludes.
type fileFilter struct {
	includeGlobs globList
	excludeGlobs globList
	includeRe    reList
	excludeRe    reList
}

func (f *fileFilter) match(path string) bool {
	base := filepath.Base(path)
	if matchAnyGlob(f.includeGlobs, path, base) || matchAnyRe(f.includeRe, path, base) {
		return true
	}
	if matchAnyGlob(f.excludeGlobs, path, base) || matchAnyRe(f.excludeRe, path, base) {
		return false
	}
	return true
}

// inputDir is a directory entry declared under the config file's input_dirs
// key, with its own include/exclude globs and backup settings.
type inputDir struct {
	Path         string   `yaml:"path"`
	Includes     []string `yaml:"includes"`
	Excludes     []string `yaml:"excludes"`
	IncludesRe   []string `yaml:"includes_re"`
	ExcludesRe   []string `yaml:"excludes_re"`
	Backup       bool     `yaml:"backup"`
	BackupFormat string   `yaml:"backup_format"`
	Report       string   `yaml:"report"`
	Transpile    bool     `yaml:"transpile"`
}

// fmtReportRecord is the per-file outcome emitted as one NDJSON line. InputDir
// is set only when the file belongs to a directory job (File is then relative
// to that directory). Error carries the failure message, omitted on success.
type fmtReportRecord struct {
	InputDir string `json:"input_dir,omitempty"`
	File     string `json:"file"`
	Error    string `json:"error,omitempty"`
	// Result holds the formatted source, included only with --report-contents.
	Result string `json:"result,omitempty"`

	// display is a human-readable path used for stderr; not serialized.
	display string
}

// fmtOptions holds the parsed flags (and config) of the `fmt` subcommand.
type fmtOptions struct {
	exclude        globList
	include        globList
	excludeRe      reList
	includeRe      reList
	backup         bool
	backupFormat   string
	codeFlags      node.CodeWriteContextFlag
	transpile      node.TranspileOptions
	transpileSet   bool
	transpileOn    bool // --transpile / config `transpile`
	jobs           int
	out            string
	report         string
	reportStream   bool
	reportContents bool
	noSave         bool
	config         string
	noConfig       bool
	inputDirs      []inputDir
}

// fmtFormatFlag returns the default formatting flag (full multi-line layout)
// from which `--no-format` and the `--no-*-in-new-line` flags clear bits.
func fmtFormatFlag() node.CodeWriteContextFlag {
	return node.CodeWriteContextFlagFormat
}

// fmtCommand is the `gad fmt [flags] PATH...` subcommand: it formats Gad source
// files, in place by default.
func fmtCommand() *cc.Command {
	return &cc.Command{
		Name:  "fmt",
		Usage: "[flags] [PATH...]",
		Description: "Format Gad source files.\n" +
			"\nPATH may be a file, a directory or - (stdin). A directory formats the .gad\n" +
			"files directly inside it; write DIR/... to recurse into sub-directories. Hidden\n" +
			"files are ignored and hidden directories are skipped. Without --out, files are\n" +
			"rewritten in place; stdin is always written to stdout.",
		New: func(ctx *cc.CommandContext) error {
			o := &fmtOptions{codeFlags: fmtFormatFlag()}
			o.registerFlags(ctx.Flags())
			ctx.WithValue(fmtOptionsKey, o)
			return nil
		},
		ParseArgs: func(ctx *cc.CommandContext) error {
			o := ctx.Value(fmtOptionsKey).(*fmtOptions)
			if err := o.loadConfig(ctx.Flags()); err != nil {
				return err
			}
			o.finalizeTranspile()
			if len(ctx.Args) == 0 && len(o.inputDirs) == 0 {
				return fmt.Errorf("no input: provide PATH... or input_dirs in the config")
			}
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			o := ctx.Value(fmtOptionsKey).(*fmtOptions)
			return o.run(ctx)
		},
	}
}

// registerFlags registers all `fmt` flags on fs, bound to o.
func (o *fmtOptions) registerFlags(fs *flag.FlagSet) {
	fs.Var(&o.exclude, "exclude", "glob of file base names to skip (repeatable, comma-separated)")
	fs.Var(&o.include, "include", "glob of file base names to format even if excluded (repeatable, comma-separated)")
	fs.Var(&o.excludeRe, "exclude-re", "regex of file base names to skip (repeatable)")
	fs.Var(&o.includeRe, "include-re", "regex of file base names to format even if excluded (repeatable)")
	fs.BoolVar(&o.backup, "backup", false, "write a backup of each file before formatting")
	fs.StringVar(&o.backupFormat, "backup-format", "BASE_NAME.backup.gad",
		"backup file name pattern; BASE_NAME is the file name without its extension")
	fs.IntVar(&o.jobs, "jobs", runtime.NumCPU(), "max concurrent format jobs")
	fs.StringVar(&o.out, "out", "", "output file (single input) or directory; inputs are left unchanged")
	fs.StringVar(&o.report, "report", "",
		"write a per-file NDJSON status report to this path (- for stdout)")
	fs.BoolVar(&o.reportStream, "report-stream", false,
		"write each report record as soon as its file is done, rather than all at the end; "+
			"the report goes to stdout when --report is unset")
	fs.BoolVar(&o.reportContents, "report-contents", false,
		"include the formatted source in each report record under the \"result\" key")
	fs.BoolVar(&o.noSave, "no-save", false,
		"do not write, create or back up any file (read-only); format and report only")
	fs.StringVar(&o.config, "config", defaultCfgFile, "YAML config file with default flag values")
	fs.BoolVar(&o.noConfig, "no-config", false, "do not read the config file")

	fs.BoolFunc("no-format", "disable all multi-line formatting", func(string) error {
		o.codeFlags &^= node.CodeWriteContextFlagFormat
		return nil
	})

	// Boolean flags that clear individual multi-line formatting bits.
	clear := func(name, desc string, bit node.CodeWriteContextFlag) {
		fs.BoolFunc(name, desc, func(string) error {
			o.codeFlags &^= bit
			return nil
		})
	}
	clear("no-array-item-in-new-line", "keep array items on a single line",
		node.CodeWriteContextFlagFormatArrayItemInNewLine)
	clear("no-dict-item-in-new-line", "keep dict items on a single line",
		node.CodeWriteContextFlagFormatDictItemInNewLine)
	clear("no-key-value-array-item-in-new-line", "keep keyValueArray items on a single line",
		node.CodeWriteContextFlagFormatKeyValueArrayItemInNewLine)
	clear("no-call-params-in-new-line", "keep call params on a single line",
		node.CodeWriteContextFlagFormatCallParamsInNewLine)
	clear("no-parem-values-in-new-line", "keep param values on a single line",
		node.CodeWriteContextFlagFormatParemValuesInNewLine)
	clear("no-decl-item-in-new-line", "keep declaration items on a single line",
		node.CodeWriteContextFlagFormatDeclItemInNewLine)

	fs.BoolVar(&o.transpileOn, "transpile", false,
		"transpile templates to Gad write(...) calls; a .gadt file is saved as .gad")
	registerTranspileFlags(fs, o)
}

// finalizeTranspile turns on transpile mode when requested (the `--transpile`
// flag or any `--transpile-*` option) and fills the TranspileOptions function
// names that were not set explicitly with runnable defaults.
func (o *fmtOptions) finalizeTranspile() {
	if o.transpileOn {
		o.transpileSet = true
	}
	// The TranspileOptions function names are shared; fill the defaults whenever
	// transpile is active anywhere (the global flags or any input_dir).
	active := o.transpileSet
	for _, d := range o.inputDirs {
		active = active || d.Transpile
	}
	if !active {
		return
	}
	if o.transpile.WriteFunc == "" {
		o.transpile.WriteFunc = "write"
	}
	if o.transpile.RawStrFuncStart == "" {
		o.transpile.RawStrFuncStart = "raw "
	}
	// RawStrFuncEnd has no default (the `raw "…"` operator needs no suffix).
}

// registerTranspileFlags registers one --transpile-NAME string flag per string
// field of node.TranspileOptions (NAME is the field name kebab-cased). Setting
// any of them enables transpile mode.
func registerTranspileFlags(fs *flag.FlagSet, o *fmtOptions) {
	tv := reflect.ValueOf(&o.transpile).Elem()
	tt := tv.Type()
	for i := 0; i < tt.NumField(); i++ {
		f := tt.Field(i)
		if f.Type.Kind() != reflect.String {
			continue
		}
		field := tv.Field(i)
		fs.Func("transpile-"+camelToKebab(f.Name), "set TranspileOptions."+f.Name, func(v string) error {
			field.SetString(v)
			o.transpileSet = true
			return nil
		})
	}
}

// camelToKebab converts a CamelCase identifier to kebab-case, e.g.
// "RawStrFuncStart" -> "raw-str-func-start".
func camelToKebab(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// loadConfig reads the YAML config file (unless --no-config) and applies its
// values to flags that were not set on the command line; input_dirs is decoded
// into o.inputDirs. The default config file is silently skipped when missing.
func (o *fmtOptions) loadConfig(fs *flag.FlagSet) error {
	if o.noConfig {
		return nil
	}

	explicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			explicit = true
		}
	})

	path := o.config
	if path == "" {
		path = defaultCfgFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return nil
		}
		return err
	}

	var top map[string]any
	if err = yaml.Unmarshal(data, &top); err != nil {
		return fmt.Errorf("config %s: %w", path, err)
	}

	section, ok := top[fmtConfigKey]
	if !ok {
		return nil // no fmt section
	}
	raw, ok := section.(map[string]any)
	if !ok {
		return fmt.Errorf("config %s: %q must be a mapping", path, fmtConfigKey)
	}

	if v, ok := raw["input_dirs"]; ok {
		b, _ := yaml.Marshal(v)
		if err = yaml.Unmarshal(b, &o.inputDirs); err != nil {
			return fmt.Errorf("config %s: input_dirs: %w", path, err)
		}
		delete(raw, "input_dirs")
	}

	setOnCLI := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setOnCLI[f.Name] = true })

	for k, v := range raw {
		if setOnCLI[k] {
			continue // command-line value wins
		}
		if fs.Lookup(k) == nil {
			return fmt.Errorf("config %s: unknown key %q", path, k)
		}
		if err = setFlagFromConfig(fs, k, v); err != nil {
			return fmt.Errorf("config %s: key %q: %w", path, k, err)
		}
	}
	return nil
}

// setFlagFromConfig applies a YAML value to a flag, expanding list values into
// repeated Set calls (for repeatable flags like --exclude/--include).
func setFlagFromConfig(fs *flag.FlagSet, name string, v any) error {
	if list, ok := v.([]any); ok {
		for _, e := range list {
			if err := fs.Set(name, fmt.Sprint(e)); err != nil {
				return err
			}
		}
		return nil
	}
	return fs.Set(name, fmt.Sprint(v))
}

// fmtTarget is a single file to format together with the metadata needed to
// route its output and backup.
type fmtTarget struct {
	path         string // file path ("" when fromStdin)
	root         string // input root for --out relative paths ("" for explicit files)
	backup       bool
	backupFormat string
	fromStdin    bool
	transpile    bool // transpile this file (and save .gadt as .gad)
}

// relPath returns the target path relative to its input root, used to mirror
// directory structure under --out.
func (t fmtTarget) relPath() string {
	if t.root != "" {
		if rel, err := filepath.Rel(t.root, t.path); err == nil {
			return rel
		}
	}
	return filepath.Base(t.path)
}

// destBase returns the destination file name for path, mapping a transpiled
// `.gadt` template to its `.gad` output (so the template is not overwritten).
func (t fmtTarget) destBase(path string) string {
	if t.transpile && strings.HasSuffix(path, ".gadt") {
		return strings.TrimSuffix(path, "t") // .gadt -> .gad
	}
	return path
}

// fmtJob is a unit of parallel work: one explicit file/stdin, or all files of a
// single input directory.
type fmtJob struct {
	targets []fmtTarget
	dir     string // input directory ("" for explicit files / stdin)
	report  string // per-directory report path ("" when none)
}

// run builds the jobs from the args + config input_dirs and formats them, with
// up to o.jobs jobs running concurrently. A failing file does not stop the
// others: every target is attempted, errors are reported to stderr, and a
// gofmt-style exit code 2 is signalled when anything failed.
func (o *fmtOptions) run(ctx *cc.CommandContext) error {
	jobs, err := o.buildJobs(ctx.Args)
	if err != nil {
		return err
	}

	total := 0
	for ji := range jobs {
		total += len(jobs[ji].targets)
	}
	if total == 0 {
		return nil
	}
	outIsFile := o.out != "" && total == 1

	sink, err := o.newReportSink(ctx.Out)
	if err != nil {
		return err
	}

	limit := o.jobs
	if limit < 1 {
		limit = 1
	}

	var (
		mu         sync.Mutex // guards ctx.Out and reportErrs
		wg         sync.WaitGroup
		sem        = make(chan struct{}, limit)
		records    = make([][]fmtReportRecord, len(jobs))
		reportErrs []error
	)

	for i, j := range jobs {
		i, j := i, j
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			recs := make([]fmtReportRecord, len(j.targets))
			for k, t := range j.targets {
				rec := fmtReportRecord{display: t.displayName()}
				if t.root != "" {
					rec.InputDir = t.root
					rec.File = t.relPath()
				} else {
					rec.File = t.displayName()
				}
				formatted, ferr := o.formatTarget(t, outIsFile, &mu, ctx.Out)
				if ferr != nil {
					rec.Error = ferr.Error()
				} else if o.reportContents {
					rec.Result = formatted
				}
				recs[k] = rec
				sink.emit(rec) // streams now, or buffers for the end
			}
			records[i] = recs

			if j.report != "" {
				if werr := writeReport(j.report, recs); werr != nil {
					mu.Lock()
					reportErrs = append(reportErrs, fmt.Errorf("report %s: %w", j.report, werr))
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	// report per-file failures to stderr (in job order)
	failures := 0
	for i := range jobs {
		for _, r := range records[i] {
			if r.Error != "" {
				failures++
				fmt.Fprintf(ctx.Err, "%s: %s\n", r.display, r.Error)
			}
		}
	}

	// flush the global report (no-op in stream mode, where it was written live)
	if werr := sink.finish(); werr != nil {
		reportErrs = append(reportErrs, fmt.Errorf("report %s: %w", o.report, werr))
	}

	for _, e := range reportErrs {
		fmt.Fprintln(ctx.Err, e)
	}

	if failures > 0 || len(reportErrs) > 0 {
		return &exitError{code: 2}
	}
	return nil
}

// reportSink is the destination of the global NDJSON report. In stream mode it
// writes each record as it arrives; otherwise it buffers and writes them all
// from finish. w is nil when no report is requested.
type reportSink struct {
	mu     sync.Mutex
	w      io.Writer
	closer io.Closer
	stream bool
	buf    []fmtReportRecord
}

// newReportSink resolves the report destination from the flags: a file
// (--report PATH), stdout (--report - or --report-stream with no --report), or
// none.
func (o *fmtOptions) newReportSink(stdout io.Writer) (*reportSink, error) {
	s := &reportSink{stream: o.reportStream}
	switch {
	case o.report == "-":
		s.w = stdout
	case o.report != "":
		if dir := filepath.Dir(o.report); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
		f, err := os.Create(o.report)
		if err != nil {
			return nil, err
		}
		s.w, s.closer = f, f
	case o.reportStream:
		s.w = stdout
	}
	return s, nil
}

func (s *reportSink) emit(r fmtReportRecord) {
	if s.w == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stream {
		s.w.Write(marshalReportLine(r))
	} else {
		s.buf = append(s.buf, r)
	}
}

func (s *reportSink) finish() error {
	if s.w != nil && !s.stream {
		for _, r := range s.buf {
			if _, err := s.w.Write(marshalReportLine(r)); err != nil {
				return err
			}
		}
	}
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// marshalReportLine encodes one record as a single-line JSON object terminated
// by a newline (NDJSON).
func marshalReportLine(r fmtReportRecord) []byte {
	data, err := json.Marshal(r)
	if err != nil {
		// fmtReportRecord is plain strings; marshalling cannot fail in practice.
		data = []byte(fmt.Sprintf(`{"file":%q,"error":%q}`, r.File, err.Error()))
	}
	return append(data, '\n')
}

// writeReport writes the records as NDJSON to path, creating parent directories
// as needed.
func writeReport(path string, records []fmtReportRecord) error {
	var buf bytes.Buffer
	for _, r := range records {
		buf.Write(marshalReportLine(r))
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// exitError carries a process exit code from a subcommand up to main, which
// turns it into os.Exit(code). It is used by `fmt` to signal gofmt-style codes.
type exitError struct {
	code int
}

func (e *exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

func (t fmtTarget) displayName() string {
	if t.fromStdin {
		return "(stdin)"
	}
	return t.path
}

// buildJobs resolves positional args and config input_dirs into jobs.
func (o *fmtOptions) buildJobs(args []string) ([]fmtJob, error) {
	var jobs []fmtJob

	for _, arg := range args {
		if arg == "-" {
			jobs = append(jobs, fmtJob{targets: []fmtTarget{{fromStdin: true}}})
			continue
		}

		recursive, path := splitRecursive(arg)
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			jobs = append(jobs, fmtJob{targets: []fmtTarget{{
				path: path, backup: o.backup, backupFormat: o.backupFormat,
				transpile: o.transpileSet,
			}}})
			continue
		}

		files, err := scanDir(path, recursive, o.filter())
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, fmtJob{
			targets: dirTargets(path, files, o.backup, o.backupFormat, o.transpileSet),
			dir:     path,
		})
	}

	for _, d := range o.inputDirs {
		recursive, path := splitRecursive(d.Path)
		filter, err := o.dirFilter(d)
		if err != nil {
			return nil, err
		}
		files, err := scanDir(path, recursive, filter)
		if err != nil {
			return nil, err
		}
		bf := d.BackupFormat
		if bf == "" {
			bf = o.backupFormat
		}
		jobs = append(jobs, fmtJob{
			targets: dirTargets(path, files, d.Backup, bf, o.transpileSet || d.Transpile),
			dir:     path,
			report:  d.Report,
		})
	}

	return jobs, nil
}

// filter returns the fileFilter built from the command-line/global glob and
// regex lists.
func (o *fmtOptions) filter() *fileFilter {
	return &fileFilter{
		includeGlobs: o.include,
		excludeGlobs: o.exclude,
		includeRe:    o.includeRe,
		excludeRe:    o.excludeRe,
	}
}

// dirFilter builds the fileFilter for a config input directory: the global
// command-line/config globs and regexes merged with the directory's own.
func (o *fmtOptions) dirFilter(d inputDir) (*fileFilter, error) {
	f := &fileFilter{
		includeGlobs: concatGlobs(o.include, d.Includes),
		excludeGlobs: concatGlobs(o.exclude, d.Excludes),
		includeRe:    append(reList{}, o.includeRe...),
		excludeRe:    append(reList{}, o.excludeRe...),
	}
	for _, p := range d.IncludesRe {
		if err := f.includeRe.Set(p); err != nil {
			return nil, err
		}
	}
	for _, p := range d.ExcludesRe {
		if err := f.excludeRe.Set(p); err != nil {
			return nil, err
		}
	}
	return f, nil
}

func concatGlobs(base globList, extra []string) globList {
	out := append(globList{}, base...)
	return append(out, extra...)
}

func dirTargets(root string, files []string, backup bool, backupFormat string, transpile bool) []fmtTarget {
	targets := make([]fmtTarget, len(files))
	for i, f := range files {
		targets[i] = fmtTarget{path: f, root: root, backup: backup, backupFormat: backupFormat, transpile: transpile}
	}
	return targets
}

// isGadSource reports whether name is a formattable Gad source file (.gad or
// the template variant .gadt).
func isGadSource(name string) bool {
	return strings.HasSuffix(name, ".gad") || strings.HasSuffix(name, ".gadt")
}

// splitRecursive strips a trailing "/..." (or a lone "...") recursion marker,
// returning whether recursion was requested and the cleaned path.
func splitRecursive(arg string) (recursive bool, path string) {
	switch {
	case arg == "...":
		return true, "."
	case strings.HasSuffix(arg, "/..."):
		return true, strings.TrimSuffix(arg, "/...")
	default:
		if arg == "" {
			return false, "."
		}
		return false, arg
	}
}

// scanDir returns the .gad files of dir (recursively when requested), skipping
// hidden files and directories and applying the file filter.
func scanDir(dir string, recursive bool, filter *fileFilter) (files []string, err error) {
	if recursive {
		err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			name := d.Name()
			if d.IsDir() {
				if p != dir && isHidden(name) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isHidden(name) && isGadSource(name) && filter.match(p) {
				files = append(files, p)
			}
			return nil
		})
		return files, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, d := range entries {
		name := d.Name()
		if d.IsDir() || isHidden(name) || !isGadSource(name) {
			continue
		}
		p := filepath.Join(dir, name)
		if filter.match(p) {
			files = append(files, p)
		}
	}
	return files, nil
}

// collectFmtTargets is a flat helper (used in tests) that resolves args into the
// list of file paths that would be formatted, applying the given filter.
func collectFmtTargets(args []string, filter *fileFilter) ([]string, error) {
	o := &fmtOptions{
		exclude:   filter.excludeGlobs,
		include:   filter.includeGlobs,
		excludeRe: filter.excludeRe,
		includeRe: filter.includeRe,
	}
	jobs, err := o.buildJobs(args)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, j := range jobs {
		for _, t := range j.targets {
			out = append(out, t.path)
		}
	}
	return out, nil
}

// isHidden reports whether a path component is hidden (starts with a dot).
func isHidden(name string) bool {
	return strings.HasPrefix(name, ".") && name != "." && name != ".."
}

func matchAnyGlob(globs globList, candidates ...string) bool {
	for _, g := range globs {
		for _, c := range candidates {
			if ok, _ := filepath.Match(g, c); ok {
				return true
			}
		}
	}
	return false
}

func matchAnyRe(res reList, candidates ...string) bool {
	for _, re := range res {
		for _, c := range candidates {
			if re.MatchString(c) {
				return true
			}
		}
	}
	return false
}

// formatSource parses src and returns its formatted form using the configured
// code flags and transpile options. name is used for error positions.
func (o *fmtOptions) formatSource(name string, src []byte, transpile bool) (string, error) {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(name, -1, src)

	// `.gadt` files are templates: parse them in mixed mode (the `# gad: …`
	// config directives are disabled since the file is template from byte 0).
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	var so *parser.ScannerOptions
	if strings.HasSuffix(name, ".gadt") {
		po.Mode |= parser.ParseMixed
		so = &parser.ScannerOptions{Mode: parser.ScanMixed | parser.ScanConfigDisabled}
	}
	file, err := parser.NewParserWithOptions(srcFile, po, so).ParseFile()
	if err != nil {
		return "", err
	}

	opts := []node.CodeOption{
		node.CodeWithFlags(o.codeFlags),
		node.CodeWithPrefix("\t"),
		node.CodeWithComments(srcFile, file.Comments),
	}
	if transpile {
		opts = append(opts, node.CodeTranspile(&o.transpile))
	}

	out := node.Code(file.Stmts, opts...)
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return out, nil
}

// formatTarget formats a single target and writes the result to its
// destination: stdout for stdin, the --out location when set, or the input file
// in place otherwise.
func (o *fmtOptions) formatTarget(t fmtTarget, outIsFile bool, mu *sync.Mutex, stdout io.Writer) (formatted string, err error) {
	var src []byte
	if t.fromStdin {
		src, err = io.ReadAll(os.Stdin)
	} else {
		src, err = os.ReadFile(t.path)
	}
	if err != nil {
		return "", err
	}

	formatted, err = o.formatSource(t.displayName(), src, t.transpile)
	if err != nil {
		return "", err
	}

	// --no-save is read-only: format and report only, write nothing.
	if o.noSave {
		return formatted, nil
	}

	switch {
	case t.fromStdin:
		mu.Lock()
		_, err = io.WriteString(stdout, formatted)
		mu.Unlock()
		return formatted, err

	case o.out != "":
		dest := o.out
		if !outIsFile {
			dest = t.destBase(filepath.Join(o.out, t.relPath()))
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return formatted, err
		}
		return formatted, os.WriteFile(dest, []byte(formatted), 0o644)

	default:
		dest := t.destBase(t.path)
		// In place, skip a no-op rewrite; a transpiled .gadt -> .gad always writes.
		if dest == t.path && string(src) == formatted {
			return formatted, nil // already formatted
		}
		if t.backup {
			if err = writeBackup(t, src); err != nil {
				return formatted, err
			}
		}
		mode := os.FileMode(0o644)
		if info, statErr := os.Stat(t.path); statErr == nil {
			mode = info.Mode().Perm()
		}
		if err = os.WriteFile(dest, []byte(formatted), mode); err != nil {
			return formatted, err
		}
		// echo the written path, unless the report is itself going to stdout
		if !o.reportToStdout() {
			mu.Lock()
			fmt.Fprintln(stdout, dest)
			mu.Unlock()
		}
		return formatted, nil
	}
}

// reportToStdout reports whether the NDJSON report is written to stdout.
func (o *fmtOptions) reportToStdout() bool {
	return o.report == "-" || (o.reportStream && o.report == "")
}

// writeBackup saves the original source next to the target using its
// backup-format pattern (BASE_NAME -> file name without extension).
func writeBackup(t fmtTarget, src []byte) error {
	dir := filepath.Dir(t.path)
	base := filepath.Base(t.path)
	baseNoExt := strings.TrimSuffix(base, filepath.Ext(base))
	name := strings.ReplaceAll(t.backupFormat, "BASE_NAME", baseNoExt)
	backupPath := name
	if !filepath.IsAbs(name) && filepath.Dir(name) == "." {
		backupPath = filepath.Join(dir, name)
	}
	return os.WriteFile(backupPath, src, 0o644)
}
