// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad/gadconfig"
	cc "github.com/moisespsena-go/command-context"
	"gopkg.in/yaml.v3"
)

// docInputDir is a directory entry under the config file's `doc.input_dirs`. Its
// dst/skip override the root `doc.dst`/`doc.skip` for sources inside Path.
type docInputDir struct {
	Path     string `yaml:"path"`
	Dst      string `yaml:"dst"`
	Skip     *bool  `yaml:"skip"`
	dstSet   bool   // dst was present in the config
	skip     bool   // resolved skip
	dst      string // resolved dst (absolute)
	resolved bool
}

// docOptions holds the parsed flags (and config) of the `doc` subcommand.
type docOptions struct {
	out       string // --out / root doc.dst (default "doc")
	dstSet    bool   // root doc.dst (or --out) was set
	skip      bool   // root doc.skip
	noSkip    bool   // --no-skip forces skip=false
	noSave    bool   // --no-save: do not write any file
	noDoctest bool   // --no-doctest: skip running embedded examples
	// mustExported (--must-exported / doc.must_exported): document only exported
	// symbols. When false (default) the output also has an "Internal" section.
	mustExported bool
	config       string
	noConfig     bool
	inputDirs    []docInputDir
	workspace    string // WORKSPACE_DIR (config dir, else cwd)

	// docTemplateMD / docTemplateHTML override the workspace doc templates
	// (.gad/doc-templates/{md,html}.gadx). Each PATH may be a .gad, .gadt or
	// .gadx file; the dialect is chosen by extension (.gad/.gadt write to
	// STDOUT, .gadx renders a tag tree).
	docTemplateMD   string
	docTemplateHTML string
	// html enables the extra .html output using the workspace/embedded HTML
	// template. noTemplate disables template rendering entirely (built-in Go
	// Markdown renderer). json/yaml additionally encode the doc structure (the
	// template `param (doc dict)` shape) to a .json / .yaml file per source.
	html       bool
	noTemplate bool
	json       bool
	yaml       bool

	examplesFailed int // count of failed embedded examples

	templates *docTemplateSet // lazily-resolved doc templates

	// written accumulates the Markdown output paths produced in template mode,
	// and indexRoots the dst roots they live under; together they drive the
	// per-directory index generation (README.md / index.html).
	written    []string
	indexRoots map[string]bool
}

const defaultDocOut = "doc"

// docCommand is the `gad doc [flags] PATH...` subcommand. It renders godoc-style
// Markdown from the doc comments of Gad source files. This scaffold resolves the
// flags/config and writes the output tree; the Markdown generation itself is a
// stub (see generateDoc).
func docCommand() *cc.Command {
	return &cc.Command{
		Name:  "doc",
		Usage: "[flags] [PATH...]",
		Description: "Generate Markdown documentation from Gad source files.\n" +
			"\nPATH may be a file or a directory; write DIR/... to recurse. Output is\n" +
			"written under --out (default \"doc\"), mirroring the source tree, unless\n" +
			"--no-save is given.",
		New: func(ctx *cc.CommandContext) error {
			o := &docOptions{out: defaultDocOut}
			o.registerFlags(ctx.Flags())
			ctx.WithValue(docOptionsKey, o)
			return nil
		},
		ParseArgs: func(ctx *cc.CommandContext) error {
			o := ctx.Value(docOptionsKey).(*docOptions)
			if err := o.loadConfig(ctx.Flags()); err != nil {
				return err
			}
			// Default to the current directory (recursive) so `gad doc` run
			// inside any workspace dir generates docs without explicit PATH args.
			if len(ctx.Args) == 0 && len(o.inputDirs) == 0 {
				ctx.Args = cc.Args{"..."}
			}
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			o := ctx.Value(docOptionsKey).(*docOptions)
			return o.run(ctx)
		},
	}
}

