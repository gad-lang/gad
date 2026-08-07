# TASK: Docs-from-samples — template-driven `gad doc` generation

> Created: 2026-08-06 | Updated: 2026-08-06

## Goal
Make the samples the single source of truth for language documentation. Move
language-feature docs into `samples/*.{gad,gadt,gadx}` as doc comments, and
generate the site docs from them via `gad doc` with customizable templates. No
more doc/sample drift.

## Plan
- [x] Dialect-aware doc templates: `--doc-template-md/html PATH` (.gad/.gadt/.gadx;
      .gad/.gadt write STDOUT, .gadx renders a tag tree); `--html` opt-in HTML;
      `--no-template` escapes to the built-in Go Markdown renderer
- [x] `go:embed` default templates (cmd/gad/doctemplates) + repo `./doc-templates`
      copies, guarded by a drift test; embedded md is the default so `gad doc`
      is template-driven out of the box
- [x] `@snippet NAME` incorporation from `//snippet NAME … //endsnippet` regions,
      fenced with the source dialect; `/**= EXPR **/` (value) and `/**< TEXT **/`
      (stdout) results verified at generation (fail on mismatch), then rendered
- [x] Per-directory index generation: README.md (+ index.html with --html) via
      md-index.gadx / html-index.gadx, listing files and linking subdir indexes
- [x] Fixed a real compiler panic: selector in a `{%= … %}` mixed island
- [x] Made the Go doc renderer mixed-aware (.gadt) and fixed samples/09_template.gad
- [x] Shebang `#!…` on line 1 of .gad (round-trips through `gad fmt`) and .gadt
      (ignored at run time)
- [x] Makefile `samples-doc` → `doc/samples/`; `generate-docs` depends on it
- [ ] JSON/YAML doc output (encode the DOC dict that feeds the templates)
- [ ] Migrate the 17 language docs + metaprogramming into the samples, remove from
      doc/, repoint doc/README.md (index → sample + generated doc)
- [ ] Incorporate doc/samples into the website (cmd/build-website)
- [ ] Follow-up: gadt/gadx shebang round-trip on format; .gadt module prose

