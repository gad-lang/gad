# Handoff: ia_todo.md language features — ALL DONE

Every item in `ia_todo.md` is implemented, each with parser + compiler + VM
tests (plus encoder tests where relevant), `make test` green, and committed to
`main`. Commits (newest first):

| Commit    | Feature |
|-----------|---------|
| `69da92f` | Coder (gofmt-like `WriteCode`) for match/comprehension nodes |
| `7959e02` | regexp `replace` method + `\|` (replace) operator |
| `dd44c16` | richer dict comprehension keys (multi-key, `[expr]`, `_` accumulator) |
| `cd5c34f` | `/regex/` literal compiled to a `Regexp` constant (+ encoder/decoder) |
| `ebf9912` | `deferb` / `deferb_ok` / `deferb_err` (block-scoped defer) |
| `9077862` | MixedParams destructuring `(a,b,**r; c,p:d,r=2,**nr) := x` |
| `3b4232e` | `defer` / `defer_ok` / `defer_err` (function defer, `$ret`/`$err`) |
| `236a645` | array & dict comprehensions (Python-like) |
| `7539b61` | `match` expression + statement (PHP8-like) |
| `700def9` | array & dict spread/merge literals `[*a]`, `{*b}` |
| `5d519fe` | dict destructuring `(;a, _b:b, r=2, **other) = dict` |
| `29d3389` | `or` error-fallback expression |

## Key design choices

- **Desugaring over new opcodes.** `or`, `defer`/`deferb`, comprehensions,
  destructuring and the regex-replace `|` are compiled by building AST and
  reusing existing constructs/opcodes. Two helpers in `compiler_defer.go`:
  `parseGadSnippet` (parse a template) + template-splicing.