// registerFlags registers the `doc` flags on fs, bound to o.
func (o *docOptions) registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.out, "out", defaultDocOut, "output directory (root doc.dst)")
	fs.BoolVar(&o.noSkip, "no-skip", false, "force doc.skip to false")
	fs.BoolVar(&o.noSave, "no-save", false, "do not write any file (render and report only)")
	fs.BoolVar(&o.noDoctest, "no-doctest", false, "do not run the ```gad examples embedded in doc comments")
	fs.BoolVar(&o.mustExported, "must-exported", false, "document only exported symbols (omit the Internal section)")
	fs.StringVar(&o.config, "config", "", "YAML config file with default flag values (default WORKDIR/.gad/gad.yaml)")
	fs.BoolVar(&o.noConfig, "no-config", false, "do not read the config file")
	fs.StringVar(&o.docTemplateMD, "doc-template-md", "", "Markdown doc template (.gad/.gadt/.gadx); overrides .gad/doc-templates/md.gadx")
	fs.StringVar(&o.docTemplateHTML, "doc-template-html", "", "HTML doc template (.gad/.gadt/.gadx); overrides .gad/doc-templates/html.gadx")
	fs.BoolVar(&o.html, "html", false, "also emit an .html file per source using the HTML doc template")
	fs.BoolVar(&o.noTemplate, "no-template", false, "disable doc templates; use the built-in Markdown renderer")
	fs.BoolVar(&o.json, "json", false, "also emit a .json file per source encoding the doc structure")
	fs.BoolVar(&o.yaml, "yaml", false, "also emit a .yaml file per source encoding the doc structure")
}

// loadConfig reads the `doc:` section of the YAML config (unless --no-config) and
// applies its values to options not set on the command line.
func (o *docOptions) loadConfig(fs *flag.FlagSet) error {
	setOnCLI := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { setOnCLI[f.Name] = true })
	o.dstSet = setOnCLI["out"]

	// The workspace root (WORK_DIR) drives config, templates and output
	// locations; config lives at WORK_DIR/.gad/gad.yaml (or $GAD_CONFIG_DIR).
	o.workspace = gadconfig.WorkDir("")

	if o.noConfig {
		o.finalize()
		return nil
	}

	explicit := setOnCLI["config"]
	path := o.config
	if path == "" {
		path = gadconfig.File(o.workspace)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			o.finalize()
			return nil
		}
		return err
	}

	var top map[string]any
	if err = yaml.Unmarshal(data, &top); err != nil {
		return fmt.Errorf("config %s: %w", path, err)
	}
	section, ok := top[docConfigKey]
	if !ok {
		o.finalize()
		return nil
	}
	var cfg struct {
		Dst          string        `yaml:"dst"`
		Skip         bool          `yaml:"skip"`
		MustExported bool          `yaml:"must_exported"`
		InputDirs    []docInputDir `yaml:"input_dirs"`
	}
	b, _ := yaml.Marshal(section)
	if err = yaml.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("config %s: %q: %w", path, docConfigKey, err)
	}

	if !setOnCLI["out"] && cfg.Dst != "" {
		o.out = cfg.Dst
		o.dstSet = true
	}
	o.skip = cfg.Skip
	if !setOnCLI["must-exported"] {
		o.mustExported = cfg.MustExported
	}
	o.inputDirs = cfg.InputDirs
	o.finalize()
	return nil
}

// finalize applies --no-skip and resolves the root dst to an absolute path.
func (o *docOptions) finalize() {
	if o.noSkip {
		o.skip = false
	}
	if o.out != "" {
		o.out = o.absFrom(o.workspace, o.out)
		o.dstSet = true
	}
}

// absFrom resolves p against base when p is relative.
func (o *docOptions) absFrom(base, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	abs, err := filepath.Abs(filepath.Join(base, p))
	if err != nil {
		return filepath.Clean(filepath.Join(base, p))
	}
	return abs
}

// resolveDir fills d.dst/d.skip from the root settings per the rules: a per-dir
// dst is relative to the INPUT_DIR path; it defaults to the root dst only when
// the root dst is not absolute; skip defaults to the root skip.
func (o *docOptions) resolveDir(d *docInputDir) {
	if d.resolved {
		return
	}
	_, dirPath := splitRecursive(d.Path)
	// The input dir itself is relative to the workspace; a per-dir dst is then
	// relative to that absolute input dir.
	absDir := o.absFrom(o.workspace, dirPath)
	if d.Skip != nil {
		d.skip = *d.Skip
	} else {
		d.skip = o.skip
	}
	if o.noSkip {
		d.skip = false
	}
	switch {
	case d.Dst != "":
		d.dstSet = true
		d.dst = o.absFrom(absDir, d.Dst)
	case o.dstSet && !filepath.IsAbs(o.rawOut()):
		d.dstSet = true
		d.dst = o.out // already absolute
	default:
		d.dstSet = false
	}
	d.resolved = true
}

// rawOut returns the dst as configured on the CLI/config before absolutization.
// finalize() makes o.out absolute, so we recover the original via the workspace.
func (o *docOptions) rawOut() string {
	if rel, err := filepath.Rel(o.workspace, o.out); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return o.out
}

