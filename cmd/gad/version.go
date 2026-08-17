// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	"runtime/debug"

	cc "github.com/moisespsena-go/command-context"
)

func init() { registerCommand("version", versionCommand) }

// version is the release version, injected at build time with
// `-ldflags "-X main.version=vX.Y.Z"` (goreleaser sets it from the git tag).
// When empty it is resolved from the Go build info (module version or VCS
// revision), falling back to "dev" for a plain `go build`/`go run`.
var version = ""

// resolveVersion returns the release version, preferring the ldflags-injected
// value and otherwise reading the module version / VCS revision embedded by the
// Go toolchain.
func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				rev := s.Value
				if len(rev) > 12 {
					rev = rev[:12]
				}
				return "dev+" + rev
			}
		}
	}
	return "dev"
}

// versionCommand is the `gad version` subcommand: it prints the Gad version and
// the build's Go/OS/arch. The IntelliJ/GoLand plugin runs it to display the
// resolved binary's version in the Gad settings.
func versionCommand() *cc.Command {
	return &cc.Command{
		Name:        "version",
		Usage:       "",
		Description: "Print the Gad version and build information.",
		Run: func(ctx *cc.CommandContext) error {
			fmt.Fprintf(ctx.Out, "gad %s\n", resolveVersion())
			fmt.Fprintf(ctx.Out, "build: %s %s/%s\n",
				runtime.Version(), runtime.GOOS, runtime.GOARCH)
			return nil
		},
	}
}
