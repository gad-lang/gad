// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	cc "github.com/moisespsena-go/command-context"
)

// reModuleName matches a valid Gad module / Go package name: a lowercase
// snake_case identifier (per the language's module naming convention).
var reModuleName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// gomodCommand groups the Go-module scaffolding subcommands under `gad gomod`.
func gomodCommand() *cc.Command {
	c := &cc.Command{
		Name:        "gomod",
		Usage:       "<subcommand> [args]",
		Description: "Scaffold and manage Go modules that embed Gad.",
		Run: func(ctx *cc.CommandContext) error {
			return ctx.Help()
		},
	}
	c.Sub(gomodInitCommand())
	return c
}

// gomodInitCommand implements `gad gomod init MODULE_NAME`: it writes a new Gad
// module package (a documented ModuleInit) plus its auto-scaffolded samples.gad,
// ready to be filled in and wired into a ModuleMap.
func gomodInitCommand() *cc.Command {
	var dir string
	return &cc.Command{
		Name:  "init",
		Usage: "[flags] MODULE_NAME",
		Description: "Scaffold a new Go Gad module.\n\n" +
			"Creates MODULE_NAME/module.go (a documented StdModuleData ModuleInit,\n" +
			"with a `gad:doc` header and a `gad:samples` directive) and\n" +
			"MODULE_NAME/samples.gad (one doctest snippet per exported member — the\n" +
			"`gad doc` / gaddoc pipeline merges these into the module's API docs).",
		New: func(ctx *cc.CommandContext) error {
			ctx.Flags().StringVar(&dir, "dir", "", "target directory (default: ./MODULE_NAME)")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			if len(ctx.Args) != 1 {
				return fmt.Errorf("usage: gad gomod init MODULE_NAME")
			}
			name := ctx.Args[0]
			if !reModuleName.MatchString(name) {
				return fmt.Errorf("invalid module name %q: use a lowercase snake_case identifier", name)
			}
			target := dir
			if target == "" {
				target = name
			}
			return scaffoldGoModule(name, target)
		},
	}
}

// scaffoldGoModule writes module.go and samples.gad for a new module named name
// into dir, creating dir as needed. Existing files are never overwritten.
func scaffoldGoModule(name, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	files := map[string]string{
		"module.go":   goModuleTemplate(name),
		"samples.gad": samplesTemplate(name),
	}
	for base, content := range files {
		path := filepath.Join(dir, base)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists (refusing to overwrite)", path)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Printf("created %s\n", path)
	}
	fmt.Printf("\nNext:\n"+
		"  - implement your functions in %s/module.go\n"+
		"  - add usage examples to %s/samples.gad (they are verified by `gad doc`)\n"+
		"  - register it: gad.NewModuleMap().AddBuiltinModuleInit(%q, %s.ModuleInit)\n",
		dir, dir, name, name)
	return nil
}

// goModuleTemplate renders the module.go skeleton for a module named name.
func goModuleTemplate(name string) string {
	return strings.ReplaceAll(`package NAME

import "github.com/gad-lang/gad"

// ModuleName is the import name of the NAME module.
const ModuleName = "NAME"

// ModuleInit initializes the NAME module. Register it with a ModuleMap:
//
//	mm := gad.NewModuleMap().AddBuiltinModuleInit(ModuleName, ModuleInit)
var ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
	spec := module.Spec
	module.Data = gad.StdModuleData{Funcs: gad.Dict{
		// gad:doc
		// # NAME module
		// gad:samples [module,auto] NAME/samples.gad
		//
		// ## Functions
		// hello(name str) <str>
		// Returns a greeting for name.
		"hello": &gad.Function{
			Module:   spec,
			FuncName: "hello",
			Value: func(c gad.Call) (gad.Object, error) {
				if err := c.Args.CheckLen(1); err != nil {
					return nil, err
				}
				return gad.Str("hello " + c.Args.Get(0).ToString()), nil
			},
		},
	}}
	return nil
}
`, "NAME", name)
}

// samplesTemplate renders the samples.gad seed for a module named name.
func samplesTemplate(name string) string {
	return strings.ReplaceAll(`// Samples of module NAME: one snippet region per exported member. These are
// standard doctest snippets, run and checked by `+"`gad doc`"+`, and merged into the
// module API as `+"`## Example`"+` sections by gaddoc (the `+"`gad:samples`"+` directive).

//snippet hello
NAME := import("NAME")
NAME.hello("Gad")
//= "hello Gad"
//endsnippet
`, "NAME", name)
}