- **defer/deferb: zero VM changes.** A defer-using function moves its body into a
  thunk whose return value is captured into `$ret`; handlers are closures
  capturing `$ret`/`$err` registered on a runtime list and drained in a
  try/catch/finally epilogue. `deferb` wraps the enclosing block similarly (no
  `$ret`). Works because captured locals are transparently shared via
  `OpGetLocalPtr`/`*ObjectPtr`.
  - FIXED VM bug (`3b1793d`): a nested `try` inside a `finally` used to clobber a
    pending return value during return-through-finally — the spent nested
    handler shadowed the outer handler's `returnTo`. `xOpThrow` (system finalize)
    now pops a finalized handler when there is no error/return. With that fixed,
    `deferb` (`0400d50`) runs handlers inside a per-handler try/catch again, so a
    throw inside a `deferb` handler IS captured into `$err`. `$ret` is shadowed
    as a block-local nil in the deferb wrapper (a block has no return value, and
    reaching the enclosing function's `$ret` corrupted the stack).
- **`/regex/` literal.** Scanner treats `/` as a regex only in operand position
  (`!InsertSemi`) AND when a closing `/` exists on the same line
  (`Reader.LooksLikeRegex`), so division is unaffected. Compiled to a `*Regexp`
  constant at compile time (invalid → compile error); encoder/decoder added
  (`typeRegexp`, POSIX-ness not preserved across encode). `return /re/` needs
  parens since `return` is a value-position keyword.
- **dict comprehension** keys are now static by default (`name:`) and computed
  with `[expr]:`; `_` is the dict being built (`_.z ?? 20`).
- **MixedParams** value construction already existed (`MultiParenExpr` →
  `MixedParams`); added `**rest` parsing in the positional section + the
  destructure path (positional index/slice + dict-destructure of `dict(mp.named)`).
- New keywords in `token/token.go`: `match`, `defer*`, `deferb*`; new literal
  token `Regex`. Selectors accept keyword names so `re.match(...)` still works.

## Where things live
- `compiler_defer.go` — defer/deferb desugar, regex literal compile, snippet parser.
- `compiler_nodes.go` — dict/MixedParams destructure, comprehensions, match,
  spread literals, block-stmt deferb hook.
- `parser/node/expr.go`, `stmt.go`, `literal.go` — new AST nodes + Coder impls.
- `parser/scanner_scan.go`, `parser/source/reader.go` — regex scanning.
- `objects_regexp.go` — regexp `replace`/`|`.
- `encoder/encoder_v1*.go`, `decoder_v1_funcs.go` — Regexp constant enc/dec.
- Tests: `parser/parser_test.go`, `compiler_test.go`, `new_test.go`,
  `vm_err_test.go`, `encoder/encoder_v1_test.go`.

## Not mine
`vm_loop.go` has an uncommitted `OpExtendModule` nil-module fix made by
tooling/user during the session — left untouched.

## Post-feature work (committed)
- `OpFinalizer` return-through-finally bug fixed; `deferb` improved ($ret
  shadowed, handler throws captured into $err); `OpExtendModule` nil guard.
- godoc completed for new nodes; README updated + examples verified; CLAUDE.md /
  ia_todo.md / handoff.md committed.

## Doc tooling (committed; metadata path ready)
- `cmd/gaddoc` stays a **markdown** generator. Headers converted to new syntax
  `Name(params) <ret>` with gad types + `*args` variadic; operator-overload docs
  left as `->`. Fixed `getModuleItem` (built module via NewModule instead of
  MustGetData(nil), which had broken `time` doc gen). Regenerated `docs/stdlib-*`.
- gaddoc now PREFERS live function metadata: for a documented `*gad.Function`
  OR `*gad.BuiltinFunction` with a `Header`, the signature is generated from
  `FuncName + Header.String()`; a non-empty `Usage` is used as the description;
  otherwise it falls back to the gad:doc comment. `getModuleFunc` returns a
  shared `funcMeta`.
- ABANDONED "migrate all stdlib funcs to Header/Usage": doc-only (safe, no
  runtime validation — `Function.Call` is just `f.Value(call)`), but ~100 funcs
  and the builder API can't represent named-param defaults (`emph="..."`),
  optional `[, n int]` markers, or `[str]` element types — would degrade those
  signatures. Would need a `NamedParamBuilder.Default(...)` core addition.

## CURRENT TASK (ia_todo.md) — IN PROGRESS
Create gad:doc strings for: `vm.ObjectConverters.RegisterToObject`,
`gad.AddMethodOverride`/`gad.AddMethod`, and `module.Data` entries.
DONE so far: extended `cmd/gaddoc` to recognize `## Converters` and
`## Method Overrides` gad:doc sections (new docgroup buffers convs/methods,
recognized in processBlocks, emitted after Functions). TODO: write the gad:doc
section strings near the RegisterToObject converters + AddMethod(Override) calls
(time module: registerConverters at ~line 656; int override timeToInt ~line 702),
regenerate docs, verify, commit.

## (history) Doc tooling earlier notes
`cmd/gaddoc` stays a **markdown** generator (YAML/HTML-UI ideas dropped per
user). Remaining work: support the new function-header syntax.
- `cmd/gaddoc/main.go`: `reFuncAnnot` matches `Name(params) -> ret`; update it to
  the new `Name(params) <ret>` syntax (see `node.FuncExpr.prefix()` /
  `FuncType.String()`). The `(\w+)\(.*\)` part already accepts named-param `;`.
- Convert all `// gad:doc` function headers in `stdlib/*` from `-> ret` to
  `<ret>` with gad type names (`string`→`str`) and the named-param `;` variation.
- Regenerate `docs/stdlib-*.md` via `make generate-docs`; verify gaddoc has no
  errors (it validates each function name exists in the module).

## Verify
`make test` (lint + cover + -race + fib smoke). Per feature:
`go test . -run 'TestVMOrExpr|TestVMMatchExpr|TestVMDeferStmt|TestVMDeferbStmt|TestVMComprehension|TestVMRegexLit|TestVMRegexpReplace|TestVMDictDestructure|TestVMMixedParamsDestructure|TestVMSpreadLiterals'`
