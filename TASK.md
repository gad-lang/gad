# TASK: Mixin system (issue #9)

> Created: 2026-08-27 | Updated: 2026-08-27

## Goal
Implement a complete mixin system for Gad: `mixin Name { … }` parses like a class
(parents, fields, props, methods) plus an optional `this { … }` interface block,
lowers to the `Mixin(…)` builtin, and classes pull mixins in with `use A, B`.
Mixin members become the class's own; mixin fields init first (parents-first).

## Plan
- [x] Tokens: `mixin` keyword (`use` is a contextual ident, not reserved)
- [x] AST: `ClassExpr.Mixin`, `.Use []Expr`, `.This *InterfaceExpr`
- [x] Parser: `ParseMixin{Expr,Stmt}`, `this {}`, `use A, B` (ident/selector/index, comma or newline sep)
- [x] Compiler lowering: `mixinCallExpr` → `Mixin("N", (mx,define)=>{ [interface N$this{…}] define(;…) })`; `use` → `mixins=[…]`
- [x] Runtime `*Mixin` (objects_mixin.go): TMixin, `Mixin(…)` builtin, internal class, raw define capture
- [x] Class: `mixins`/`mixinsFlat`, `mixins=` Define arg, `useMixins` merge (dedup first-wins), `@mixins`
- [x] Init: mixin `initFields` run first (flattened, parents-first)
- [x] Attributes: mixin `@fields`/`@props`/`@methods`/`@parents`/`@module`/`@name`/`@interface`; class `@mixins`
- [x] `gad.Mixin` type registered (module_gad.go)
- [x] Formatting member order (parents, use, 4 field groups by name, this, props, methods, funcs) for class/interface/mixin
- [x] `use A, B` greedy line-wrap with indentation at column limit
- [x] `use` accepts ident/selector/index expressions
- [x] `@interface` renders faithfully (`f int; get p; run()`); Interface.String() now delegates to member ToString()
- [x] Go tests: VM/compiler-level mixin semantics (mixin_test.go, 12 tests)
- [x] Documented sample under samples/ (samples/class/mixins_test.gad, 7 tests + doc comments)
- [x] `make verify` (green: regen+lint+race-tests+CLI+WASM)

## Log
### 2026-08-27
- `use` reverted from keyword to contextual ident; `use` accepts ident/selector/index — `go test ./parser/ -run 'TestParseWith|TestParseClass'` → ok
- Full runtime+compiler build — `go build ./...` → exit 0
- End-to-end `.__tmp/m1.gad` (use merges field/prop/method; field override; @mixins/@fields): ran, count=3→(inc)4, doubled=6, mixin fields first ✓
- End-to-end `.__tmp/m2.gad` (this{}-typed this→plus2=7; mixin parents b=10; `use SubA,A` dedup no error; C.@mixins=2; SubA.@parents=2; A.@interface={plus2()}; `A :: gad.Mixin`=true; anon mixin x=7): all printed as expected ✓
- Full suite — `go test ./...` → 29 ok, 0 FAIL
- Formatting member-order added (sortedClassFields/Members, sortedInterface*); `use` greedy-wrap verified (maxcol 60 → indented continuation under `use`)
- `Interface.String()` now delegates to member ToString(): `@interface` renders `f int; get p; run()`; only the 4 mixin tests needed updating, rest of suite green — so no existing golden/doc render depended on the old type-less form
- Added `ClassProperty.accessors()` (getter=1 param `(this)`, setter=2 `(this, value)`) so `@interface` mirrors get/set/prop
- mixin_test.go (12 tests): use-merge, init-order (own + parents-first), this-interface typing, parents, dedup first-wins, anonymous, per-instance initFields, `:: gad.Mixin`, @interface instance+render, reflection attrs — `go test . -run TestMixin` → ok
- Full suite after all formatting+interface changes — `go test ./...` → 29 ok
- samples/class mixin sample — `gad test samples/class` → 11 passed, 0 failed
- `make verify` → "verify OK" (regen clean after `git add doc/`, lint, race tests, CLI + WASM build all pass)

## Follow-up work (same session, after mixin core)
- **samples/mixins.gad** created — detailed top-level narrative sample (runs, `return w.count`), added to site.go `langOrder` + samples/README table.
- **Denumbered all top-level samples** — `git mv NN_name.ext → name.ext` (36 files); `09_template.gad→template.gad`, `23_template.gadt→template_example.gadt` (avoid `template.md` collision). Fixed every cross-ref in samples/**, Go doc sources (builtins_doc.go, module_gad_doc.go), site.go `langOrder`, samples/README, doc/embedding.md, doc/getting-started.md, ERROS.md, handoff.md, and the 4 test files (markdown_test/nav_test/doc_filedoc/doc_template_dialect + tmtest). `git rm` old numbered doc/samples/*.md, regenerated as name.md.
- **`///` → `/** **/`** — converted single-line doc-comment runs to block form in 27 hand-written samples (16 top-level + 11 subdir); left doc_comments.gad (illustrative) and generated stubs untouched.
- `make verify` → OK (after `git add doc/`). `samples.gen.ts` regenerated via bun (gitignored).

## Committed + plugins (final)
- main (unpushed, 4 new commits): 117b6c8 feat(mixin) [+ sample denumber + `///`→block], f17ac9c bump vscode/intellij, 795f767 bump codemirror/prism, d014630 tmtest mixin scopes.
- Grammar is single-source (generated from Go vocabulary via `update-vscode-plugin -print`); `mixin` keyword + `Mixin` builtin land automatically. Published + pushed: gad-textmate d507150; plugin submodules bumped+pushed: vscode-gad a432961, intellij-gad d13d5b6, codemirror-gad 9a08b44, prism-gad fec1db7.
- IntelliJ completion needs no code change — it is driven by `gad complete`, which already surfaces mixin/Mixin from the compiled binary.
- prism-gad grammar edited by hand (its generator reports "array not found" — pre-existing); codemirror via its generator.
- `make grammar-test` (23 tests) + `make verify` green.

## Unverified / Pending
- main repo commits are LOCAL (user said "comita", not push); origin/main is 6 behind (2 pre-existing + 4 this session). Offer to push.

## Current State
Mixin system is implemented and verified: `mixin` parses, lowers to the `Mixin(…)`
builtin, produces a `*Mixin` whose members merge into a class via `use A, B`
(dedup first-wins across the flattened lineage; dedup is a using-class concern,
the mixin accepts everything). Mixin fields initialise first, parents-first.
`this { … }` types the `this` of the mixin's props/methods via a local `Name$this`
interface. Attributes work: class `@mixins`, mixin `@fields`/`@props`/`@methods`/
`@parents`/`@module`/`@name`/`@interface` (a cached Interface instance named
`Name$interface` mirroring declared members with fidelity). Formatting orders
class/interface/mixin members canonically and wraps long `use` lists. 12 Go tests
+ parser round-trip tests + full 29-package suite all green. Remaining: a
documented sample under samples/, then `make verify`.

Note: interface satisfaction of a method-only interface via `::` returns false
even for a plain class with the method — a PRE-EXISTING `::` limitation unrelated
to mixins (confirmed with a non-mixin class). Not in scope; @interface itself
returns a correct Interface value.
