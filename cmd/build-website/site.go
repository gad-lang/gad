package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// page is one rendered documentation page.
type page struct {
	Slug     string
	Title    string
	OutFile  string
	Section  string
	BodyHTML template.HTML
	Headings []Heading
	// plain is the searchable plain text of the page.
	plain string
}

// navGroup is a sidebar section.
type navGroup struct {
	Name  string
	Pages []*page
}

// guideOrder / refOrder give the curated nav ordering (filenames without .md).
var guideOrder = []string{
	"README", "getting-started", "values-and-types", "variables-and-scopes",
	"operators", "control-flow", "functions", "collections",
	"strings-bytes-regex", "error-handling", "modules", "builtins",
	"formatting", "embedding",
}

var refOrder = []string{
	"tutorial", "metaprogramming", "workspace-config", "stdlib-strings", "stdlib-fmt", "stdlib-json", "stdlib-time",
}

// gadxOrder is the curated nav ordering for the Gadx template docs, sourced from
// the ./gadx submodule's docs directory. Pages are emitted with a `gadx-` prefix
// so they never collide with the Gad guide/reference pages.
var gadxOrder = []string{
	"getting-started", "syntax", "components-and-slots", "examples",
	"embedding", "api", "conventions", "source-positions",
	"project-structure", "benchmarks", "cms-example",
}

// buildSite renders the whole website into outDir.
func buildSite(repoRoot, outDir string, buildWASM bool) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	guide, err := collectPages(filepath.Join(repoRoot, "doc"), guideOrder, "Guide", true)
	if err != nil {
		return err
	}
	ref, err := collectPages(filepath.Join(repoRoot, "doc"), refOrder, "Reference", false)
	if err != nil {
		return err
	}

	// Gadx template docs live in the ./gadx submodule. When it is checked out,
	// publish them as their own "Gadx" nav section (prefixed pages + copied
	// image assets); when it is absent the section is simply omitted.
	gadxDir := filepath.Join(repoRoot, "gadx", "docs")
	gadxPages, err := collectGadxPages(gadxDir, gadxOrder)
	if err != nil {
		return err
	}
	if len(gadxPages) > 0 {
		if err := copyGadxAssets(gadxDir, outDir); err != nil {
			return err
		}
	}

	play := &page{Slug: "playground", Title: "Playground", OutFile: "playground.html", Section: "Playground"}

	// JS module docs (web/*/README.md) become the "JS modules" nav section, one
	// page per package at /js-modules/<name>.
	jsPages, err := collectJSModulePages(repoRoot)
	if err != nil {
		return err
	}

	// The embedded server-less IDE (the Vuetify demo) is built into /ide/ when a
	// WASM build is requested and bun is available; it is tolerated if absent.
	ideBuilt := false
	if buildWASM {
		if err := buildEmbeddedIDE(repoRoot, outDir); err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping embedded IDE (/ide): %v\n", err)
		} else {
			ideBuilt = true
		}
	}

	groups := []navGroup{
		{Name: "Guide", Pages: guide},
		{Name: "Reference", Pages: ref},
	}
	if len(gadxPages) > 0 {
		groups = append(groups, navGroup{Name: "Gadx", Pages: gadxPages})
	}
	if len(jsPages) > 0 {
		groups = append(groups, navGroup{Name: "JS modules", Pages: jsPages})
	}
	if ideBuilt {
		groups = append(groups, navGroup{Name: "IDE", Pages: []*page{
			{Slug: "ide", Title: "IDE (server-less)", OutFile: "ide/index.html", Section: "IDE"},
		}})
	}
	groups = append(groups, navGroup{Name: "Playground", Pages: []*page{play}})

	all := append(append([]*page{}, guide...), ref...)
	all = append(all, gadxPages...)
	all = append(all, jsPages...)

	tmpl := template.Must(template.New("layout").Parse(layoutTemplate))

	for _, p := range all {
		if err := writePage(outDir, tmpl, groups, p, p.BodyHTML); err != nil {
			return err
		}
	}
	// Playground page (custom body).
	if err := writePage(outDir, tmpl, groups, play, template.HTML(playgroundBody)); err != nil {
		return err
	}

	if err := writeSearchIndex(outDir, all); err != nil {
		return err
	}
	if err := writeAssets(outDir); err != nil {
		return err
	}
	if buildWASM {
		if err := buildWASMAssets(repoRoot, outDir); err != nil {
			return fmt.Errorf("building wasm: %w", err)
		}
	}
	return nil
}

