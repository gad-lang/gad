# TASK: Structured docs (JSON) + doc templates + config relocation to .gad/

> Created: 2026-08-03 | Updated: 2026-08-03 15:05

## Goal
Make documentation generation return a structured (JSON-serializable) representation
instead of only Markdown, usable by both the WASM `gadDoc`/`gadDocData` bridge and the
`gad doc` CLI (which can now render HTML/Markdown via user giom templates). Relocate all
workspace config to `WORK_DIR/.gad/` (gad.yaml, ide.yaml, doc-templates/), driven by a new
central `gadconfig` package and `GAD_WORK_DIR` / `GAD_CONFIG_DIR` env vars. Clean cut — no
legacy `.gad.yaml` fallback.

## Plan
- [x] `web/gadbridge/doc.go`: `DocData`/`DocSection`/`DocSymbol` structs + `ExtractDoc` +
      `RenderMarkdown` + `DocData.GadDict()` (gad dict for templates)
- [x] WASM: add `gadDocData(source, sourceType)` export next to `gadDoc`
- [x] `gadconfig` package: WorkDir/Dir/File/IDEFile/DocTemplatesDir/DocHTMLTemplate/DocMDTemplate
- [x] `gad doc` template rendering: `cmd/gad/doc_template.go` (renderDocTemplate + resolve + outputs)
- [x] Wire templates into `processFile` (md.giom replaces .md, html.giom adds .html)
- [x] Relocate config reads/writes: cmd/gad (cmd.go, env.go, doc_cmd.go), web/ide (config.go, run.go)
- [x] Move `samples/.gad.yaml`→`samples/.gad/gad.yaml`; templates→`samples/.gad/doc-templates/{html,md}.giom`
- [x] Migrate tests (env_test, doc_cmd_test, ide_test) to gadconfig paths
- [x] Docs: doc/workspace-config.md (new layout + doc-templates section) + short refs + UI labels

## Log
### 2026-08-03
- Restructured gadbridge doc to JSON structure — `go build ./web/gadbridge/` exit 0;
  `go test ./web/gadbridge/` → ok (TestDocGad, TestDocGiom)
- Added WASM `gadDocData` — `GOOS=js GOARCH=wasm go build -o …/gad_wasm_check.wasm ./web/wasm/`
  exit 0, 20556625 bytes
- Added doc templates + rendering — `go test ./cmd/gad/ -run 'TestRenderDocTemplate'` →
  PASS (MD + HTML); `-run TestDocCommandUsesTemplates` → PASS (emits doc/m.md + doc/m.html)
- Config relocation to `.gad/` via `gadconfig` — full suite green:
  `go test ./...` (root) → no failures; `cd giom && go test ./...` → no failures
- Format/vet/build clean — `gofmt -l .` (excl. giom submodule): empty; `go vet ./...`: clean;
  `go build ./...`: exit 0

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| `DocData redeclared` | struct and extractor both named DocData | renamed extractor to `ExtractDoc` | `go build ./web/gadbridge/` exit 0 |
| `unknown field GiomOptions in CompileOptions{}` | GiomOptions/ModuleFile live on embedded CompilerOptions | assign after `opts := CompileOptions{}` | `go build ./cmd/gad/` exit 0 |
| template render `NotIndexableError: nil` | `param (doc dict)` is positional, passed via NamedArgs | pass as `Args{Array{doc.GadDict()}}` | TestRenderDocTemplateMD PASS |
| html template empty output | `@main` giom returns a giom.Element, not StdOut | `el.WriteTo(vm, &out)` on the return | TestRenderDocTemplateHTML PASS |
| `undefined: defaultCfgFile` in env_test | const removed during relocation | tests use `gadconfig.File(dir)` + MkdirAll | `go test ./cmd/gad/` ok |

## Current State
Documentation generation now produces a structured `DocData` (prose + typed sections of
symbols with name/signature/doc/line/column), JSON-serializable. `gadbridge.ExtractDoc` is the
extractor; `RenderMarkdown` is the default renderer; `DocData.GadDict()` yields the gad dict a
giom template consumes via `param (doc dict)`. WASM exposes both `gadDoc` (markdown) and
`gadDocData` (structure). `gad doc` renders through optional workspace templates at
`.gad/doc-templates/md.giom` (overrides the .md) and `.gad/doc-templates/html.giom` (adds a
.html); absent templates keep the built-in Markdown. All workspace config moved under
`WORK_DIR/.gad/` (`gad.yaml`, `ide.yaml`) via the new `gadconfig` package honoring
`GAD_WORK_DIR`/`GAD_CONFIG_DIR`; clean cut, no legacy fallback. Full test suites for the root
and giom modules pass; gofmt/vet/build clean; WASM builds. `gad ide` untouched behaviorally
(only config path source changed). Not yet committed. Newest still-open thread from earlier in
the session (unrelated to this task): the app WASM debug backend + web-only embeddable IDE page
wiring (tasks #50/#51).
