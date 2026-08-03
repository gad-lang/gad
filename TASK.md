# TASK: Server-less embeddable IDE via @gad-lang/ide-react + localIdeApi

> Created: 2026-08-03 | Updated: 2026-08-03 19:10

## Goal
Ship the standalone, server-less IDE page (webide.html) by reusing the extracted
`@gad-lang/ide-react` <Ide> component, driven by an in-browser `localIdeApi`
(WebFS sample tree + LocalStorage overlay, Gad WASM in a Web Worker). No Go
server. Supersedes the earlier custom WebIde.tsx/tree.ts (now removed).

## Plan
- [x] webide.tsx renders <Ide api={localIdeApi}> from @gad-lang/ide-react (user)
- [x] backends/localIde.ts: full IdeApi impl over WebFS + WASM worker (user)
- [x] client.ts: run(sourceType)/diagnose(sourceType)/docComments/evalExpr/transpile (user)
- [x] bridge.go: EvalExpr (+ Transpile/DocComments) for the IDE panels (user)
- [x] wasm main.go: gadEval/gadTranspile/gadDocComments/gadDocData exports (user)
- [x] backends/debug.ts: re-export debug wire types from @gad-lang/ide-react (user)
- [x] vite: alias + dedupe for @gad-lang/ide-react; two-page build (user)
- [x] Remove custom WebIde.tsx + webide/tree.ts (superseded)
- [x] worker.ts: complete GadGlobals with the 4 new fn declarations (me)
- [x] gadbridge: add TestEvalExpr (was untested) (me)

## Log
### 2026-08-03
- App typecheck clean — `bun run typecheck` → exit 0
- Go build/test clean — `go build ./...` exit 0; `go test ./web/... ./cmd/gad/` all ok
- WASM builds with new exports — `GOOS=js GOARCH=wasm go build ./web/wasm` → 20591567 bytes
- Production build both entries — `bunx vite build` → built in 3.18s; dist/index.html +
  dist/webide.html + separate webide chunk (85.92 kB) + worker chunk
- All WASM exports the client calls exist in main.go (gadEval/gadTranspile/
  gadDocComments/gadDocData/gadDoc/gadDocData/gadRun/gadDiagnose/gadFormat/gadDebug*)
- Added EvalExpr coverage — `go test ./web/gadbridge/ -run TestEvalExpr` → PASS
- gofmt clean on touched Go files

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| TestEvalExpr repr expected `"hi"` | Gad repr renders the type-annotated `‹str: "hi"›` | assert Contains(`"hi"`) for repr, exact `hi` for str | TestEvalExpr PASS |

## Current State
The server-less IDE is delivered by reusing `@gad-lang/ide-react`'s <Ide>
component (the same UI `gad ide` serves), injected with `localIdeApi` — a full
in-browser IdeApi over WebFS (read-only samples + LocalStorage overlay) and the
Gad WASM module in a Web Worker (wasm/client.ts + worker.ts). client.ts and the
WASM bridge (bridge.go EvalExpr/Transpile/DocComments + main.go gadEval/
gadTranspile/gadDocComments/gadDocData exports) supply run/diagnose/doc/eval/
transpile/debug; debug wire types are re-exported from ide-react so both sides
share one definition. My earlier custom WebIde.tsx/tree.ts are removed. This
turn I completed the worker's GadGlobals type (the 4 new fns) and added
TestEvalExpr. Verified: Go build/test/vet/gofmt clean, WASM builds, app
typecheck clean, production build of both entries (index + webide) succeeds. Not
committed yet. Live-browser WASM execution not verified (needs a browser); every
other check passes.
