package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// content.json is the data model consumed by the Vue + Vuetify SPA in
// web/website: the site info, the navigation groups, every documentation page's
// rendered HTML + table of contents, and a search index.

type jsonSite struct {
	RepoURL      string `json:"repoURL"`
	Commit       string `json:"commit"`
	PlayHref     string `json:"playHref"`
	DownloadSlug string `json:"downloadSlug"`
	ReleaseName  string `json:"releaseName"`
	HasRelease   bool   `json:"hasRelease"`
	Tagline      string `json:"tagline"`
}

type jsonNavPage struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	// Href is set for an external nav entry (e.g. the Playground app), in which
	// case there is no doc page to route to.
	Href string `json:"href,omitempty"`
	// Children is set for a submenu entry: a directory of pages (e.g. class/) with
	// no page of its own. Nested recursively for deeper subdirectories.
	Children []jsonNavPage `json:"children,omitempty"`
}

type jsonNavGroup struct {
	Name  string        `json:"name"`
	Pages []jsonNavPage `json:"pages"`
}

type jsonToc struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Level int    `json:"level"`
}

type jsonDoc struct {
	Slug  string    `json:"slug"`
	Title string    `json:"title"`
	HTML  string    `json:"html"`
	Toc   []jsonToc `json:"toc"`
	// Source is the backing sample source (language chapters), with SourceLang its
	// PrismJS language; omitted for pages without a source.
	Source     string `json:"source,omitempty"`
	SourceLang string `json:"sourceLang,omitempty"`
	SourcePath string `json:"sourcePath,omitempty"`
}

type jsonSearch struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type jsonContent struct {
	Site   jsonSite           `json:"site"`
	Home   string             `json:"home"`
	Groups []jsonNavGroup     `json:"groups"`
	Pages  map[string]jsonDoc `json:"pages"`
	Search []jsonSearch       `json:"search"`
}

const siteTagline = "A fast, dynamic scripting language embedded in Go — great as a frontend layer (Gadx) and for report generation."

// slugOf turns a page OutFile into its SPA route slug (drops the .html suffix,
// keeping any subdirectory so js-modules/time.html -> js-modules/time).
func slugOf(outFile string) string {
	return strings.TrimSuffix(outFile, ".html")
}

// tocEntries collects the level-2/3 headings for a page's on-this-page rail. It
// always returns a non-nil slice so the JSON `toc` is an array (never null).
func tocEntries(hs []Heading) []jsonToc {
	out := []jsonToc{}
	for _, h := range hs {
		if h.Level == 2 || h.Level == 3 {
			out = append(out, jsonToc{ID: h.ID, Text: h.Text, Level: h.Level})
		}
	}
	return out
}

// navNode is an intermediate nav tree node: a leaf (a page) or a directory with
// ordered children keyed by their path segment.
type navNode struct {
	item     jsonNavPage
	isDir    bool
	children []*navNode
	byName   map[string]*navNode
}

// buildNavTree turns a group's ordered pages into a nav tree: a page whose
// NavPath carries a subdirectory (`class/classes`) nests under a submenu named
// for the directory, recursively for deeper paths. Pages keep their group order;
// a directory takes the position of its first page. Pages with no NavPath (or no
// `/`) stay flat top-level entries, so existing sections are unchanged.
func buildNavTree(pages []*page) []jsonNavPage {
	root := &navNode{isDir: true, byName: map[string]*navNode{}}

	for _, p := range pages {
		leaf := jsonNavPage{Slug: slugOf(p.OutFile), Title: p.Title}
		if p.BodyHTML == "" {
			leaf = jsonNavPage{Title: p.Title, Href: p.OutFile}
		}

		segments := []string{}
		if p.NavPath != "" {
			segments = strings.Split(p.NavPath, "/")
		}
		// Walk directory segments (all but the last, which is the page itself),
		// creating submenu nodes as needed.
		node := root
		for i := 0; i+1 < len(segments); i++ {
			dir := segments[i]
			child := node.byName[dir]
			if child == nil {
				child = &navNode{isDir: true, byName: map[string]*navNode{}, item: jsonNavPage{Title: navDirTitle(dir)}}
				node.byName[dir] = child
				node.children = append(node.children, child)
			}
			node = child
		}
		node.children = append(node.children, &navNode{item: leaf})
	}
	return flattenNav(root.children)
}

// flattenNav converts nav nodes to jsonNavPages, attaching each directory node's
// children recursively.
func flattenNav(nodes []*navNode) []jsonNavPage {
	out := make([]jsonNavPage, 0, len(nodes))
	for _, n := range nodes {
		it := n.item
		if n.isDir {
			it.Children = flattenNav(n.children)
		}
		out = append(out, it)
	}
	return out
}

// navDirTitle makes a readable submenu label from a directory segment
// (`class` -> `Class`).
func navDirTitle(dir string) string {
	if dir == "" {
		return dir
	}
	return strings.ToUpper(dir[:1]) + dir[1:]
}

// writeContent emits content.json from the navigation groups. A page with body
// HTML becomes a routed doc page; a page without (the external Playground app) is
// a plain link.
func writeContent(outDir string, groups []navGroup, cfg siteConfig) error {
	c := jsonContent{
		Site: jsonSite{
			RepoURL:      cfg.RepoURL,
			Commit:       cfg.commit,
			PlayHref:     cfg.playHref,
			DownloadSlug: "download",
			ReleaseName:  cfg.releaseName(),
			HasRelease:   cfg.hasRelease(),
			Tagline:      siteTagline,
		},
		Pages: map[string]jsonDoc{},
	}
	for _, g := range groups {
		// Register every doc page (flat), then build the group's nav items as a tree
		// so pages sharing a subdirectory prefix (NavPath `class/…`) nest under an
		// expandable submenu.
		for _, p := range g.Pages {
			slug := slugOf(p.OutFile)
			if p.BodyHTML == "" {
				continue // external link: no doc page to register
			}
			c.Pages[slug] = jsonDoc{
				Slug: slug, Title: p.Title, HTML: string(p.BodyHTML), Toc: tocEntries(p.Headings),
				Source: p.Source, SourceLang: p.SourceLang, SourcePath: p.SourcePath,
			}
			c.Search = append(c.Search, jsonSearch{Slug: slug, Title: p.Title, Text: p.plain})
		}
		c.Groups = append(c.Groups, jsonNavGroup{Name: g.Name, Pages: buildNavTree(g.Pages)})
	}
	data, err := json.Marshal(&c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "content.json"), data, 0o644)
}

// buildWebsiteSPA builds the Vue + Vuetify docs shell (web/website) with vite and
// copies its dist into outDir. Requires bun. The caller writes content.json and
// the logo/prism/wasm assets afterwards so they sit next to the built app.
func buildWebsiteSPA(repoRoot, outDir string) error {
	app := filepath.Join(repoRoot, "web", "website")
	if _, err := os.Stat(app); err != nil {
		return fmt.Errorf("website app not found: %w", err)
	}
	cmd := exec.Command("bash", "-c", "bunx vite build --base=./")
	cmd.Dir = app
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return copyTree(filepath.Join(app, "dist"), outDir)
}
