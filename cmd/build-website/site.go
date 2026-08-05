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

// siteConfig carries build-time site metadata: repository/tasks links and the
// release info (tag, name, notes) surfaced in the header banner and the Download
// page.
type siteConfig struct {
	RepoURL      string // e.g. https://github.com/gad-lang/gad
	TasksURL     string // link to TASK.md (defaults to <RepoURL>/blob/main/TASK.md)
	ReleaseTag   string // e.g. v1.2.3 (empty for a plain commit build)
	ReleaseName  string // display name (defaults to the tag)
	ReleaseNotes string // release notes as Markdown
	ReleaseDate  string // display date next to the name
	BuildWASM    bool

	// playHref is the site-root-relative path of the Playground/demo, set during
	// the build (not a flag) for the header link.
	playHref string
	// prism is set when the PrismJS bundle was built (loads prism.js + enables
	// syntax highlighting).
	prism bool
}

// tasksURL returns the effective Tasks link (the repository issues by default).
func (c siteConfig) tasksURL() string {
	if c.TasksURL != "" {
		return c.TasksURL
	}
	return c.RepoURL + "/issues"
}

// releaseName returns the display name for the release banner (name, else tag).
func (c siteConfig) releaseName() string {
	if c.ReleaseName != "" {
		return c.ReleaseName
	}
	return c.ReleaseTag
}

// hasRelease reports whether a tagged release is known (banner + asset links).
func (c siteConfig) hasRelease() bool { return c.ReleaseTag != "" }

// releaseURL is the GitHub release page (specific tag, or /latest).
func (c siteConfig) releaseURL() string {
	if c.hasRelease() {
		return c.RepoURL + "/releases/tag/" + c.ReleaseTag
	}
	return c.RepoURL + "/releases/latest"
}

