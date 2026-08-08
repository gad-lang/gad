// Command build-website renders the Gad documentation (./doc) into a
// static, GitHub-Pages-ready website with client-side search, a light/dark
// theme and a WebAssembly playground.
//
//	go run ./cmd/build-website build --out dist/website
//	go run ./cmd/build-website serve --out dist/website   # preview on :8090
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	cc "github.com/moisespsena-go/command-context"
)

func main() {
	root := &cc.Command{
		Name:        "build-website",
		Description: "Build the static Gad documentation website.",
		Run: func(ctx *cc.CommandContext) error {
			return ctx.Help()
		},
	}
	root.Sub(buildCommand())
	root.Sub(serveCommand())

	ctx, err := root.Parse(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	if err = ctx.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
}

func buildCommand() *cc.Command {
	var (
		out          *string
		repo         *string
		noWASM       *bool
		repoURL      *string
		relTag       *string
		relName      *string
		relNotes     *string
		relNotesFile *string
		relDate      *string
	)
	return &cc.Command{
		Name:        "build",
		Usage:       "[flags]",
		Description: "Render the website into the output directory.",
		New: func(ctx *cc.CommandContext) error {
			out = ctx.Flags().String("out", "dist/website", "output directory")
			repo = ctx.Flags().String("repo", ".", "repository root (contains doc/)")
			noWASM = ctx.Flags().Bool("no-wasm", false, "skip building the WebAssembly playground module")
			repoURL = ctx.Flags().String("repo-url", "https://github.com/gad-lang/gad", "repository URL (header + download links)")
			relTag = ctx.Flags().String("release-tag", "", "release tag (e.g. v1.2.3) for the Download page and release banner")
			relName = ctx.Flags().String("release-name", "", "release display name (defaults to the tag)")
			relNotes = ctx.Flags().String("release-notes", "", "release notes as Markdown (inline)")
			relNotesFile = ctx.Flags().String("release-notes-file", "", "path to a file with the release notes (Markdown)")
			relDate = ctx.Flags().String("release-date", "", "release date shown next to the name")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			notes := *relNotes
			if notes == "" && *relNotesFile != "" {
				data, err := os.ReadFile(*relNotesFile)
				if err != nil {
					return fmt.Errorf("read release notes file: %w", err)
				}
				notes = string(data)
			}
			cfg := siteConfig{
				RepoURL:      strings.TrimRight(*repoURL, "/"),
				ReleaseTag:   *relTag,
				ReleaseName:  *relName,
				ReleaseNotes: notes,
				ReleaseDate:  *relDate,
				BuildWASM:    !*noWASM,
			}
			if err := buildSite(*repo, *out, cfg); err != nil {
				return err
			}
			fmt.Fprintf(ctx.Out, "website written to %s\n", *out)
			return nil
		},
	}
}

func serveCommand() *cc.Command {
	var (
		out  *string
		addr *string
	)
	return &cc.Command{
		Name:        "serve",
		Usage:       "[flags]",
		Description: "Serve a previously built website for local preview.",
		New: func(ctx *cc.CommandContext) error {
			out = ctx.Flags().String("out", "dist/website", "directory to serve")
			addr = ctx.Flags().String("addr", ":8090", "listen address")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			log.Printf("serving %s on %s", *out, *addr)
			return http.ListenAndServe(*addr, http.FileServer(http.Dir(*out)))
		},
	}
}
