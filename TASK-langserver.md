# TASK: Gad language service (precise navigation + completion)

> Created: 2026-08-20

## Goal
Give the IntelliJ and VS Code plugins **precise** go-to-definition and
auto-completion, driven by a single engine in `gad` (so the logic is not
duplicated per editor). Both plugins call the engine and render results.

## Engine (in gad) — a small language service
Exposed as CLI commands operating on a file + a caret offset:

- `gad def --offset N [file|-]` → the declaration's location (line/col/offset)
  of the identifier at the caret. Scope-aware (blocks, functions, shadowing —
  e.g. the `a/b/c/d` case), using a resolver over the AST (`node.Walk`).
- `gad complete --offset N [file|-]` → a JSON list of candidates, each with a
  label, kind, and **documentation** when available.

### Completion sources (precise)
1. **In-scope identifiers** — variables/params/functions visible at the caret,
   from the AST scope resolver. Doc = the declaration's lead doc comment.
2. **Keywords + builtins** — from the language vocabulary; builtin docs from the
   existing gad doc system.
3. **Imported module members** — resolve the module and list its exports (with
   their docs).
4. **Member / index access** (`x.` , `x[`) incl. nested, **class & class
   instance members** — **runtime introspection**: build a prelude (source up to
   the receiver + `return x`), Eval it with a timeout + recover + sandbox, then
   inspect the real value: Dict → keys, Array → indices/methods, Module →
   exports, class/instance → fields + methods + props, else → the value's type
   methods. Doc from the value/type where present.

### Documentation
Every completion item and the def target carry documentation when it exists:
lead doc comments (`///`, `/?? … ??`) for local declarations; the gad doc system
for builtins and module exports.

## Key decision (assumed unless told otherwise)
Member/index completion **evaluates the user's code up to the caret** — the only
way to know a dynamic value's members. Mitigated with an Eval **timeout +
recover + sandbox**. (Static-only fallback is possible but cannot know dict
keys / array shape / module values.)

## Plan (each stage tested in gad)
- [x] Stage 1: AST scope resolver + `gad def` (scope-aware; shadowing + params tested).
- [ ] Stage 2: `gad complete` — in-scope idents + keywords + builtins, with docs.
- [x] Stage 3: imported module members (with docs) — via runtime introspection.
- [x] Stage 4: member/index/nested/class/instance via runtime introspection (with docs).
- [x] Stage 5: wire IntelliJ (CompletionContributor + `gad def` GotoDeclaration) and
      VS Code (CompletionItemProvider + DefinitionProvider) to the engine.

## Foundation already in place
`gad ast --json` (astio), `node.Walk`/`IdentNames`, the gad doc system, the VM
`Eval` API.