// run renders the documentation for the positional args and the config input
// dirs.
func (o *docOptions) run(ctx *cc.CommandContext) error {
	o.indexRoots = map[string]bool{}

	// Positional (non-INPUT_DIR) sources honour the root skip.
	if !o.skip {
		for _, arg := range ctx.Args {
			o.indexRoots[o.out] = true
			if err := o.processArg(ctx, arg, o.out, o.workspace); err != nil {
				return err
			}
		}
	} else if len(ctx.Args) > 0 {
		o.logSkip(ctx, ".", "doc.skip is set")
	}

	for i := range o.inputDirs {
		d := &o.inputDirs[i]
		o.resolveDir(d)
		_, dirPath := splitRecursive(d.Path)
		if d.skip {
			o.logSkip(ctx, dirPath, "doc.skip is set")
			continue
		}
		if !d.dstSet || d.dst == "" {
			o.logSkip(ctx, dirPath, "no doc.dst")
			continue
		}
		if d.dst == o.out {
			return fmt.Errorf("doc: input_dir %q dst resolves to the root doc.dst %q", d.Path, o.out)
		}
		o.indexRoots[d.dst] = true
		if err := o.processArg(ctx, d.Path, d.dst, dirPath); err != nil {
			return err
		}
	}

	// Per-directory indexes (README.md / index.html) are generated once all the
	// per-file docs are written, in template mode only.
	if tset := o.resolveDocTemplates(); tset.any() && !o.noSave {
		if err := o.generateIndexes(ctx, tset); err != nil {
			return err
		}
	}

	if o.examplesFailed > 0 {
		return fmt.Errorf("doc: %d embedded example(s) failed", o.examplesFailed)
	}
	return nil
}

// processArg renders the file or directory arg, writing each output under dst
// mirroring the tree rooted at base.
func (o *docOptions) processArg(ctx *cc.CommandContext, arg, dst, base string) error {
	recursive, path := splitRecursive(arg)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return o.processFile(ctx, path, dst, filepath.Dir(path))
	}
	files, err := scanDir(path, recursive, &fileFilter{})
	if err != nil {
		return err
	}
	for _, f := range files {
		if err = o.processFile(ctx, f, dst, base); err != nil {
			return err
		}
	}
	return nil
}

// processFile renders a single .gad file to Markdown and writes it under dst,
// preserving its path relative to base. The rendering itself (and the output
// path) is done by DocGenerator.FromFile, which performs no file I/O; this method
// owns the read and the write.
func (o *docOptions) processFile(ctx *cc.CommandContext, path, dst, base string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	gen := &DocGenerator{
		OnError:      func(msg string) { fmt.Fprintln(ctx.Err, msg) },
		MustExported: o.mustExported,
		NoTest:       o.noDoctest,
	}
	res, err := gen.FromFile(src, path, dst, base)
	if err != nil {
		return err
	}
	o.examplesFailed += res.ExamplesFailed

	// Optional gadx templates in the workspace customize the rendered output:
	// .gaddoc-md.gadx overrides the built-in Markdown; .gaddoc.gadx emits HTML
	// alongside it. Both receive the extracted docs as `param (doc dict)`.
	outputs := []docOutput{{res.OutPath, res.Markdown}}
	if tset := o.resolveDocTemplates(); tset.any() {
		if outputs, err = o.renderTemplateOutputs(path, src, res, tset); err != nil {
			return err
		}
		// Record the Markdown output path so run() can build the directory
		// indexes once every file is written.
		o.written = append(o.written, res.OutPath)
	}

	// --json / --yaml additionally encode the doc structure alongside the
	// Markdown/HTML outputs.
	if o.json || o.yaml {
		encoded, err := o.renderEncodedOutputs(path, src, res)
		if err != nil {
			return err
		}
		outputs = append(outputs, encoded...)
	}

	if o.noSave {
		for _, out := range outputs {
			fmt.Fprintln(ctx.Out, out.path)
		}
		return nil
	}
	for _, out := range outputs {
		if err = os.MkdirAll(filepath.Dir(out.path), 0o755); err != nil {
			return err
		}
		if err = os.WriteFile(out.path, []byte(out.body), 0o644); err != nil {
			return err
		}
		fmt.Fprintln(ctx.Out, out.path)
	}
	return nil
}

// logSkip writes a skip notice to stderr (coloured when stderr is a terminal).
func (o *docOptions) logSkip(ctx *cc.CommandContext, what, why string) {
	msg := fmt.Sprintf("doc: skipping %s: %s", what, why)
	if f, ok := ctx.Err.(*os.File); ok && isTerminal(f) {
		msg = "\x1b[33m" + msg + "\x1b[0m"
	}
	fmt.Fprintln(ctx.Err, msg)
}

// isTerminal reports whether f is a character device (a terminal).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