## Log
### 2026-08-07 (content migration + 2 compiler bug fixes)
- MIGRATED into sample `/*** … ***/` headers (prose + @snippet regions with
  verified `/**= **/` / `/**< **/` results), doc/*.md removed, doc/samples
  regenerated. Commits: 1245013 (enums 20, error-handling 07), 19b09cb
  (doc-comments 16, special-keywords 29), 56ea81e (control-flow → 06+18),
  b2ae3c7 (properties 31), c1cbea4 (functions → 03+10), 515c81a (collections
  core → 04). b70f770 reverted snippet result rendering to Python-docs inline
  `// =>` / `Output:` (per user).
- COMPILER FIX (in c1cbea4): a call with a non-trailing positional spread
  (`f(*a, b)`, `f(1,*a,2,*b)`, incl. the arrow `func(*a;**k)=> …` form) recursed
  forever in the Gadx compile fallback and overflowed the stack — but only with
  the fallback installed, which CompileFile does by default for every .gad file
  run by the CLI (pure-compiler tests never hit it). Bounded the fallback
  recursion (compiler_gadx.go fallbackDepth cap). Regression
  TestCallManySpreadsWithGadxFallback. `go test ./...` exit 0.
- Verified API facts the result-markers surfaced (documented correctly now):
  `delete` is a STATEMENT (`delete m.a`), `arr += x` APPENDS x as one element
  (concat is `arr + x`), a keyValue array indexes by int (not key selector).
- REMAINING to migrate: collections comprehensions→05, keyValue→22,
  destructuring→27 (then rm collections.md); values-and-types→02,
  variables-and-scopes, operators (→14/15/17), classes→11/19,
  method-interfaces→12, interfaces→24, strings-bytes-regex→08, modules,
  templates→09/23, metaprogramming. Then repoint doc/README.md and wire
  doc/samples into cmd/build-website. Plus JSON/YAML doc output.

### 2026-08-06 (docs-from-samples infrastructure)
- Flags/renderer: `--doc-template-md/html`, `--html`, `--no-template` added
  (cmd/gad/doc_cmd.go); renderDocTemplate is now dialect-aware by template
  extension (cmd/gad/doc_template.go). `go test ./cmd/gad -run TestRenderDocTemplateDialects`
  → PASS (t.gad/t.gadt/t.gadx all render)
- Embedded defaults cmd/gad/doctemplates/{md,html,md-index,html-index}.gadx via
  go:embed + repo ./doc-templates copies; TestDocTemplatesInSyncWithEmbedded PASS
- Snippets + results: cmd/gad/doc_snippet.go; TestExtractSnippets,
  TestExpandSnippetsRunsAndVerifies, TestExpandSnippetsResultMismatchFails,
  TestExampleSourceStripsMarkersAndModuleDoc, TestDocCommandExpandsAndVerifiesSnippets
  all PASS
- Per-dir indexes: cmd/gad/doc_index.go; TestBuildIndexTree +
  TestDocCommandGeneratesIndexes PASS (root→sub links, nested)
- Compiler panic fix: mixed `{%= x.y %}` compiled the synthetic write() off the
  stack → pushSelector read c.stack[-1]. Fixed by pushing the call via atDo +
  guarding pushSelector (compiler.go, compiler_nodes.go). Regression test
  TestCompilerMixedSelectorInValueStmt PASS; `go test .` PASS
- Shebang: parser/scanner_scan.go scans a first-line `#!…` as a comment (normal
  mode, round-trips) and skips/keeps it in mixed mode. TestScanner_Shebang PASS.
  Verified via bridge: `#!/usr/bin/env gad\nx:=1\nprintln(x)` runs → "1\n"; fmt
  preserves the shebang on line 1; a mid-file `#!` stays an error
- samples/09_template.gad: removed the leading space before `# gad: mixed` so the
  config directive is honoured (prose captured, parses); Go renderer FromContent
  made mixed-aware for .gadt (doc.go)
- Makefile: `samples-doc` now `cd samples && gad doc --out ../doc/samples
  --doc-template-md ../doc-templates/md.gadx .` (flags BEFORE the positional, or
  Go's flag pkg ignores them); `generate-docs: … samples-doc`. Removed legacy
  samples/.gad/doc-templates and stale samples/doc
- EVIDENCE (full suite): `go build ./...` exit 0; `go test ./...` (root) exit 0;
  `go test ./...` in gadx exit 0; `go vet` clean; `gofmt -l` clean. `make
  samples-doc` → 33 files in doc/samples (32 samples + README.md index);
  doc/samples/09_template.md has prose, README.md lists all files

## Previous task — Vuetify/React IDE + release/site ops (history)
Rework @gad-lang/ide-vuetify: resizable/movable dockview panels; a Settings
dialog (Panels/Formatter/Transpile/Template); two independent v-models —
`layoutConfig` (dockview layout) and `config` (settings) — persisted to
LocalStorage by the demo; hover-locals like `gad ide`. Apply the layout
save/restore to the React `<Ide>` too. Components authored in TSX (chosen over
.vue/.jsx: keeps full TS typing, matches the React package).

## Log
### 2026-08-04
- TSX toolchain validated — added @vitejs/plugin-vue-jsx + tsconfig
  jsxImportSource:"vue"; vue-tsc canary passed
- Extracted controller.ts (framework-agnostic state/actions), provided via
  inject; 5 dockview panels (Explorer/Editor/CallStack/Locals/Output) inject it
- GadIde.tsx: top toolbar (Format/Doc/Run/Debug/Settings + step controls) +
  DockviewVue; v-model:layoutConfig (toJSON/fromJSON, echo-guarded) and
  v-model:config; Settings + Doc dialogs
- Vuetify+TSX friction (native onClick/title/onKeyup not typed) → contained in
  src/vuetify.ts (permissive re-exports); our own types stay strict
- dockview theme CSS is the consumer's import (like Vuetify) → removed from the
  lib, added to the demo; externalized /^dockview/ and /^vuetify/ in the lib build
- Hover-locals wired (editor getLocals ← paused frame); inspect icons + tree
  dialog retained
- React `<Ide>`: added optional `layoutConfig` + `onLayoutConfigChange`
  (additive, ref-based) restoring/saving the dockview layout
- Verified: ide-vuetify typecheck + lib build; demo typecheck + production build
  (dockview + Vuetify + worker); ide-react + app typecheck — all clean

### 2026-08-05
- Editor font-size controls — GadEditorView fontComp compartment + setFontSize();
  GadIde `fontSize` v-model (clamped 8–32 in controller); A−/A+ on the right of
  the editor status bar (flex row, path ellipsis w/ min-width:0 so a long path
  never triggers horizontal scroll); demo persists size to LocalStorage
- gad SVG icon (gad-24.svg) beside the "GAD" title in the demo app bar
- Verified: `bun run typecheck` (pkg + demo) exit 0; `bun run build` (pkg,
  ide-vuetify.js 95.47 kB) + `bunx vite build` (demo) both built ok →
  commit de87d28

- Playground Output pane moved to the right (editor left) → commit 593b216
- Playground sourceType not reaching execution — the ide-vuetify demo runner
  dropped the 2nd arg on run/diagnose (and format didn't take one). Runner now
  forwards sourceType on all three; `GadRunner`/Playground already passed it
- `gad.Compile` always lowers Gadx nodes — CompileFile installs
  gadxCompileFallback by default when FallbackFunc is nil (works even with
  GadxOptions nil); removed the now-redundant default in the GadxOptions branch
- `gadbridge.FormatSource(src, sourceType)` + wasm `gadFormat(src[, sourceType])`
  + client `format(src, sourceType)`: gadx→Gadx syntax, gadTemplate→mixed,
  gad/""→plain Gad
- Verified: `go test ./...` (root) + `go test ./...` (gadx submodule) no failures;
  `go build ./...` ok; new bridge tests TestFormatSource{Template,Gadx,
  GadxParseError} pass (`go test ./web/gadbridge -run TestFormat` → ok);
  rebuilt demo gad.wasm; ide-vuetify pkg + demo `bun run typecheck` exit 0;
  demo `bunx vite build` ok → commit 8cf228b

- Two WASM builds via `gadwasmdebug` tag: gad.wasm (no debugger) +
  gad_debug.wasm (gadDebug* protocol). web/wasm/debug.go (tagged) /
  debug_off.go (no-op) split; main.go calls registerDebug(). Shared
  scripts/build-wasm.sh (normal|debug|both). app/demo scripts build the debug
  variant (their IDE needs it). Makefile build-wasm→./dist both; dist deps on it
- Docs Download page: release banner (name highlighted) + rendered notes + asset
  table (linux/windows amd64/arm64 archives, both WASMs local, checksums).
  build-website flags --repo-url/--tasks-url/--release-*. Header links on every
  page: Repo, Tasks (TASK.md), Download + release chip. website.yml passes
  release fields via env (injection-safe)
- goreleaser: CLI restricted to linux+windows × amd64+arm64; gad.wasm/
  gad_debug.wasm/wasm_exec.js attached via release.extra_files (before hook
  builds them into ./wasm-assets)
- Verified: `go build ./...` ok; both `GOOS=js GOARCH=wasm go build` (normal +
  -tags gadwasmdebug) compile; `go test ./cmd/build-website` ok; full site build
  with --release-tag emits gad.wasm+gad_debug.wasm+wasm_exec.js, /ide embedded,
  Download page has highlighted release name + local wasm download links + header
  Repo/Tasks/Download; no-release build falls back to "Latest" (no chip); YAML of
  .goreleaser.yml + website.yml parse; gofmt clean → commit 2940bba
- Installed goreleaser v2 (`go install github.com/goreleaser/goreleaser/v2@latest`)
  and ran `goreleaser check` → "1 configuration file(s) validated" (exit 0)
- `goreleaser release --snapshot --clean` → "release succeeded after 29s" (exit 0):
  built 4 binaries (linux/windows × amd64/arm64), 4 archives
  (dist/gad_0.0.4-next_{linux_amd64.tar.gz,linux_arm64.tar.gz,windows_amd64.zip,
  windows_arm64.zip}) + checksums.txt; the WASM before-hook produced
  ./wasm-assets/{gad.wasm 20.5MB, gad_debug.wasm 20.6MB, wasm_exec.js} — the
  extra_files that attach on a real release (upload skipped in snapshot mode)
- DISCOVERED pre-existing goreleaser bug (present at HEAD~3, NOT from this task):
  archives `files: doc/**/*` and `gadx/docs/**/*` match zero files (goreleaser
  fileglob needs `doc/**`, not `doc/**/*`, to include files directly in doc/ —
  all 31 doc files are top-level, no subdirs), so release archives ship without
  the docs (only LICENSE, README.md, gad). RESOLVED by user direction: docs are
  not shipped in the CLI archives at all (Pages only), so the doc globs were
  removed from archives.files rather than fixed (commit e1f4e60)

### 2026-08-05 (afternoon — website/wasm overhaul)
- Collapsed WASM to a single debugger-enabled gad.wasm (the two builds were the
  same size). Removed the gadwasmdebug split (debug_off.go deleted; debug.go
  always compiled). scripts/build-wasm.sh, app/demo scripts, Makefile,
  build-website, goreleaser all build/ship one gad.wasm → commit e1f4e60
- goreleaser extra_files = gad.wasm + wasm_exec.js (docs out of archives)
- Website "Playground" menu now hosts the full ide-vuetify demo (built into
  /playground; simple playground is the bun-missing fallback); redundant "IDE"
  menu removed. website.yml gained bun setup
- Modern layout keyed to the logo palette (navy/cyan #06B6D4/amber #D97706): logo
  in header + favicon (from assets/identity/gad.svg), gradient accents, cards,
  restyled nav/code/tables/hero
- Home hero + 3-path quickstart (CLI run/fmt/doc, WASM embed, Go module) +
  GAD_CONFIG_DIR link; new "Embed the WASM" page (JS API table). GAD_CONFIG_DIR
  is already the "Workspace Configuration" reference page, now linked from home
- Docs API refresh (commit 526fc66): embedding examples used a pre-refactor API
  (Compile→3 returns, NewVM 1-arg, Run(globals,args), gad.Map/SyncMap,
  gad.String, `exports = {}`). Fixed embedding.md/metaprogramming.md/README.md +
  rewrote doc/tutorial.md snippets to Compile→*CompileResult/.Bytecode,
  NewVM(builtins.Build(), bc), Run/RunOpts, gad.Dict/SyncDict, gad.Str, `export`
- Verified: `go build ./...` ok; single wasm compiles; build-website renders home
  hero+cards, /playground demo (full build), wasm-embed page, logo, one gad.wasm;
  every rewritten embedding example compiled+run: [2,4,6,8], fib(35)=9227465,
  "big", {fn1:1,fn2:1}. gofmt clean; YAML parses
  Unverified: live GitHub Pages deploy + a real goreleaser release; site not
  opened in a real browser

### 2026-08-05 (evening — Prism highlighting + fence sourceTypes)
- Header: Tasks link → repo /issues (57f3740); Playground added to header nav
  before Download, theme toggle stays right-most (644e58f)
- `make website` / `make website-fast` targets (57a7fcf); tested by serving +
  curl (/ 200, download/gad.svg/styles.css 200)
- PrismJS highlighting: web/prism-gad/site-bundle.mjs bundles Prism core +
  go/json/bash/yaml + gad/gadt/gadx grammars; build-website builds it best-effort
  with bun into prism.js (buildPrismBundle; abs outfile path), loads it, and adds
  palette-matched token colors. Verified the bundle registers gad/gadt/gadx/go
  and highlights (runtime eval test)
- Fence sourceTypes: reclassified doc/*.md — 282 `go`→`gad` (real Go embedding
  kept `go` via markers: package/import/func main/gad.<Sym>/interface/[]byte);
  embedding.md/metaprogramming.md Go untouched. gadx/docs `gadx`→(temporarily
  gad-gadx then) `gadx`; templates.md mixed-template blocks → `gadt`. Final
  decision: fences use plain gad/gadt/gadx (Prism registers those natively; no
  gad-gadt/gad-gadx aliases)
- Caught + fixed misfires: gadx docs `go`→`gad` (11 blocks were real Go using
  `gadx.`/aliased imports/go.mod — reverted, only gadx→gadx normalized);
  metaprogramming block with `{%` inside a string literal (kept `gad`, not gadt)
- Added templates.md to the Guide nav ("coloque templates no site")
- Verified: build-website renders functions.html=24×language-gad,
  embedding.html=10×language-go, templates.html=6×language-gadt,
  gadx-syntax=31×language-gadx, ref-workspace-config gad/gadx/sh/yaml; no gad-gad
  left; `go build ./cmd/build-website` ok; gofmt clean. NOTE: gadx is NOT a
  submodule (no gadx/.git) — its files are tracked directly in this repo; all
  commits landed here

### 2026-08-05 (test release v0.0.4-test1)
- Pushed main (65 commits) to origin, tagged v0.0.4-test1 (prerelease)
- 1st release run FAILED: before-hook `make goreleaser-setup`→web-build errored
  on a clean checkout — web/app imports @gad-lang/* as workspace:* (need their
  dist/.d.ts) and src/samples.gen.ts is git-ignored (generated). My earlier local
  `--snapshot` passed only because those artifacts were already on disk
- Fix (commit 79d9034): web-build runs `bun run plugins:build` first; app gained
  gen-samples + pre(dev|build) hooks. Verified from a simulated-clean state
  (removed dists + samples.gen): `make web-build` AND full `make goreleaser-setup`
  (incl. VS Code vsix) → exit 0
- Deleted+recreated the tag at the fixed HEAD; 2nd release run SUCCEEDED:
  https://github.com/gad-lang/gad/releases/tag/v0.0.4-test1 (prerelease) with
  assets: checksums.txt, gad.wasm, wasm_exec.js, and gad_0.0.4-test1_{linux_amd64,
  linux_arm64,windows_amd64,windows_arm64}.{tar.gz,zip}

### 2026-08-05 (release notes docs link + live-site fixes)
- Release footer: goreleaser `release.footer` links the docs — leads with
  /<tag>/ (· latest). Republished v0.0.4-test1 (deleted release+tag, recreated at
  HEAD) so its notes carry the footer. Verified via `gh release view`:
  "📖 Documentation: https://gad-lang.github.io/gad/v0.0.4-test1/ · latest"
- website.yml: token-created releases don't fire the `release` event, so /latest
  (root redirect target) went stale & had no banner. Now every main push refreshes
  /latest AND publishes under the resolved release tag dir (/<tag>/), and resolves
  the newest release via `gh release list` to fill the banner. Verified on
  gh-pages: /latest + /v0.0.4-test1 have home-hero + rel-chip "v0.0.4-test1"
- Two CI-only best-effort failures found + fixed (worked locally only because
  artifacts were on disk):
  1. website.yml set up bun but never `bun install` → prism bundle (no prismjs)
     and demo (no vite) both failed. Added `cd web && bun install --frozen-lockfile`
  2. buildEmbeddedIDE ran `bunx vite build` skipping the demo's samples.gen.ts
     regen (git-ignored) → "Could not resolve ./samples.gen". Prepended
     `bun run samples`
- Verified live (gh-pages/latest, bypassing CDN cache): prism.js present +
  functions.html loads it with 24×language-gad; playground/ is the full demo dir
  (nav → playground/index.html); templates/download/wasm-embed/gad.svg all
  published; final website run has zero warnings

### 2026-08-05 (migrate Pages to the org root site)
- Moved docs publishing from gad repo gh-pages (/gad/) to the org site repo
  gad-lang/gad-lang.github.io, served at root https://gad-lang.github.io/
- Deploy key: generated ed25519 keypair; public → write deploy key on
  gad-lang.github.io (id 159365775); private → secret ACTIONS_DEPLOY_KEY on gad;
  local private key shredded. website.yml deploys via peaceiris deploy_key +
  external_repository=gad-lang/gad-lang.github.io, publish_branch=main
- goreleaser footer URL dropped /gad → https://gad-lang.github.io/<tag>/
- Verified: website run exit 0; gad-lang.github.io/main got index.html(redirect)
  + latest/ (51 files incl prism.js/gad.wasm/playground) + <sha>/; live root
  https://gad-lang.github.io/ → 200 (redirects to ./latest/), /latest/ → 200
- gad repo Pages still serves the stale /gad/ snapshot. API disable is blocked
  (422 "not allowed" — legacy branch source). User will disable it manually via
  Settings → Pages (Source: None); if org policy blocks that, fall back to a
  gh-pages redirect to the root site. Left gh-pages untouched. commit a37d0e2

### 2026-08-05 (ide-react parity with ide-vuetify — IN PROGRESS)
- Goal: make @gad-lang/ide-react an identical replica of ide-vuetify, running
  with OR without a backend; cmd/gad ide lets the user pick server/server-less but
  its explorer stays real files. Decisions: add features to existing (already
  dockview-based, MUI) Ide.tsx; gad ide = hybrid (Go serves real files always,
  flag chooses compute Go-vs-WASM); port Playground+Notebook too; reusable
  components like vuetify; similar layout
- DONE: GadPlayground + GadNotebook as reusable React components (commit 70cce4f);
  upload.ts + fileTypes.ts (FileTypeRegistry) + UploadedFile type foundations
  (commit 82aa931). Both typecheck+build green
- DONE (more): reusable MUI dialogs DirTree/UrlImportDialog/UploadReviewDialog/
  PromptDialog+ConfirmDialog (commit 9772d45); wired upload button + drag-drop +
  URL import + in-app prompt into <Ide> (readonly + onUpload props; upload/
  uploadUrl/archiveKind/pathExists) (commit e43091b); editor read-only via
  readonly prop (commit 9069077). All typecheck+build green
- DONE (#57 hybrid): `gad ide --serverless` — Go Server.Compute field + flag;
  /api/ide/workspace reports compute "server"|"wasm"; web/app main.tsx builds a
  hybrid IdeApi (files→httpIdeApi, compute→localIdeApi/WASM) when wasm. Verified
  end to end: workspace endpoint = wasm under --serverless, server by default; go
  build+vet ok; ide-react + web/app typecheck. Run/debug/format/diagnose already
  pass `source` (editor content, saved to the real file first), so WASM compute
  needs no local FS. Known limit: cross-file imports need server mode (WASM can't
  read real files for imports) — same as the pure webide. commit 1c88ae9
- DONE (#56 polish, commit 5733ab6): autosave prop (false|true debounced|number
  interval ms); tabNameMax prop (truncate + full-name tooltip); font-size moved to
  a thin editor status bar (path ellipsis, no h-scroll, A−/A+ right); dirty tabs
  colored name + ●/✕ close; active-file explorer highlight already present;
  rename/remove tree actions (menu + F2/Delete) hidden when readonly. ide-react
  build + web/app typecheck green
- ALL THREE PHASES DONE: #55 Playground/Notebook, #56 upload/dialogs/readonly/
  polish parity, #57 hybrid gad ide --serverless. ide-react is now a
  feature-parity replica of ide-vuetify (MUI), runs with backend or server-less,
  and cmd/gad ide offers both modes with a real-file explorer
- Note: ide-react is MUI-based (not Vuetify); icons via @mui/icons-material, so
  fileTypes mdi-* icon strings aren't used for React icons (language part is)

### 2026-08-05 (ModuleSpec.Flags + raw argv)
- Replaced ModuleSpec `Main bool` → `Flags ModuleFlags` bitmask
  (ModuleMain/ModuleRawArgv) + IsMain()/IsRawArgv()/Has(). Name chosen by user:
  ModuleFlags/ModuleMain/ModuleRawArgv
- Compiler sets ModuleRawArgv when params are a lone variadic `param (*argv)`
  (Params.Len()==1 && Variadic() && NamedParams.Len()==0); independent of
  ModuleMain (user: "não precisa ser main")
- cmd/gad: for a ModuleRawArgv module, pass args straight through, argv[0]=module
  path (s.modulePath), no ParseArgs; when main, drop the first bare `--` options
  terminator (dropFirstOptionsTerminator). A CLI-run script is now flagged
  ModuleMain
- Encoder: Flags serialized as int (no bytecode compat, per user)
- Verified: go build ./... + go test ./... (root) + gadx submodule all green; vet
  + gofmt clean; manual E2E: `gad a/b/s.gad x --y=1 -- z` → argv
  ["a/b/s.gad","x","--y=1","z"]; normal `param (name; count=0)` still parses
  --count=5→int. Tests: module_flags_test.go, cmd/gad/argv_test.go. Docs:
  getting-started.md; sample 32_raw_argv.gad. commit cfbf6f8
- `@main` now consistent (commit 172de86): was folding to false in main modules
  because the optimizer built its OpIsMain-folding spec from name+URL only,
  dropping Flags. Fixed by copying c.module.Flags in compiler.optimize(). With
  cmd/gad flagging CLI scripts ModuleMain, `@main` is now true in the entry module
  (incl. `param (*argv)`) and false in imports. Regression test added

### 2026-08-05 (gadx tag-encode JSON/YAML)
- Bug: running a .gadx echoed `⇦ gadx.Tag()` instead of rendering. Root cause was
  the return (a gadx.Element) not being rendered: cmd/gad execute() discarded it
  (printed nothing) and web/ide server run set res.Result=ret.ToString(). Both
  fixed to render the Element to stdout
- Feature: tag-encode mode. gadx.ElementData(el)→gad.Object tree {tag,attrs,
  children}; gadx.EncodeElement(el,"json"|"yaml") marshals it (shared). Instead of
  HTML render, encode the returned tag as JSON/YAML
- Backend: gadbridge.RunSourceArgs + wasm gadRun gain a tagEncode arg; web/ide
  runRequest gains tagEncode; cmd/gad `--tag-encode json|yaml` flag. commit 32487ed
  (+ server fix 2abdf58)
- Frontend selector (Render|JSON|YAML), gadx-only: ide-react (RunConfig/RunProfile
  + Run/Debug dialog + GadPlayground/GadNotebook) commit 0500e4b; ide-vuetify
  (RunProfile/RunTarget + RunProfileDialog + GadPlayground/GadNotebook, VSelect
  added) commit bf1937c. GadRunner.run + all wasm clients (app + both demos) take
  tagEncode
- Verified: go build ./... + wasm build; go test (root/cmd/gad/gadbridge/ide/gadx)
  green; CLI E2E: gad p.gadx → HTML, -tag-encode json/yaml → encoded; ide-react +
  ide-vuetify + both demos typecheck+build; demo gad.wasm rebuilt. Tests:
  bridge (html/json/yaml), cmd/gad (render+encode)

### 2026-08-05 (interface context-function members `:Expr <header>`)
- New interface member `:Expr <(params)>` / `:Expr { … }` validating a free
  function in scope handles the interface's object. `@self` = interface type
  (a type placeholder in a positional param); >=1 `@self` per header (compile
  error otherwise); block headers all required; each `:Expr` checked
  independently. Expr captured BY VALUE where declared (supports locals +
  selectors) via new opcode OpInterfaceBind
- Phase 1 (parser/AST/coder) commit 9a08c13; Phase 2 (runtime Interface.
  ContextFuncs + InterfaceContextFunc + TypedIdent.Self, compiler build+bind,
  CanAssignVM check via SplitCaller, OpInterfaceBind + delve regen, encoder,
  tests) commit 5daabb8. Docs doc/method-interfaces.md + sample
  samples/24_interfaces.gad
- Interface can be built directly in Go (no symbols): set ContextFuncs Fn +
  resolved Types + Self; tested (TestInterfaceContextFuncGoBuilt)
- DONE: runtime interface-satisfaction cache (commit 87bfa9c). Memoized on the
  root VM keyed by (interface, value's ObjectType); shared by sub-VMs via
  pool.root; GC'd with the VM. Only class instances + reflected Go values cached
  (dict keys vary → never cached). Exported InterfaceSatCache +
  NewInterfaceSatCache + (*VM).SetInterfaceSatCache (build/inject/pre-warm
  outside the VM). Tests: cacheability, short-circuit, dict exclusion, sub-VM
  sharing, injection
- DONE: gadx.Render reuses the cache per compiled template + resets on recompile
  (commit b344b67); docs gadx/docs/api.md + doc/method-interfaces.md; test
  TestRenderInterfaceCacheReused
- Plugins (web/): no change needed — prism/codemirror already highlight the new
  syntax (verified by tokenizing: interface=keyword, :=operator, @self=class-name
  type). The updated samples/24_interfaces.gad is served by both plugin demos
- Verified: go test ./... (root) + gadx submodule + encoder + parser green; vet
  + gofmt clean; check-delve up to date; sample runs

### 2026-08-06 (release/site ops)
- CI `test` was failing on staticcheck ST1003 (pre-existing __crN vars from the
  b7c307e refactor). Renamed __crN→crN across 23 test files (commit 1cdd87a); test
  workflow green
- Notebook (react+vuetify) opens with one example per source type GAD/GADt/GADx
  (commit b56b4ac); Playground already had per-dialect samples. Samples verified
  to compile+run via the bridge
- Released v0.1.0-rc.1 (prerelease). Site: cleaned gad-lang.github.io (orphan
  commit, ~380MB of accumulated per-SHA/latest dirs dropped, kept README+.nojekyll)
  then republished; /latest + /v0.1.0-rc.1 live (HTTP 200), banner shows the tag
- release.yml now dispatches website.yml after publishing (commit 53a12f1):
  actions: write + `gh workflow run website.yml` — workflow_dispatch is the
  exception where GITHUB_TOKEN may trigger a run, so /<tag> auto-publishes on each
  release (previously needed a manual dispatch)

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| JSX.IntrinsicElements missing | no Vue JSX types | tsconfig jsxImportSource:"vue" | canary tc exit 0 |
| Vuetify props reject onClick/title/onKeyup | Vuetify types omit native attrs in TSX | src/vuetify.ts permissive re-exports | pkg tc exit 0 |
| Rollup can't resolve dockview-core css (lib) | lib shouldn't bundle theme css | consumer imports it; externalize dockview | pkg build ok |
| demo can't resolve dockview-core css | not a direct demo dep (bun isolation) | add dockview-core to demo deps | demo build ok |

## Current State
The docs-from-samples INFRASTRUCTURE is complete and fully tested; the content
MIGRATION and website wiring are not yet done.

`gad doc` is now template-driven by default: it renders each source through an
embedded (or workspace/flag-overridden) template that may be written in any Gad
dialect. Templates receive `param (doc dict)` = `{name, file, lang, source,
prose, sections}` and can incorporate real code via `@snippet NAME` placeholders
resolved from `//snippet … //endsnippet` regions; a region may assert a value
(`/**= EXPR **/`) or STDOUT (`/**< TEXT **/`) that is executed and verified at
generation time. Each output directory also gets an index (README.md, plus
index.html under --html). The official templates live in ./doc-templates and are
byte-identical to the go:embed defaults (drift-tested). `make samples-doc`
generates doc/samples/*.md (+ README.md) from samples/*, and `generate-docs`
depends on it. All of root+parser+gadx+cmd/gad tests pass, vet/gofmt clean, and a
pre-existing mixed-mode compiler panic (selector inside `{%= … %}`) was fixed
with a regression test. Shebang `#!` lines are ignored at run time and preserved
by `gad fmt` for .gad.

NOT done yet: (1) JSON/YAML doc output; (2) moving the language-feature prose out
of doc/*.md into the sample doc comments and removing those doc/*.md (repointing
doc/README.md); (3) incorporating doc/samples into cmd/build-website; (4) gadt/
gadx shebang round-trip on format. These are the next steps.

## Previous Current State — IDE work (history)
@gad-lang/ide-vuetify is now a TSX package: a `controller.ts` holds all reactive
state/actions and is provided via inject to five dockview-vue panels (Explorer,
Editor, Call Stack, Locals, Output) that are resizable, movable, dockable and
tabbable. GadIde.tsx adds a top toolbar (Run/Debug/Format/Doc/Settings + step
controls) and exposes two independent v-models — `layoutConfig` (dockview
toJSON/fromJSON) and `config` (the Settings document: Panels/Formatter/Transpile/
Template). Hovering an identifier while paused shows its value; Locals/Evaluate
rows have an inspect icon opening the value tree navigator. Vuetify+TSX native-
attribute friction is contained in src/vuetify.ts (permissive re-exports) so our
own code stays fully typed. The demo (App.tsx) binds both v-models to
LocalStorage (and seeds config from the backend). The dockview theme CSS is
imported by the host (demo main.ts), like Vuetify's. The React `<Ide>` gained
additive `layoutConfig`/`onLayoutConfigChange` props that restore/persist its
dockview layout. Everything typechecks and builds (package lib, demo production,
ide-react, app). The editor now has a `fontSize` v-model (A−/A+ controls on the
right of the status bar, clamped 8–32px, persisted in the demo) and the demo app
bar shows the gad SVG icon. NOT verified live in a browser — dockview-vue
rendering, provide/inject teleport, Vuetify interactions and the new font/icon UI
need a browser check (run `make ide-vuetify-demo`).
