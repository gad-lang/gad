# TASK: @gad-lang/ide-vuetify component + server-less Vuetify demo

> Created: 2026-08-03 | Updated: 2026-08-03 22:10

## Goal
Create a reusable Vuetify 3.8 IDE component (`web/ide-vuetify`, the Vuetify
counterpart of `@gad-lang/ide-react`, same injectable IdeApi contract) and a
server-less demo app (`web/ide-vuetify/demo`) like the React `webide.html`:
in-browser only (WebFS samples + LocalStorage overlay, Gad WASM in a Web Worker).

## Plan
- [x] Package: api.ts (IdeApi contract + httpIdeApi + probeIde), types.ts
- [x] Package: codemirror.ts (langOf/langExtension + GadEditorView) + CM6 helpers
      (breakpointGutter, debugDecorations, docMarkdown copied from ide-react)
- [x] Package: GadEditor.vue (CodeMirror wrapper) + GadIde.vue (explorer + editor
      + Run/Doc/Debug panels, injectable `api`, optional `onReset`)
- [x] Package: index.ts, package.json, tsconfig(+build), vite lib config, env.d.ts, README
- [x] Demo: localIde.ts (in-browser IdeApi over WebFS + WASM worker), App.vue,
      main.ts (Vuetify bootstrap), index.html, vite config, tsconfig
- [x] Demo: copy wasm client/worker + webfs + gen-samples (+ build-wasm.sh); wasm/shared.ts
- [x] Add both to web/ bun workspaces; add @lezer/* direct deps for the build

## Log
### 2026-08-03
- Package typecheck clean — `bun run typecheck` (vue-tsc) → exit 0
- Package builds — `bun run build` (vite lib + vue-tsc d.ts) → dist/ide-vuetify.js
  32.65 kB + dist/index.d.ts + dist/style.css
- Demo samples generated — `bun run scripts/gen-samples.ts` → 51 files
- Demo typecheck clean — `bun run typecheck` → exit 0
- Demo production build — `bunx vite build` → built in 3.27s (index.html + worker
  chunk + mdi fonts + bundle)
- `git add -n` confirms only source staged (no dist/node_modules/wasm/samples.gen)

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| TS2459 RunResult not exported from ./api | it lives in types.ts | import RunResult from ./types | pkg typecheck exit 0 |
| demo: Cannot find module @gad-lang/ide-vuetify | package dist not built; vue-tsc ignores vite alias | build the package first (dist/index.d.ts) | demo typecheck exit 0 |
| Rollup can't resolve @lezer/highlight from @codemirror/theme-one-dark | bun isolated node_modules hides transitive @lezer | add @lezer/common+highlight+lr as demo direct deps | demo `vite build` succeeds |

## Current State
`web/ide-vuetify` is a reusable Vuetify 3.8 IDE: <GadIde> (explorer with
create/delete + optional reset, CodeMirror editor with breakpoint gutter + debug
line highlight, Run/Format/Doc/Debug panels with call stack, locals, output and
evaluate) driven by an injectable `IdeApi` (same contract as ide-react; httpIdeApi
for a `gad ide` server). CodeMirror 6 is mounted framework-agnostically via
GadEditorView and wrapped in GadEditor.vue. The package builds to dist (ES lib +
d.ts + extracted style.css, exported as ./style.css). `web/ide-vuetify/demo` is a
server-less app (like React's webide.html): localIdeApi implements IdeApi over
WebFS (read-only bundled samples + LocalStorage overlay, reset via onReset) and
the Gad WASM module in a Web Worker; main.ts bootstraps Vuetify (mdi font
iconset). Both added to the bun workspace. Verified: package + demo typecheck
clean, package build, demo production build all succeed. Not committed yet.
Live-browser run not verified (needs a browser); every other check passes.
