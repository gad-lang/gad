// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gad "github.com/gad-lang/gad"
	cc "github.com/moisespsena-go/command-context"
)

// indexNode is one directory in the generated output tree: the Markdown files
// documented directly in it and its immediate subdirectories.
type indexNode struct {
	files []string            // absolute .md output paths in this directory
	dirs  map[string]struct{} // absolute immediate-subdirectory paths
}

// generateIndexes writes a per-directory index — README.md (and index.html when
// HTML output is enabled) — for every directory that received documentation,
// listing the files in it and linking to the subdirectory indexes.
func (o *docOptions) generateIndexes(ctx *cc.CommandContext, tset *docTemplateSet) error {
	roots := make([]string, 0, len(o.indexRoots))
	for r := range o.indexRoots {
		roots = append(roots, r)
	}
	tree := buildIndexTree(o.written, roots)

	dirs := make([]string, 0, len(tree))
	for d := range tree {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		node := tree[dir]
		if err := o.writeIndex(ctx, tset, dir, node, "md", indexFileName("md"), tset.indexMdSrc, tset.indexMdPath); err != nil {
			return err
		}
		if tset.indexHTMLSrc != nil {
			if err := o.writeIndex(ctx, tset, dir, node, "html", indexFileName("html"), tset.indexHTMLSrc, tset.indexHTMLPath); err != nil {
				return err
			}
		}
		// JSON/YAML indexes follow the same per-directory rule, encoding the
		// index structure instead of rendering a template.
		if o.json {
			if err := o.writeEncodedIndex(ctx, dir, node, "json"); err != nil {
				return err
			}
		}
		if o.yaml {
			if err := o.writeEncodedIndex(ctx, dir, node, "yaml"); err != nil {
				return err
			}
		}
	}
	return nil
}

// indexFileName maps an output extension to its directory index file name.
func indexFileName(ext string) string {
	switch ext {
	case "html":
		return "index.html"
	case "json":
		return "README.json"
	case "yaml":
		return "README.yaml"
	default:
		return "README.md"
	}
}

// writeIndex renders and writes one directory index (ext "md" or "html").
func (o *docOptions) writeIndex(ctx *cc.CommandContext, tset *docTemplateSet, dir string, node *indexNode, ext, indexFile string, tmplSrc []byte, tmplPath string) error {
	dict := buildIndexDict(dir, node, ext, indexFileName(ext))
	body, err := renderIndexTemplate(tmplSrc, tmplPath, dict)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, indexFile)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err = os.WriteFile(out, []byte(body), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Out, out)
	return nil
}

// buildIndexTree groups the written Markdown paths into a directory tree bounded
// by roots: every path registers under its directory, and each directory up to
// (and including) its root is linked as a child of its parent.
func buildIndexTree(paths, roots []string) map[string]*indexNode {
	tree := map[string]*indexNode{}
	get := func(d string) *indexNode {
		n := tree[d]
		if n == nil {
			n = &indexNode{dirs: map[string]struct{}{}}
			tree[d] = n
		}
		return n
	}
	for _, p := range paths {
		root := containingRoot(p, roots)
		if root == "" {
			continue
		}
		d := filepath.Dir(p)
		get(d).files = append(get(d).files, p)
		for d != root {
			parent := filepath.Dir(d)
			if parent == d {
				break // reached the filesystem root without matching
			}
			get(parent).dirs[d] = struct{}{}
			d = parent
		}
		get(root)
	}
	return tree
}

// containingRoot returns the root that contains p (p is at or below it), or "".
func containingRoot(p string, roots []string) string {
	best := ""
	for _, r := range roots {
		rel, err := filepath.Rel(r, p)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if len(r) > len(best) {
			best = r
		}
	}
	return best
}

// buildIndexDict builds the `param (index dict)` value an index template
// consumes: the directory name and the file/subdirectory links (ext-suffixed;
// subdir links point at the subdirectory's index file).
func buildIndexDict(dir string, node *indexNode, ext, indexFile string) gad.Dict {
	items := make(gad.Array, 0, len(node.files))
	names := make([]string, 0, len(node.files))
	for _, f := range node.files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
	}
	sort.Strings(names)
	for _, n := range names {
		items = append(items, gad.Dict{"name": gad.Str(n), "file": gad.Str(n + "." + ext)})
	}

	subs := make([]string, 0, len(node.dirs))
	for d := range node.dirs {
		subs = append(subs, filepath.Base(d))
	}
	sort.Strings(subs)
	dirs := make(gad.Array, 0, len(subs))
	for _, d := range subs {
		dirs = append(dirs, gad.Dict{"name": gad.Str(d), "file": gad.Str(d + "/" + indexFile)})
	}

	name := filepath.Base(dir)
	return gad.Dict{"name": gad.Str(name), "path": gad.Str(dir), "items": items, "dirs": dirs}
}

// buildIndexData is the Go-struct counterpart of buildIndexDict, for the encoded
// (JSON/YAML) directory indexes: the same directory name, file entries and
// subdirectory links (ext-suffixed; subdir links point at the subdirectory's
// index file).
func buildIndexData(dir string, node *indexNode, ext, indexFile string) indexEncoded {
	names := make([]string, 0, len(node.files))
	for _, f := range node.files {
		names = append(names, strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)))
	}
	sort.Strings(names)
	items := make([]indexEntry, 0, len(names))
	for _, n := range names {
		items = append(items, indexEntry{Name: n, File: n + "." + ext})
	}

	subs := make([]string, 0, len(node.dirs))
	for d := range node.dirs {
		subs = append(subs, filepath.Base(d))
	}
	sort.Strings(subs)
	dirs := make([]indexEntry, 0, len(subs))
	for _, d := range subs {
		dirs = append(dirs, indexEntry{Name: d, File: d + "/" + indexFile})
	}

	return indexEncoded{Name: filepath.Base(dir), Path: dir, Items: items, Dirs: dirs}
}

// renderIndexTemplate compiles and runs an index template with `param (index
// dict)`, returning the rendered body. It shares the dialect-aware machinery of
// renderDocTemplate.
func renderIndexTemplate(tmplSrc []byte, tmplPath string, index gad.Dict) (string, error) {
	return renderDocTemplate(tmplSrc, tmplPath, index)
}
