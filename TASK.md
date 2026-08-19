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
- [ ] Stage 1: reorder within a `var`/`const`/`global` paren group (covers a/b/c/d).
- [ ] Stage 2: collapse a single-item group to short form.
- [ ] Stage 3: merge adjacent declaration statements.
- [ ] Stage 4: `param` (named only) + destructuring as an item.
- [ ] Stage 5: multiline value forces item-per-line layout.

## Log
### 2026-08-19
- Spec locked and saved.
- Built `parser/node/walk.go`: reflection-based `Inspect(root, f)` AST walker +
  `IdentNames(n)` (over-approx of referenced names, for the scope check).
  `go test ./parser/node -run TestInspect\|TestIdentNames` → PASS (3 tests);
  `go test ./...` → no FAIL.

## Current State
AST walker done and green (foundation for the scope-safe reorder). Next: Stage 1
(reorder within a var/const/global paren group), using `IdentNames` to build the
resolution-preserving constraints. Prior formatter work in this branch (opt-in
column-aware fmt, comment preservation for array/dict) is merged and green.