// assetURL returns the download URL of a release asset for the known tag; when
// no tag is known it points at the latest release's asset by name.
func (c siteConfig) assetURL(name string) string {
	if c.hasRelease() {
		return c.RepoURL + "/releases/download/" + c.ReleaseTag + "/" + name
	}
	return c.RepoURL + "/releases/latest/download/" + name
}

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
	"templates", "formatting", "embedding",
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
func buildSite(repoRoot, outDir string, cfg siteConfig) error {
	buildWASM := cfg.BuildWASM
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

	download := &page{Slug: "download", Title: "Download", OutFile: "download.html", Section: "Download"}
	wasmEmbed := &page{Slug: "wasm-embed", Title: "Embed the WASM", OutFile: "wasm-embed.html", Section: "Integrate"}

	// JS module docs (web/*/README.md) become the "JS modules" nav section, one
	// page per package at /js-modules/<name>.
	jsPages, err := collectJSModulePages(repoRoot)
	if err != nil {
		return err
	}

	// The "Playground" menu hosts the full ide-vuetify demo app (Playground /
	// Notebook / IDE tabs) built into /playground when a WASM build is requested
	// and bun is available. When that build is unavailable (e.g. bun missing), we
	// fall back to the simple in-page WASM playground so the menu still works.
	demoBuilt := false
	if buildWASM {
		if err := buildEmbeddedIDE(repoRoot, outDir, "playground"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: full ide-vuetify demo unavailable, using the simple playground: %v\n", err)
		} else {
			demoBuilt = true
		}
	}
	var play *page
	if demoBuilt {
		// The demo is a standalone Vite app with its own index.html; link to it.
		play = &page{Slug: "playground", Title: "Playground", OutFile: "playground/index.html", Section: "Playground"}
	} else {
		play = &page{Slug: "playground", Title: "Playground", OutFile: "playground.html", Section: "Playground"}
	}
	cfg.playHref = play.OutFile

	// PrismJS bundle for static syntax highlighting (best-effort; needs bun).
	if err := buildPrismBundle(repoRoot, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: syntax highlighting disabled (prism bundle): %v\n", err)
	} else {
		cfg.prism = true
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
	groups = append(groups, navGroup{Name: "Integrate", Pages: []*page{wasmEmbed}})
	groups = append(groups, navGroup{Name: "Playground", Pages: []*page{play}})
	groups = append(groups, navGroup{Name: "Download", Pages: []*page{download}})

	// The home page (index) opens with a hero + a three-path quickstart (CLI,
	// WASM, Go module) prepended to the rendered README.
	for _, p := range guide {
		if p.Slug == "index" {
			p.BodyHTML = homeIntroHTML(cfg, play.OutFile) + p.BodyHTML
			break
		}
	}

	all := append(append([]*page{}, guide...), ref...)
	all = append(all, gadxPages...)
	all = append(all, jsPages...)

	tmpl := template.Must(template.New("layout").Parse(layoutTemplate))

	for _, p := range all {
		if err := writePage(outDir, tmpl, groups, p, p.BodyHTML, cfg); err != nil {
			return err
		}
	}
	// Simple in-page playground — only when the full demo is not embedded.
	if !demoBuilt {
		if err := writePage(outDir, tmpl, groups, play, template.HTML(playgroundBody), cfg); err != nil {
			return err
		}
	}
	// Download page (custom body: release banner, notes and asset table).
	if err := writePage(outDir, tmpl, groups, download, downloadBody(cfg), cfg); err != nil {
		return err
	}
	// WASM-embedding guide (custom body).
	if err := writePage(outDir, tmpl, groups, wasmEmbed, wasmEmbedBody(play.OutFile), cfg); err != nil {
		return err
	}

	if err := writeSearchIndex(outDir, all); err != nil {
		return err
	}
	if err := writeAssets(outDir); err != nil {
		return err
	}
	if err := copyLogo(repoRoot, outDir); err != nil {
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

// buildEmbeddedIDE builds the server-less Vuetify IDE demo (the full app:
// Playground / Notebook / IDE tabs) with a relative base and copies its output
// into <outDir>/<subDir>, so the docs site serves it in-browser. Requires bun
// and the workspace to be installed; the caller tolerates failure.
func buildEmbeddedIDE(repoRoot, outDir, subDir string) error {
	demo := filepath.Join(repoRoot, "web", "ide-vuetify", "demo")
	if _, err := os.Stat(demo); err != nil {
		return fmt.Errorf("demo not found: %w", err)
	}
	cmd := exec.Command("bash", "-c", "bun run wasm && bunx vite build --base=./")
	cmd.Dir = demo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return copyTree(filepath.Join(demo, "dist"), filepath.Join(outDir, subDir))
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
	// Header links / release banner.
	RepoURL     string
	TasksURL    string
	PlayHref    string
	ReleaseName string
	ReleaseURL  string
	HasRelease  bool
	Prism       bool
}

func writePage(outDir string, tmpl *template.Template, groups []navGroup, p *page, body template.HTML, cfg siteConfig) error {
	data := layoutData{
		Title:       p.Title,
		Groups:      groups,
		Active:      p.OutFile,
		Content:     body,
		TOC:         tocOf(p.Headings),
		Base:        baseFor(p.OutFile),
		RepoURL:     cfg.RepoURL,
		TasksURL:    cfg.tasksURL(),
		PlayHref:    cfg.playHref,
		ReleaseName: cfg.releaseName(),
		ReleaseURL:  cfg.releaseURL(),
		HasRelease:  cfg.hasRelease(),
		Prism:       cfg.prism,
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

// downloadAsset is one row of the Download page's asset table.
type downloadAsset struct {
	label string // human label, e.g. "Linux (amd64)"
	name  string // file name, e.g. gad_1.2.3_linux_amd64.tar.gz
	url   string // download URL
	desc  string // short description
	local bool   // served directly from the site (download attribute)
}

// downloadBody renders the Download page: a highlighted release banner, the
// release notes, and a table of downloadable assets (CLI binaries for
// linux/windows on amd64/arm64, both WASM modules, and checksums).
func downloadBody(cfg siteConfig) template.HTML {
	var b strings.Builder

	// Release banner — the release name is shown prominently.
	b.WriteString(`<div class="rel-hero">`)
	if cfg.hasRelease() {
		b.WriteString(`<div class="rel-badge">Release</div>`)
		fmt.Fprintf(&b, `<h1 class="rel-name"><a href="%s">%s</a></h1>`,
			template.HTMLEscapeString(cfg.releaseURL()), template.HTMLEscapeString(cfg.releaseName()))
		if cfg.ReleaseDate != "" {
			fmt.Fprintf(&b, `<div class="rel-date">%s</div>`, template.HTMLEscapeString(cfg.ReleaseDate))
		}
	} else {
		b.WriteString(`<div class="rel-badge">Latest</div>`)
		fmt.Fprintf(&b, `<h1 class="rel-name"><a href="%s">Latest release</a></h1>`,
			template.HTMLEscapeString(cfg.releaseURL()))
		b.WriteString(`<div class="rel-date">This is a development build; download links point at the latest published release.</div>`)
	}
	b.WriteString(`</div>`)

	// Release notes (rendered Markdown).
	if strings.TrimSpace(cfg.ReleaseNotes) != "" {
		notes, _ := renderMarkdown(cfg.ReleaseNotes)
		b.WriteString(`<section class="rel-notes"><h2>Release notes</h2>`)
		b.WriteString(notes)
		b.WriteString(`</section>`)
	}

	ver := strings.TrimPrefix(cfg.ReleaseTag, "v")
	archiveName := func(goos, arch string) string {
		ext := "tar.gz"
		if goos == "windows" {
			ext = "zip"
		}
		return fmt.Sprintf("gad_%s_%s_%s.%s", ver, goos, arch, ext)
	}

	// CLI binaries (goreleaser archives). Names include the version only when a
	// release tag is known; otherwise the "latest/download/<name>" URL resolves
	// server-side, so we still show a stable label.
	bins := []struct{ label, goos, arch, desc string }{
		{"Linux (amd64)", "linux", "amd64", "64-bit Intel/AMD Linux"},
		{"Linux (arm64)", "linux", "arm64", "64-bit ARM Linux"},
		{"Windows (amd64)", "windows", "amd64", "64-bit Intel/AMD Windows"},
		{"Windows (arm64)", "windows", "arm64", "64-bit ARM Windows"},
	}
	var assets []downloadAsset
	for _, bn := range bins {
		name := archiveName(bn.goos, bn.arch)
		assets = append(assets, downloadAsset{label: bn.label, name: name, url: cfg.assetURL(name), desc: bn.desc})
	}
	// WebAssembly module — served directly from the site (it is built into the
	// output dir), so it downloads even without a published release.
	assets = append(assets,
		downloadAsset{label: "WebAssembly", name: "gad.wasm", url: "gad.wasm", desc: "Browser/WASM module (run, format, diagnostics and the gadDebug* debugger)", local: true},
	)
	if cfg.hasRelease() {
		assets = append(assets, downloadAsset{label: "Checksums", name: "checksums.txt", url: cfg.assetURL("checksums.txt"), desc: "SHA-256 checksums of the release assets"})
	}

	b.WriteString(`<h2>Assets</h2><div class="dl-table"><table><thead><tr><th>Platform</th><th>File</th><th>Description</th></tr></thead><tbody>`)
	for _, a := range assets {
		dl := ""
		if a.local {
			dl = ` download`
		}
		fmt.Fprintf(&b, `<tr><td>%s</td><td><a href="%s"%s><code>%s</code></a></td><td>%s</td></tr>`,
			template.HTMLEscapeString(a.label), template.HTMLEscapeString(a.url), dl,
			template.HTMLEscapeString(a.name), template.HTMLEscapeString(a.desc))
	}
	b.WriteString(`</tbody></table></div>`)

	fmt.Fprintf(&b, `<p class="muted">All release assets are on the <a href="%s">GitHub releases page</a>.</p>`,
		template.HTMLEscapeString(cfg.releaseURL()))

	return template.HTML(b.String())
}

// codeBlock renders code as an HTML-escaped <pre><code> block, tagging both with
// a language-<lang> class so PrismJS highlights it.
func codeBlock(code, lang string) string {
	cls := ""
	if lang != "" {
		cls = ` class="language-` + lang + `"`
	}
	return "<pre" + cls + "><code" + cls + ">" + template.HTMLEscapeString(code) + "</code></pre>"
}

// homeIntroHTML builds the home page hero + the three-path quickstart (CLI,
// WASM, Go module) prepended to the rendered README. playHref is the site path
// of the Playground/demo.
func homeIntroHTML(cfg siteConfig, playHref string) template.HTML {
	var b strings.Builder
	b.WriteString(`<div class="home-hero">`)
	b.WriteString(`<img class="home-hero-logo" src="gad.svg" alt="Gad logo" width="96" height="96">`)
	b.WriteString(`<div class="home-hero-text"><h1>Gad</h1>`)
	b.WriteString(`<p class="tagline">A fast, dynamic scripting language embedded in Go — run it from the CLI, in the browser via WebAssembly, or embedded in your Go application.</p>`)
	b.WriteString(`<div class="home-cta">`)
	b.WriteString(`<a class="btn btn-primary" href="getting-started.html">Get started</a>`)
	fmt.Fprintf(&b, `<a class="btn btn-ghost" href="%s">Playground</a>`, template.HTMLEscapeString(playHref))
	b.WriteString(`<a class="btn btn-ghost" href="download.html">Download</a>`)
	fmt.Fprintf(&b, `<a class="btn btn-ghost" href="%s" target="_blank" rel="noopener">GitHub</a>`, template.HTMLEscapeString(cfg.RepoURL))
	b.WriteString(`</div></div></div>`)

	b.WriteString(`<h2 class="qs-title">Quickstart</h2>`)
	b.WriteString(`<p class="qs-sub">Three ways to run Gad — pick the one that fits your project.</p>`)
	b.WriteString(`<div class="cards">`)

	// Card 1 — CLI.
	b.WriteString(`<div class="card"><h3><span class="card-badge">⌘</span> Command line</h3>`)
	b.WriteString(`<p>Download the <code>gad</code> binary from the releases, or install it with Go:</p>`)
	b.WriteString(codeBlock("go install github.com/gad-lang/gad/cmd/gad@latest", "bash"))
	b.WriteString(`<p>Run, format and document:</p>`)
	b.WriteString(codeBlock("gad script.gad        # run a script\ngad fmt script.gad    # format in place\ngad doc               # generate workspace docs", "bash"))
	b.WriteString(`<a class="card-link" href="download.html">Download &amp; run →</a>`)
	b.WriteString(`</div>`)

	// Card 2 — WebAssembly.
	b.WriteString(`<div class="card"><h3><span class="card-badge">◈</span> WebAssembly</h3>`)
	b.WriteString(`<p>Grab <code>gad.wasm</code> + <code>wasm_exec.js</code> and run Gad in any browser:</p>`)
	b.WriteString(codeBlock("<script src=\"wasm_exec.js\"></script>\n<script>\n  const go = new Go();\n  WebAssembly.instantiateStreaming(\n    fetch(\"gad.wasm\"), go.importObject\n  ).then(r => {\n    go.run(r.instance);\n    const res = JSON.parse(gadRun(\"return 6 * 7\"));\n    console.log(res.result); // \"42\"\n  });\n</script>", "markup"))
	b.WriteString(`<a class="card-link" href="wasm-embed.html">Embed the WASM →</a>`)
	b.WriteString(`</div>`)

	// Card 3 — Go module.
	b.WriteString(`<div class="card"><h3><span class="card-badge">Go</span> Go module</h3>`)
	b.WriteString(`<p>Embed the compiler and VM in your Go program:</p>`)
	b.WriteString(codeBlock("go get github.com/gad-lang/gad", "bash"))
	b.WriteString(codeBlock("builtins := gad.NewBuiltins()\nst := gad.NewSymbolTable(builtins.NameSet)\ncr, _ := gad.Compile(st, []byte(`return 6 * 7`), gad.CompileOptions{})\nret, _ := gad.NewVM(builtins.Build(), cr.Bytecode).Run()\nfmt.Println(ret) // 42", "go"))
	b.WriteString(`<a class="card-link" href="embedding.html">Embedding guide →</a>`)
	b.WriteString(`</div>`)

	b.WriteString(`</div>`) // .cards

	b.WriteString(`<p class="muted">Configure a workspace via the <a href="ref-workspace-config.html"><code>GAD_CONFIG_DIR</code> (<code>.gad/</code>)</a> directory.</p>`)
	return template.HTML(b.String())
}

// wasmEmbedBody renders the "Embed the WASM" page: how to load the module and
// the JS API it exposes. playHref is the Playground/demo path (Web Worker note).
func wasmEmbedBody(playHref string) template.HTML {
	var b strings.Builder
	b.WriteString(`<h1>Embed the Gad WASM in your site</h1>`)
	b.WriteString(`<p>The Gad WebAssembly module runs the compiler, VM, formatter, doc extractor and debugger entirely in the browser — no server. It installs a small set of functions on the JS global object, each taking strings and returning a JSON string.</p>`)

	b.WriteString(`<h2>1. Get the files</h2>`)
	b.WriteString(`<p>Download <code>gad.wasm</code> and <code>wasm_exec.js</code> from the <a href="download.html">Download</a> page (both are attached to every release). <code>wasm_exec.js</code> is Go's WASM loader and must match the Go toolchain the module was built with — always ship the one from the same release.</p>`)

	b.WriteString(`<h2>2. Load it</h2>`)
	b.WriteString(codeBlock("<script src=\"wasm_exec.js\"></script>\n<script>\n  const go = new Go();\n  WebAssembly.instantiateStreaming(fetch(\"gad.wasm\"), go.importObject)\n    .then(r => { go.run(r.instance); onReady(); });\n\n  function onReady() {\n    // window.gadRun / gadFormat / gadDiagnose / gadDoc / … are ready.\n    const r = JSON.parse(gadRun('println(\"hi\"); return 6 * 7'));\n    console.log(r.stdout, r.result); // \"hi\\n\" \"42\"\n  }\n</script>", "markup"))
	fmt.Fprintf(&b, `<blockquote>A long-running script blocks the thread. For a responsive UI, load the module inside a <strong>Web Worker</strong> — exactly what the <a href="%s">Playground</a> does.</blockquote>`, template.HTMLEscapeString(playHref))

	b.WriteString(`<h2>3. The API</h2>`)
	b.WriteString(`<p>Each function returns a JSON string. <code>sourceType</code> selects the dialect: <code>""</code>/<code>"gad"</code> (plain Gad), <code>"gadTemplate"</code> (mixed <code>.gadt</code>) or <code>"gadx"</code>.</p>`)
	b.WriteString(`<div class="dl-table"><table><thead><tr><th>Function</th><th>Returns (JSON)</th></tr></thead><tbody>`)
	rows := [][2]string{
		{`gadRun(src[, sourceType[, argsJSON]])`, `{ ok, stdout, stderr, result, diagnostics }`},
		{`gadFormat(src[, sourceType])`, `{ ok, source, diagnostics }`},
		{`gadDiagnose(src[, sourceType])`, `{ diagnostics: [{line, column, message, severity}] }`},
		{`gadDoc(src, sourceType)`, `{ markdown } | { error }`},
		{`gadDocData(src, sourceType)`, `{ doc } | { error }`},
		{`gadEval(src, expr, repr)`, `{ ok, value, error, stdout }`},
		{`gadTranspile(src, mixed)`, `{ ok, source, diagnostics }`},
		{`gadInspect(session, expr, src)`, `{ ok, inspect } | { ok:false, error }`},
		{`gadDebugStart / gadDebugCommand / gadDebugEval / gadDebugStop`, `debugger stepping protocol`},
	}
	for _, r := range rows {
		fmt.Fprintf(&b, `<tr><td><code>%s</code></td><td><code>%s</code></td></tr>`,
			template.HTMLEscapeString(r[0]), template.HTMLEscapeString(r[1]))
	}
	b.WriteString(`</tbody></table></div>`)

	b.WriteString(`<h2>Prefer a ready-made editor?</h2>`)
	b.WriteString(`<p>The <a href="js-modules/ide-vuetify.html">@gad-lang/ide-vuetify</a> and <a href="js-modules/ide-react.html">@gad-lang/ide-react</a> packages wrap this module into a full IDE component (editor, run profiles, debugger, inspector), and <a href="js-modules/codemirror-gad.html">@gad-lang/codemirror-gad</a> / <a href="js-modules/prism-gad.html">@gad-lang/prism-gad</a> provide syntax highlighting.</p>`)
	return template.HTML(b.String())
}

// buildPrismBundle bundles PrismJS core + common languages + the Gad-family
// grammars (gad / gadt / gadx) into <outDir>/prism.js for static syntax
// highlighting. Requires bun; the caller tolerates failure (code blocks then
// render unstyled).
func buildPrismBundle(repoRoot, outDir string) error {
	pkg := filepath.Join(repoRoot, "web", "prism-gad")
	if _, err := os.Stat(filepath.Join(pkg, "site-bundle.mjs")); err != nil {
		return err
	}
	// bun runs from the package dir (to resolve prismjs + ./src); the outfile must
	// be absolute so it lands in the site output, not under the package.
	out, err := filepath.Abs(filepath.Join(outDir, "prism.js"))
	if err != nil {
		return err
	}
	cmd := exec.Command("bun", "build", "./site-bundle.mjs", "--outfile", out, "--minify", "--format=iife")
	cmd.Dir = pkg
	if o, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, o)
	}
	return nil
}

// copyLogo copies the Gad logo (assets/identity/gad.svg) into the site root as
// gad.svg (header brand, favicon, home hero). A missing logo is tolerated.
func copyLogo(repoRoot, outDir string) error {
	src := filepath.Join(repoRoot, "assets", "identity", "gad.svg")
	data, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "gad.svg"), data, 0o644)
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

// buildWASMAssets compiles the Gad WASM module (gad.wasm, debugger-enabled) and
// copies wasm_exec.js into the output directory so the Playground works offline /
// on GitHub Pages and the Download page can offer it.
func buildWASMAssets(repoRoot, outDir string) error {
	cmd := exec.Command("go", "build", "-o", filepath.Join(outDir, "gad.wasm"), "./web/wasm")
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gad.wasm: %v: %s", err, out)
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
