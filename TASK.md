# TASK: Declaration organizer for the Gad formatter
> Created: 2026-08-19 | Updated: 2026-08-19

## Goal
Teach `gad fmt` to organize declaration groups (`var` / `const` / `global` /
`param`, and short `:=`) — reorder/group their items, merge adjacent declaration
statements, and lay out multiline values — **without changing program semantics**
(name resolution / shadowing must be preserved).

## Spec (locked)

### Ordering within a group (rank ascending; within a rank, sort by identifier)
1. no value, untyped
2. no value, typed (**`param` only** — `var/const/global` have no types)
3. value = **plain** (literal or bare identifier)
4. value = **expression** (`a+1`, calls, operators, cond, index, selector)
5. value = **ComputedExpr** `(= …)`
6. value = **closure** (`=>`)
7. value = **func** (`func`)

### Scope preservation (correctness > ordering)
A value that references a name declared in the same group keeps its order relative
to that declaration (so `var (x, b = a, c = () => a, a = 2, d = a+2)` still has
`b`/`c` see the OUTER `a` and `d` see the group `a`). Implemented with a **safe
over-approximation**: scan each value's text for group-declared names (no AST
walker exists) — never breaks resolution, at worst reorders less.

### Statement merging
Adjacent declaration statements of the same kind merge into one paren group
(`:=` ≡ `var`; `const`; `global`; `param` — each only with its own kind). A
non-declaration statement **or a blank line** between them breaks the merge.

### Other rules
- A single item renders in short form (`a := 1`).
- `param`: positional params are **fixed** (never moved); only named params sort.
- Destructuring (`{a,b} = z`) is an item, sorted by its **first name**; it merges.
- A value whose **formatted code is multiline** forces **every item onto its own
  line** (one per line); the value stays `name = <multiline value>`, indented.
- `const` with `iota` → group left **intact**.
- Docs / comments travel with their item.

## Plan (each stage tested by RUNNING original vs formatted)
- [x] Stage 1: reorder within a `var`/`const` paren group (covers a/b/c/d).
      (`global` uses ParamSpec, deferred to Stage 4 with `param`.)
- [x] Stage 2: collapse a lone `var name = value` to `name := value`.
- [x] Stage 3: merge adjacent declaration statements (`:=`/var/const/global/param;
      blank line or floating comment breaks the run; lead docs travel).
- [ ] Stage 4: `param` (named only) + destructuring as an item + `global`.
- [x] Stage 5: multiline value forces item-per-line (existing measure logic).

## Log
### 2026-08-19
- Spec locked and saved.
- Built `parser/node/walk.go`: reflection-based `Walk(root, f)` AST walker +
  `IdentNames(n)` (over-approx of referenced names, for the scope check). Keyed on
  **`ast.Node`** (Pos/End/String), so custom node implementations (the gadx AST)
  are traversed too. Chose a package function over a `Node.Walk` interface method:
  there is no shared base struct, so a method would mean boilerplate on 100+ node
  types, whereas reflection covers all of them (incl. custom) with none — same as
  go/ast's `ast.Inspect`. `go test ./parser/node -run TestWalk` → PASS (4 tests);
  `go test ./...` → no FAIL; `go test ./gadx/...` → ok.

### Stage 1 done
- `parser/node/decl_order.go`: `GenDecl.orderedSpecs()` reorders a `var`/`const`
  paren group of single-ident, pattern-free specs by (rank, name), where rank =
  1 valueless / 3 plain / 4 expression / 5 ComputedExpr / 6 closure / 7 func.
  Constraints from `IdentNames` keep any value that references a group-declared
  name in its original position relative to that declaration → a greedy
  topological sort that can never change resolution. Eligibility is conservative
  (skips patterns, multi-ident specs, `iota`, `global`); ineligible groups are
  emitted unchanged. Wired via `specsForWrite()` in `GenDecl.WriteCode`.
- Tests (`cmd/gad/decl_order_test.go`): grouping order, idempotency, and
  **resolution preserved by RUNNING** original vs formatted (a/b/c/d → `[1,1,4]`
  both ways). `go test ./...` → no FAIL.

## Current State
Stages 0 (walker) and 1 (within-group reorder) done and green. Next: Stage 2
(collapse single-item group to short form), then merging, param/destructuring,
multiline-forces-item-per-line.
