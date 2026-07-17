package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
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
	"tutorial", "stdlib-strings", "stdlib-fmt", "stdlib-json", "stdlib-time",
}

// giomOrder is the curated nav ordering for the Giom template docs, sourced from
// the ./giom submodule's docs directory. Pages are emitted with a `giom-` prefix
// so they never collide with the Gad guide/reference pages.
var giomOrder = []string{
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

	// Giom template docs live in the ./giom submodule. When it is checked out,
	// publish them as their own "Giom" nav section (prefixed pages + copied
	// image assets); when it is absent the section is simply omitted.
	giomDir := filepath.Join(repoRoot, "giom", "docs")
	giomPages, err := collectGiomPages(giomDir, giomOrder)
	if err != nil {
		return err
	}
	if len(giomPages) > 0 {
		if err := copyGiomAssets(giomDir, outDir); err != nil {
			return err
		}
	}

	play := &page{Slug: "playground", Title: "Playground", OutFile: "playground.html", Section: "Playground"}

	groups := []navGroup{
		{Name: "Guide", Pages: guide},
		{Name: "Reference", Pages: ref},
	}
	if len(giomPages) > 0 {
		groups = append(groups, navGroup{Name: "Giom", Pages: giomPages})
	}
	groups = append(groups, navGroup{Name: "Playground", Pages: []*page{play}})

	all := append(append([]*page{}, guide...), ref...)
	all = append(all, giomPages...)

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

// giomLinkRe matches Markdown links to a sibling `.md` doc (no scheme, no path
// separator), so intra-giom links can be rewritten to the prefixed output names.
var giomLinkRe = regexp.MustCompile(`\]\(([A-Za-z0-9_-]+)\.md(#[A-Za-z0-9_-]+)?\)`)

// collectGiomPages renders the Giom docs found in dir (in order) into pages named
// `giom-<name>.html`. Intra-doc `.md` links are rewritten to the same prefixed
// names so cross-references resolve within the generated site; missing pages are
// tolerated (the submodule may not be checked out).
func collectGiomPages(dir string, order []string) ([]*page, error) {
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
		// Rewrite `](other.md)` / `](other.md#anchor)` to `](giom-other.html…)`.
		rewritten := giomLinkRe.ReplaceAllString(string(src), "](giom-$1.html$2)")
		body, headings := renderMarkdown(rewritten)
		title := firstHeading(headings)
		if title == "" {
			title = name
		}
		pages = append(pages, &page{
			Slug:     "giom-" + name,
			Title:    title,
			OutFile:  "giom-" + name + ".html",
			Section:  "Giom",
			BodyHTML: template.HTML(body),
			Headings: headings,
			plain:    plainText(rewritten),
		})
	}
	return pages, nil
}

// copyGiomAssets copies non-Markdown files (images such as the benchmark SVGs)
// from the Giom docs directory into the site root, where the prefixed Giom pages
// reference them by their original relative names.
func copyGiomAssets(dir, outDir string) error {
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

// layoutData is passed to the page template.
type layoutData struct {
	Title   string
	Groups  []navGroup
	Active  string
	Content template.HTML
	TOC     []Heading
}

func writePage(outDir string, tmpl *template.Template, groups []navGroup, p *page, body template.HTML) error {
	data := layoutData{
		Title:   p.Title,
		Groups:  groups,
		Active:  p.OutFile,
		Content: body,
		TOC:     tocOf(p.Headings),
	}
	f, err := os.Create(filepath.Join(outDir, p.OutFile))
	if err != nil {
		return err
	}
	defer f.Close()
	return tmpl.Execute(f, data)
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
