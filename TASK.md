# TASK: WASM worker backend + embeddable web-only IDE (#50/#51)

> Created: 2026-08-03 | Updated: 2026-08-03 16:20

## Goal
Move the app's in-browser Gad execution off the UI thread into a Web Worker
(run/format/diagnose + a stepping debugger) and ship a fully server-less,
embeddable IDE page: a read-only sample tree with a LocalStorage overlay for
user edits/new files/deletions (resettable), reusing the CodeMirror editor and
driving Run/Doc/Debug entirely through the WASM module in the worker.

## Plan
- [x] client.ts: align run/format/diagnose types to backend types; add docData()
- [x] backends/debug.ts: DebugBackend interface + serverDebugBackend (+debugBackend alias)
- [x] backends/wasmWorker.ts: sharedClient + wasmWorkerBackend (GadBackend) + wasmDebugBackend
- [x] App.tsx: switch "wasm" backend to the worker; Debug tab picks debug backend by selector
- [x] Debug.tsx: accept a DebugBackend prop; hint conditional on needsServer
- [x] webide/tree.ts: pure buildTree + langFor helpers
- [x] webide/WebIde.tsx: tree sidebar (create/delete/reset) + Editor + Run/Doc/Debug panels
- [x] webide.tsx + webide.html entries; vite multi-page input
- [x] styles.css: WebIde layout (responsive)
- [x] Remove orphaned backends/wasm.ts (UI-thread backend, superseded)
- [x] Add @lezer/common + @lezer/lr direct deps so the production build resolves

## Log
### 2026-08-03
- App typecheck clean after wiring — `bun run typecheck` → exit 0
- buildTree sanity — bun script printed correct folders-first nesting/sort
- Production build (both entries) — `bunx vite build` → built in 3.17s, emitted
  dist/index.html + dist/webide.html + separate webide chunk (91.65 kB) + worker chunk
- Rebuilt current wasm (adds gadDocData) — `bun run wasm` → wrote public/gad.wasm
- Doc-template Go tests still green after user's pug-style html.giom rewrite +
  brace fix — `go test ./cmd/gad/ -run 'TestRenderDocTemplate|TestDocCommandUsesTemplates'`
  → PASS (MD, HTML, e2e md+html)

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| TS2724 wasmDebugBackend not in ./backends/debug | imported from wrong module | import from ./backends/wasmWorker | typecheck exit 0 |
| Rollup can't resolve @lezer/common from @codemirror/commands | bun isolated node_modules; not a direct dep | add @lezer/common + @lezer/lr as direct deps; bun install | `bunx vite build` built in 3.17s |
| html.giom `unresolved reference "{s.line}"` | pug `[attr=expr]` value is an expression, not a `{…}` interpolation | `data-line=s.line` (no braces) | doc-template tests PASS |

## Current State
The app's in-browser backend now runs in a Web Worker: backends/wasmWorker.ts hosts
one shared WasmClient and exposes wasmWorkerBackend (run/format/diagnose) and
wasmDebugBackend (start/command/evaluate/stop) implementing a new DebugBackend
interface (serverDebugBackend is the HTTP twin). App.tsx's backend selector now
picks the matching run and debug backends; the Debug tab works without a Go server
when "WebAssembly" is selected. A new standalone entry (webide.html → webide.tsx →
WebIde) is a server-less embeddable IDE: WebFS gives a read-only sample tree with a
LocalStorage overlay (edit samples, create/delete files & folders, reset), the
CodeMirror editor is reused, and Run/Doc/Debug all go through the worker. vite builds
both pages (index + webide) as separate bundles. Typecheck clean; production build
succeeds; the doc-template Go tests pass against the user's pug-style html.giom.
Not yet committed. Not verified in a live browser (WASM execution needs a browser);
all other checks (typecheck, production build, Go tests) pass.