func collectPages(dir string, order []string, section string, isGuide bool) ([]*page, error) {
	var pages []*page
	for _, name := range order {
		path := filepath.Join(dir, name+".md")
		src, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // tolerate missing optional pages
			}
			return nil, err
		}
		body, headings := renderMarkdown(string(src))
		title := firstHeading(headings)
		if title == "" {
			title = name
		}
		out := name + ".html"
		slug := name
		if isGuide && name == "README" {
			out = "index.html"
			slug = "index"
			title = "Gad Language"
		} else if !isGuide {
			out = "ref-" + name + ".html"
			slug = "ref-" + name
		}
		pages = append(pages, &page{
			Slug:     slug,
			Title:    title,
			OutFile:  out,
			Section:  section,
			BodyHTML: template.HTML(body),
			Headings: headings,
			plain:    plainText(string(src)),
		})
	}
	return pages, nil
}

// gadxLinkRe matches Markdown links to a sibling `.md` doc (no scheme, no path
// separator), so intra-gadx links can be rewritten to the prefixed output names.
var gadxLinkRe = regexp.MustCompile(`\]\(([A-Za-z0-9_-]+)\.md(#[A-Za-z0-9_-]+)?\)`)

// collectGadxPages renders the Gadx docs found in dir (in order) into pages named
// `gadx-<name>.html`. Intra-doc `.md` links are rewritten to the same prefixed
// names so cross-references resolve within the generated site; missing pages are
// tolerated (the submodule may not be checked out).
func collectGadxPages(dir string, order []string) ([]*page, error) {
	var pages []*page
	for _, name := range order {
		path := filepath.Join(dir, name+".md")
		src, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		// Rewrite `](other.md)` / `](other.md#anchor)` to `](gadx-other.html…)`.
		rewritten := gadxLinkRe.ReplaceAllString(string(src), "](gadx-$1.html$2)")
		body, headings := renderMarkdown(rewritten)
		title := firstHeading(headings)
		if title == "" {
			title = name
		}
		pages = append(pages, &page{
			Slug:     "gadx-" + name,
			Title:    title,
			OutFile:  "gadx-" + name + ".html",
			Section:  "Gadx",
			BodyHTML: template.HTML(body),
			Headings: headings,
			plain:    plainText(rewritten),
		})
	}
	return pages, nil
}

// copyGadxAssets copies non-Markdown files (images such as the benchmark SVGs)
// from the Gadx docs directory into the site root, where the prefixed Gadx pages
// reference them by their original relative names.
func copyGadxAssets(dir, outDir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outDir, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// jsModules are the publishable web packages documented under "JS modules".
var jsModules = []struct{ Name, Title string }{
	{"codemirror-gad", "@gad-lang/codemirror-gad"},
	{"prism-gad", "@gad-lang/prism-gad"},
	{"ide-react", "@gad-lang/ide-react"},
	{"ide-vuetify", "@gad-lang/ide-vuetify"},
}

// mdLinkRe matches Markdown link targets `](target)`.
var mdLinkRe = regexp.MustCompile(`\]\(([^)]+)\)`)

// rewriteModuleLinks rewrites a module README's relative links to absolute GitHub
// URLs (rooted at web/<name>/), so its `./docs/*.md`, `../<pkg>` and example
// links resolve on the docs site. External, anchor and absolute links are left
// as-is.
func rewriteModuleLinks(md, name string) string {
	return mdLinkRe.ReplaceAllStringFunc(md, func(m string) string {
		target := m[2 : len(m)-1]
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") ||
			strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "/") {
			return m
		}
		anchor := ""
		if i := strings.IndexByte(target, '#'); i >= 0 {
			anchor, target = target[i:], target[:i]
		}
		resolved := path.Join("web", name, target)
		return "](https://github.com/gad-lang/gad/blob/main/" + resolved + anchor + ")"
	})
}

// collectJSModulePages renders each web module's README into a page at
// js-modules/<name>.html (the "JS modules" nav section). A missing package is
// tolerated.
func collectJSModulePages(repoRoot string) ([]*page, error) {
	var pages []*page
	for _, m := range jsModules {
		fp := filepath.Join(repoRoot, "web", m.Name, "README.md")
		src, err := os.ReadFile(fp)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		md := rewriteModuleLinks(string(src), m.Name)
		body, headings := renderMarkdown(md)
		pages = append(pages, &page{
			Slug:     "js-modules-" + m.Name,
			Title:    m.Title,
			OutFile:  "js-modules/" + m.Name + ".html",
			Section:  "JS modules",
			BodyHTML: template.HTML(body),
			Headings: headings,
			plain:    plainText(md),
		})
	}
	return pages, nil
}

