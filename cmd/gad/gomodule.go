package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gad-lang/gad/gomodule"
	"github.com/gad-lang/gad/stdlib/helper"
	cc "github.com/moisespsena-go/command-context"
)

// `gad gomodule` (issue #17): a module written in gad made a builtin module in
// Go — a package that embeds the module compiled (`.gadb`), documents it with
// `gad:doc` comments (its API stub, `gaddoc api`'s, written beside it) and has
// a hook for a Go init of its own (`Init`, set by a hand-written file).

// gomoduleOpts are the options of `gad gomodule`.
type gomoduleOpts struct {
	pkg, module, out, api string
	imports               []string
}

func gomoduleCommand() *cc.Command {
	var (
		o       gomoduleOpts
		imports string
	)
	return &cc.Command{
		Name:  "gomodule",
		Usage: "[flags] MODULE.gad",
		Description: "Make a gad module a builtin module in Go (issue #17).\n\n" +
			"Writes into --out (a Go package): MODULE.gadb (the module compiled, linked\n" +
			"into the programs that import it), MODULE.go (the package: ModuleName, the\n" +
			"embedded bytecode, Module() — the importable —, AddTo, Imports, the Init\n" +
			"hook, and the module's docs as gad:doc comments), MODULE_test.go (links it\n" +
			"against the default builtins) and the API stub (--api, `gaddoc api`'s).\n\n" +
			"A Go init of the module — a hand-written file of the package setting Init —\n" +
			"runs after the gad part; the modules it imports are declared: --import, or\n" +
			"`//gad:import NAME[,NAME]` lines in the package's Go files.",
		New: func(ctx *cc.CommandContext) error {
			f := ctx.Flags()
			f.StringVar(&o.pkg, "pkg", "", "Go package name (default: the module name)")
			f.StringVar(&o.module, "module", "", "module name (default: the file name)")
			f.StringVar(&o.out, "out", ".", "output directory (the Go package)")
			f.StringVar(&o.api, "api", "", "API stub file (default: OUT/MODULE.gad)")
			f.StringVar(&imports, "import", "", "modules the Go init imports, comma separated")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			if len(ctx.Args) != 1 {
				return fmt.Errorf("usage: gad gomodule [flags] MODULE.gad")
			}
			for _, n := range strings.Split(imports, ",") {
				if n = strings.TrimSpace(n); n != "" {
					o.imports = append(o.imports, n)
				}
			}
			return runGomodule(ctx.Args[0], o)
		},
	}
}

func runGomodule(srcPath string, o gomoduleOpts) error {
	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	base := filepath.Base(srcPath)
	ext := filepath.Ext(base)
	sourceType := strings.TrimPrefix(ext, ".")
	if o.module == "" {
		o.module = strings.TrimSuffix(base, ext)
	}
	if o.api == "" {
		o.api = filepath.Join(o.out, o.module+".gad")
	}
	if err = os.MkdirAll(o.out, 0o755); err != nil {
		return err
	}

	// the hand-written Go files of the package (the Go init)
	files := map[string][]byte{}
	entries, err := os.ReadDir(o.out)
	if err != nil {
		return err
	}
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || n == o.module+".go" || n == o.module+"_test.go" {
			continue
		}
		if files[n], err = os.ReadFile(filepath.Join(o.out, n)); err != nil {
			return err
		}
	}

	rel, err := filepath.Rel(o.out, srcPath)
	if err != nil {
		rel = srcPath
	}
	pkg := o.pkg
	if pkg == "" {
		pkg = o.module
	}
	gen := gomodule.CmdPath() + " gomodule --pkg " + pkg + " --module " + o.module + " --out ."
	if apiRel, err := filepath.Rel(o.out, o.api); err == nil && apiRel != o.module+".gad" {
		gen += " --api " + filepath.ToSlash(apiRel)
	}
	if len(o.imports) > 0 {
		gen += " --import " + strings.Join(o.imports, ",")
	}
	gen += " " + filepath.ToSlash(rel)

	r, err := gomodule.Make(src, gomodule.Options{
		Module:     o.module,
		Pkg:        pkg,
		SourceType: sourceType,
		Imports:    o.imports,
		Files:      files,
		Generate:   "go run " + gen,
		// the CLI's: every builtin module (the system's too)
		ModuleMap: helper.NewModuleMap(),
	})
	if err != nil {
		return fmt.Errorf("%s: %w", srcPath, err)
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(os.Stderr, "gad gomodule: %s: %s\n", o.module, w)
	}
	out := r.Files()
	api := out[o.module+".gad"]
	delete(out, o.module+".gad")
	names := make([]string, 0, len(out))
	for n := range out {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		p := filepath.Join(o.out, n)
		if err = os.WriteFile(p, out[n], 0o644); err != nil {
			return err
		}
		fmt.Println("wrote", p)
	}
	if err = os.WriteFile(o.api, api, 0o644); err != nil {
		return err
	}
	fmt.Println("wrote", o.api)
	return nil
}
