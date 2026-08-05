# TASK: Vuetify IDE — dockview panels, two v-models, Settings (TSX)

> Created: 2026-08-03 | Updated: 2026-08-05 06:00

## Goal
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
  Unverified: goreleaser itself (binary not installed) — config validated by
  YAML parse + manual review only; not run live in a browser

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| JSX.IntrinsicElements missing | no Vue JSX types | tsconfig jsxImportSource:"vue" | canary tc exit 0 |
| Vuetify props reject onClick/title/onKeyup | Vuetify types omit native attrs in TSX | src/vuetify.ts permissive re-exports | pkg tc exit 0 |
| Rollup can't resolve dockview-core css (lib) | lib shouldn't bundle theme css | consumer imports it; externalize dockview | pkg build ok |
| demo can't resolve dockview-core css | not a direct demo dep (bun isolation) | add dockview-core to demo deps | demo build ok |

## Current State
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