// buildEmbeddedIDE builds the server-less Vuetify IDE demo with a relative base
// and copies its output into <outDir>/ide, so the docs site serves a full,
// in-browser IDE (debug, inspect, run profiles) at /ide/. Requires bun and the
// workspace to be installed; the caller tolerates failure.
func buildEmbeddedIDE(repoRoot, outDir string) error {
	demo := filepath.Join(repoRoot, "web", "ide-vuetify", "demo")
	if _, err := os.Stat(demo); err != nil {
		return fmt.Errorf("demo not found: %w", err)
	}
	cmd := exec.Command("bash", "-c", "bun run wasm && bunx vite build --base=./")
	cmd.Dir = demo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return copyTree(filepath.Join(demo, "dist"), filepath.Join(outDir, "ide"))
}

// copyTree recursively copies the directory src into dst.
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

// layoutData is passed to the page template.
type layoutData struct {
	Title   string
	Groups  []navGroup
	Active  string
	Content template.HTML
	TOC     []Heading
	// Base is the relative prefix from this page's directory back to the site
	// root ("" for root pages, "../" for pages one level deep like js-modules/*),
	// so assets and nav links resolve at any base path and any page depth.
	Base string
}

func writePage(outDir string, tmpl *template.Template, groups []navGroup, p *page, body template.HTML) error {
	data := layoutData{
		Title:   p.Title,
		Groups:  groups,
		Active:  p.OutFile,
		Content: body,
		TOC:     tocOf(p.Headings),
		Base:    baseFor(p.OutFile),
	}
	outPath := filepath.Join(outDir, filepath.FromSlash(p.OutFile))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
}

// baseFor returns the relative path from a page's directory back to the site
// root: "" for a root-level OutFile, "../" per directory of depth otherwise.
func baseFor(outFile string) string {
	depth := strings.Count(outFile, "/")
	return strings.Repeat("../", depth)
}

// tocOf returns the H2 headings of a page for the right-hand table of contents.
func tocOf(hs []Heading) []Heading {
	var out []Heading
	for _, h := range hs {
		if h.Level == 2 {
			out = append(out, h)
		}
	}
	return out
}

func firstHeading(hs []Heading) string {
	for _, h := range hs {
		if h.Level == 1 {
			return h.Text
		}
	}
	if len(hs) > 0 {
		return hs[0].Text
	}
	return ""
}

// searchDoc is one entry in the client-side search index.
type searchDoc struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"`
}

func writeSearchIndex(outDir string, pages []*page) error {
	docs := make([]searchDoc, 0, len(pages))
	for _, p := range pages {
		text := p.plain
		if len(text) > 4000 {
			text = text[:4000]
		}
		docs = append(docs, searchDoc{Title: p.Title, URL: p.OutFile, Text: text})
	}
	data, err := json.Marshal(docs)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "search.json"), data, 0o644)
}

// plainText strips markdown to plain text for the search index.
func plainText(src string) string {
	var b strings.Builder
	inCode := false
	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(strings.TrimRight(line, " "), "```") {
			inCode = !inCode
			continue
		}
		l := line
		if !inCode {
			l = strings.TrimLeft(l, "#>-*| ")
		}
		l = stripInline(l)
		if strings.TrimSpace(l) != "" {
			b.WriteString(l)
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func writeAssets(outDir string) error {
	files := map[string]string{
		"styles.css": siteCSS,
		"search.js":  searchJS,
		"theme.js":   themeJS,
		"play.js":    playJS,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// buildWASMAssets compiles the Gad WASM module and copies wasm_exec.js into the
// output directory so the Playground page works offline / on GitHub Pages.
func buildWASMAssets(repoRoot, outDir string) error {
	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "gad.wasm"), "./web/wasm")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}

	goroot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return err
	}
	root := strings.TrimSpace(string(goroot))
	for _, cand := range []string{
		filepath.Join(root, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(root, "misc", "wasm", "wasm_exec.js"),
	} {
		if data, err := os.ReadFile(cand); err == nil {
			return os.WriteFile(filepath.Join(outDir, "wasm_exec.js"), data, 0o644)
		}
	}
	return fmt.Errorf("wasm_exec.js not found under %s", root)
}
