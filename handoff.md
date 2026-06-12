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

## Verify
`make test` (lint + cover + -race + fib smoke). Per feature:
`go test . -run 'TestVMOrExpr|TestVMMatchExpr|TestVMDeferStmt|TestVMDeferbStmt|TestVMComprehension|TestVMRegexLit|TestVMRegexpReplace|TestVMDictDestructure|TestVMMixedParamsDestructure|TestVMSpreadLiterals'`
