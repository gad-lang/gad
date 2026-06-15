// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

//go:build !js
// +build !js

package main

import (
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
	defaultCfgFile        = ".gad.yaml"
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

// subcommandNames is the set of first-argument tokens routed through the
// command-context framework instead of the legacy run/REPL entry point.
var subcommandNames = map[string]bool{
	"run":    true,
	"fmt":    true,
	"debug":  true,
	"ide":    true,
	"help":   true,
	"--help": true,
}

// isSubcommand reports whether name selects a command-context subcommand.
func isSubcommand(name string) bool { return subcommandNames[name] }

// buildRootCommand assembles the `gad` command tree: a root that prints help
// plus the `run` and `fmt` subcommands.
func buildRootCommand() *cc.Command {
	root := &cc.Command{
		Name:        "gad",
		Description: "Gad scripting language CLI.",
		Run: func(ctx *cc.CommandContext) error {
			return ctx.Help()
		},
	}
	root.Sub(runCommand())
	root.Sub(fmtCommand())
	root.Sub(debugCommand())
	root.Sub(ideCommand())
	return root
}

// runCommand is the explicit `gad run [flags] [FILE [ARGS...]]` subcommand. It
// shares all behavior with the legacy bare invocation.
func runCommand() *cc.Command {
	return &cc.Command{
		Name:        "run",
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
	ReportFormat string   `yaml:"report_format"`
}

// fmtReportFile is the per-file outcome recorded in a format report. Error is
// nil on success or the error message string on failure.
type fmtReportFile struct {
	Path  string `yaml:"path" json:"path"`
	Error any    `yaml:"error" json:"error"`
}

// fmtReportDir groups the per-file outcomes of a single input directory.
type fmtReportDir struct {
	Path  string          `yaml:"path" json:"path"`
	Files []fmtReportFile `yaml:"files" json:"files"`
}

// fmtReport is the document written by --report: explicit files plus the files
// grouped per input directory.
type fmtReport struct {
	Files     []fmtReportFile `yaml:"files" json:"files"`
	InputDirs []fmtReportDir  `yaml:"input_dirs" json:"input_dirs"`
}

// fmtOptions holds the parsed flags (and config) of the `fmt` subcommand.
type fmtOptions struct {
	exclude      globList
	include      globList
	excludeRe    reList
	includeRe    reList
	backup       bool
	backupFormat string
	codeFlags    node.CodeWriteContextFlag
	transpile    node.TranspileOptions
	transpileSet bool
	jobs         int
	out          string
	report       string
	reportFormat string
	config       string
	noConfig     bool
	inputDirs    []inputDir
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
			if err := validateReportFormat(o.reportFormat); err != nil {
				return err
			}
			for _, d := range o.inputDirs {
				if err := validateReportFormat(d.ReportFormat); err != nil {
					return fmt.Errorf("input_dir %q: %w", d.Path, err)
				}
			}
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
	fs.StringVar(&o.report, "report", "", "write a per-file status report to this path")
	fs.StringVar(&o.reportFormat, "report-format", reportFmtYAML, "report file format: yaml or json")
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

	registerTranspileFlags(fs, o)
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

// fmtJob is a unit of parallel work: one explicit file/stdin, or all files of a
// single input directory.
type fmtJob struct {
	targets      []fmtTarget
	dir          string // input directory ("" for explicit files / stdin)
	report       string // per-directory report path ("" when none)
	reportFormat string // per-directory report format
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
	for _, j := range jobs {
		total += len(j.targets)
	}
	if total == 0 {
		return nil
	}
	outIsFile := o.out != "" && total == 1

	limit := o.jobs
	if limit < 1 {
		limit = 1
	}

	var (
		mu         sync.Mutex // guards ctx.Out and reportErrs
		wg         sync.WaitGroup
		sem        = make(chan struct{}, limit)
		outcomes   = make([]fmtReportDir, len(jobs))
		reportErrs []error
	)

	for i, j := range jobs {
		i, j := i, j
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			files := make([]fmtReportFile, len(j.targets))
			for k, t := range j.targets {
				files[k] = fmtReportFile{Path: t.displayName()}
				if err := o.formatTarget(t, outIsFile, &mu, ctx.Out); err != nil {
					files[k].Error = err.Error()
				}
			}
			outcomes[i] = fmtReportDir{Path: j.dir, Files: files}

			if j.report != "" {
				if werr := writeReport(j.report, j.reportFormat, outcomes[i]); werr != nil {
					mu.Lock()
					reportErrs = append(reportErrs, fmt.Errorf("report %s: %w", j.report, werr))
					mu.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	// Assemble the global report and report per-file failures.
	var report fmtReport
	failures := 0
	for i, j := range jobs {
		oc := outcomes[i]
		for _, f := range oc.Files {
			if f.Error != nil {
				failures++
				fmt.Fprintf(ctx.Err, "%s: %v\n", f.Path, f.Error)
			}
		}
		if j.dir == "" {
			report.Files = append(report.Files, oc.Files...)
		} else {
			report.InputDirs = append(report.InputDirs, oc)
		}
	}

	if o.report != "" {
		if werr := writeReport(o.report, o.reportFormat, report); werr != nil {
			reportErrs = append(reportErrs, fmt.Errorf("report %s: %w", o.report, werr))
		}
	}
	for _, e := range reportErrs {
		fmt.Fprintln(ctx.Err, e)
	}

	if failures > 0 || len(reportErrs) > 0 {
		return &exitError{code: 2}
	}
	return nil
}

// Report file formats.
const (
	reportFmtYAML = "yaml"
	reportFmtJSON = "json"
)

// writeReport marshals v in the given format (yaml or json) and writes it to
// path, creating parent directories as needed. An empty format defaults to
// yaml.
func writeReport(path, format string, v any) error {
	data, err := marshalReport(format, v)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

// validateReportFormat accepts an empty value (defaults to yaml), "yaml" or
// "json".
func validateReportFormat(format string) error {
	switch strings.ToLower(format) {
	case "", reportFmtYAML, reportFmtJSON:
		return nil
	default:
		return fmt.Errorf("invalid report-format %q (want yaml or json)", format)
	}
}

// marshalReport encodes v as yaml or json. An empty or unknown format defaults
// to yaml.
func marshalReport(format string, v any) ([]byte, error) {
	switch strings.ToLower(format) {
	case reportFmtJSON:
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	default:
		return yaml.Marshal(v)
	}
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
			}}})
			continue
		}

		files, err := scanDir(path, recursive, o.filter())
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, fmtJob{
			targets: dirTargets(path, files, o.backup, o.backupFormat),
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
		rf := d.ReportFormat
		if rf == "" {
			rf = o.reportFormat
		}
		jobs = append(jobs, fmtJob{
			targets:      dirTargets(path, files, d.Backup, bf),
			dir:          path,
			report:       d.Report,
			reportFormat: rf,
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

func dirTargets(root string, files []string, backup bool, backupFormat string) []fmtTarget {
	targets := make([]fmtTarget, len(files))
	for i, f := range files {
		targets[i] = fmtTarget{path: f, root: root, backup: backup, backupFormat: backupFormat}
	}
	return targets
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
			if !isHidden(name) && strings.HasSuffix(name, ".gad") && filter.match(p) {
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
		if d.IsDir() || isHidden(name) || !strings.HasSuffix(name, ".gad") {
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
func (o *fmtOptions) formatSource(name string, src []byte) (string, error) {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(name, -1, src)
	file, err := parser.NewParserWithOptions(srcFile, nil, nil).ParseFile()
	if err != nil {
		return "", err
	}

	opts := []node.CodeOption{
		node.CodeWithFlags(o.codeFlags),
		node.CodeWithPrefix("\t"),
	}
	if o.transpileSet {
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
func (o *fmtOptions) formatTarget(t fmtTarget, outIsFile bool, mu *sync.Mutex, stdout io.Writer) error {
	var (
		src []byte
		err error
	)
	if t.fromStdin {
		src, err = io.ReadAll(os.Stdin)
	} else {
		src, err = os.ReadFile(t.path)
	}
	if err != nil {
		return err
	}

	formatted, err := o.formatSource(t.displayName(), src)
	if err != nil {
		return err
	}

	switch {
	case t.fromStdin:
		mu.Lock()
		_, err = io.WriteString(stdout, formatted)
		mu.Unlock()
		return err

	case o.out != "":
		dest := o.out
		if !outIsFile {
			dest = filepath.Join(o.out, t.relPath())
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dest, []byte(formatted), 0o644)

	default:
		if string(src) == formatted {
			return nil // already formatted
		}
		if t.backup {
			if err = writeBackup(t, src); err != nil {
				return err
			}
		}
		info, statErr := os.Stat(t.path)
		if statErr != nil {
			return statErr
		}
		if err = os.WriteFile(t.path, []byte(formatted), info.Mode().Perm()); err != nil {
			return err
		}
		mu.Lock()
		fmt.Fprintln(stdout, t.path)
		mu.Unlock()
		return nil
	}
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
