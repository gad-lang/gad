# Handoff: TASK.md IDE epic + Language tasks

## ACTIVE WORK (2026-06-30): IDE epic wrap-up

Continued from previous session. All items below committed; tsc --noEmit
clean for web/app, codemirror-gad, prism-gad and vscode-gad packages.
TASK.md updated in-repo after each commit.

DONE this session (commits newest-last):
- **Tree navigator hover inspect + goto-source** `78e942e` — debugDecorations.ts:
  getInspect callback; hover tooltip gets ⊕ inspect button → onInspectVar →
  Ide.setInspectTarget. TreeNavigator/InspectDialog: extractGadFile() parses
  .gad paths from repr values; ↗ goto-source button; onGotoSource prop chain →
  openFile.
- **Doc-comment sub-language highlighting** `b9e47a0` — codemirror-gad: GadState
  gains docCodeFence; ``` fenced blocks inside doc comments get full Gad
  tokenization; `>>> ` result lines → "docResult" token (t.special(t.comment)).
  prism-gad: inside patterns for doc-code-fence + doc-result. docMarkdown.ts:
  `>>> ` lines wrapped in <span class="doc-result"> (italic/muted). styles.css.
- **VS Code format-on-save** `66725bb` — extension.ts: registerDocumentFormatting
  EditProvider → `gad fmt -` (stdin/stdout); gad.format.useConfig → --no-config.
- **Template string interpolation highlighting** `fd79720` — codemirror-gad:
  GadState gains tmplClose/tmplDepth; tokenTmplText + tokenTmplCode handle the
  two sub-regions of #"…{expr}…". tokenTmplCode delegates to tokenCodeLine for
  full Gad highlighting (autocomplete/hover inside {expr}). prism-gad: template-
  string pattern with interpolation inside.

PRIOR SESSION (same day — IDE epic middle):
- **GadInput.tsx** `3c85e8d`, **Multi-format editors** `cb84475`,
  **Run/Debug tabs** `608968c`, **FetchFromWebDialog** `52bb55a`,
  **Breakpoints double-click** `e866839`, **gaddoc TOC** `1ebbc6f`,
  **Builtin hover tooltips** `893a9ac`.

REMAINING in TASK.md:
- `[ ] create plugin like vscode to JetBrains` — large scope, not started

Verify: `go test ./...` + `go build ./...` + (in web/app) `pnpm exec tsc --noEmit`.

---

## PRIOR SESSION (2026-06-30): doc-comment markers + `in`-string + `class` + `enum` + heredoc/keyValueArray samples

Worked the `TASK.md` **Language** section to completion. All committed; `go build
./...`, `go vet ./...`, `go test ./...` green; all 22 `samples/*.gad` run clean.
Only `TASK.md` is uncommitted (user-maintained). The user maintains `TASK.md`
live; do NOT clobber their edits — re-read before editing.

DONE this session (commits newest-last):
- **doc-comment markers** `6c9e1a1`, `e51562f` — SINGLE `/?`→`///` (`////` is a
  normal comment), BLOCK `/??…??`→`/**…**/`, ROOT_BLOCK `/???…???`→`/***…***/`.
  Scanner, parser attach, formatter, gadbridge, doc generator, doctest, all four
  tokenizers, docs, samples.
- **`Class(...; fields=...)` fix** `5b1f12e` — `GetDo` (strict) → `GetDoCheck(
  false, …)` so the leftover `define`/`fields` args are not rejected. Class +
  ClassInstance godoc `2c7eae1`.
- **`in` string membership** `282ecef`, `475ca97` — `'e' in "hello"` /
  `"ell" in "hello"` via `Str.BinOpIn` / `RawStr.BinOpIn`; `ain` string cases.
- **`class` syntax** `c587c1e`, `7067dcb`, `267fd13`, `5f4aa85`, `7964f84` —
  expr + stmt + export forms. New `Class` keyword (extends/props/methods/new are
  contextual idents). `ClassExpr`/`ClassStmt` AST (`parser/node/class.go`),
  parser (`parser/class.go`), compiler lowering (`compiler_class.go`) to
  `Class(name; define=(Type,define) => define(; extends, fields, properties,
  methods, new))`. Methods get typed `this Type` (overload dispatch); property
  accessors + constructors get UNTYPED `this` (a typed `this` resolves `Type` to
  the instance at invocation time — see `7964f84` ParamTypes guard). Formatter
  (`ClassExpr.WriteCode`, expanded layout) + class-body doc-comment claiming
  (`claimClassBodyDocs` in `parser/node/coder.go`). samples/19, doc/classes.md.
- **`enum` syntax** `5683ec9`, `5789943`, `29fdc21` — expr + stmt + export.
  Builds a compile-time `Enum` CONSTANT (not runtime calls): `compiler_enum.go`
  `buildEnum` + `evalEnumExpr` (compile-time integer expr evaluator). Value
  rules: default = prev magnitude +1 (or 1); `bit`→`1<<n`; `+`/`-` make a field
  signed & set a running sign; type (int/uint) propagates left-to-right; `_`
  advances value w/o being added; explicit values ref earlier fields. Runtime
  `Enum`/`EnumValue` are the user's (`objects_enum.go`); I added
  `EnumValue.IndexGet` (`.name/.value/.index/.enum`) + godoc. Encoder/decoder
  (`encoder/encoder_v1_enum.go`, registered in `encoder_v1.go`). samples/20,
  doc/enums.md.
  - SPEC CONTRADICTIONS (documented in TASK.md enum note): 3 examples conflict;
    `bit _` was incomplete in the spec — implemented a consistent model.
- **block-doc grouping fix** (in `29fdc21`) — `consumeCommentGroup` now ends the
  group after a fenced block doc (`/**…**/`, `/***…***/`) so a following `///`
  lead doc is not absorbed; fixes root-block-then-doc round-tripping (affected
  the class sample too).
- **heredoc samples + scanner fix** `550745b`, `a01abce` — samples/21_heredocs
  (str/raw/template heredocs, single+multi-line) + expanded
  doc/strings-bytes-regex.md. FIX: backslash broke single-line raw heredocs;
  `Reader.ReadAt` (`parser/source/reader.go`) no longer treats `\` as an escape
  (raw = verbatim). Tests in TestScanner/TestVMRawHeredoc.
- **keyValueArray samples + docs** `0c51b90` — samples/22_key_value_array +
  doc/values-and-types.md (flags, func/closure values, typed keys, dup keys,
  flag/values/delete, indexing+mutation, spread, dict()).

REMAINING in `TASK.md`: the **IDE epic** (top section — React `web/app` + Go
`web/ide` backend) and a new "doc generator: group consts/classes/enums/
functions/methods/properties into sections + table of contents" item. The IDE
items are mostly UI (typecheck only, need browser verification).

NOTE on adding a keyword: append to the END of the `token` keyword group
(`token/token.go`) so existing token values do not shift; add to both the
`tokens` and `tokenNames` maps. Hook the operand dispatch (`ParseOperand`), the
`exprStart` token set, the statement dispatch (`DefaultParseStmt`), and
`ParseExportStmt`. Compiler dispatch is a type switch in `compiler.go`.

---

## ACTIVE WORK (2026-06-24): operator op_api trilogy + `with` + plugin sync + IDE epic (in progress)

Task source is now `TASK.md` (renamed from todo.md). This session worked the
operator-API refactor, the `with` statement, the editor-plugin sync commands,
and started the large IDE epic. All committed; `go test ./...` green.

DONE this session (commits newest-last):
- `f7fecf4` **unary op_api** — `mkoptypes` generates `ObjectWith{Op}UnaryOperator`
  + `unOpObject` in `op_api.go`; `VM.xOpUnary` dispatches through a new
  `gad.unOp(op, operand)` builtin; per-type logic moved to `UnOp{Op}` methods
  (`op_unary.go`). `!` is universal (truthiness fallback); `Flag.UnOpNot` keeps
  it a Flag. Temporal `++`/`--` step the **least non-zero unit**: date→day,
  `08:05:00`→minute, `08:05:30`→second, `08:00:00`→hour, midnight→day
  (`op_unary_time.go`, `leastTimeStep`).
- `e5b5ab4` **self-assign op_api** — `ObjectWith{Op}SelfAssignOperator` +
  `selfAssignOpObject`; `gad.selfAssignOp` dispatches through it, falls back to
  the binary op. Removed the old `SelfAssignOperatorHandler` interface; `Array`
  now has `SelfAssignOpAdd` (`+=`) / `SelfAssignOpInc` (`++=`).
- `87ee3ab` **`ain`** array-membership operator (`A ain B` = every value of A in
  B). New `ain` keyword (appended to keyword group, no token-value shift),
  comparison precedence. Dispatched on the right operand like `in`; falls back to
  testing each value via `in` (through `gad.binOp`), so it works for any `in`
  container (Go or Gad). `binAinFallback`. Excluded from optimizer folding.
- `8756c17` **`with`** statement + expression. New interfaces `ObjectEnter` /
  `ObjectExit`; `gad.enter` / `gad.exit` builtins dispatch to the Go interfaces
  OR a Gad object's `enter()` / `exit(err)` methods. New `With` keyword + AST
  `WithStmt` / `WithExpr`. Parser: `with R {}`, `with R as f {}`, `with x = R {}`,
  `with x := R {}`, and the expression `with R [as f]: V`. `inHeader` guard stops
  the resource's trailing `{` becoming a func-def. **Compiler desugars to**
  `{ deferb { gad.exit(h,$err) }; gad.enter(h); body }` (stmt) or an IIFE
  (expr) — no new opcode (`compiler_with.go`).
- `96b7a1b` **plugin sync** — `cmd/update-codemirror-plugin` /
  `cmd/update-prism-plugin` (shared `cmd/internal/pluginsync`). Extracts keywords
  /atoms/constants from the token group + builtin func names from `BuiltinsMap`,
  diffs against the plugin TS arrays (`-w` applies, builtins advisory-only),
  prints language commits since the plugin's last update. Added the missing
  `ain`/`with`/`met`/`meti`/`prop` keywords to both plugins.
- `c4c4066` `gad.TranspileOptions()` default `RawStrFuncStart: "raw "` (operator
  form, empty end) — matches the CLI; transpiled raw text is `write(raw "…")`.

**IDE epic** (`web/ide` backend, unit-tested; `web/app` React, tsc-only) — split
into sub-tasks in `TASK.md`. Done so far:
- `b7b1949` hidden-files toggle (`/api/ide/tree?hidden=true` + eye button);
  locals per-row copy button.
- `412c704` explorer F2 / right-click context menu (open/run/format/transpile/
  rename/remove) + recursive-remove confirm dialog; run output split into
  stdout/stderr files + combine flag; `/api/ide/fetch` (URL→workspace file).
- `07eea21` `gadbridge.Transpile(src, mixed, opts)` + `/api/ide/transpile`
  (config-layered `TranspileOptions`); Settings dialog Transpile section.

IN PROGRESS (uncommitted working tree): editor controls slice — added
`undo()`/`redo()` to `EditorHandle` (`web/app/src/Editor.tsx`); next is wiring
reload/undo/redo toolbar buttons + a backend-error dialog. `TASK.md` lists the
remaining IDE sub-tasks (run/debug tabbed dialog, breakpoint condition dialog,
evaluate panel + backend eval endpoint, multi-format editors, doc-comment side
panel, builtin tooltips, template-string autocomplete, "plugin isn't working"
diagnosis). Most are pure UI — typecheck only, need browser verification.

Key files this session: `op_api.go` (generated), `op_unary*.go`,
`builtins_operator_methods.go` (gad.unOp/enter/exit registration),
`compiler_with.go`, `cmd/internal/pluginsync/`, `web/ide/{fs,run,ide}.go`,
`web/gadbridge/bridge.go`, `web/app/src/{Ide,Editor}.tsx`,
`web/app/src/backends/ide.ts`. Verify: `go test ./...` + (in `web/app`,
`nvm use v26.3.0`) `pnpm exec tsc --noEmit`.

--- (older detail below) ---

## ACTIVE WORK (2026-06-16): builtin modules DONE; next = time literals (todo L78)

DONE this batch — `time`/`strings`/`fmt`/`base64` are now **builtin module
namespaces** (usable without `import`); todo.md L77 marked [x]. Commits:
- `e7a836b` base64 + the builtin-module infra (reserved enum group
  `GroupBuiltinModules…`, import-compat re-export).
- `4a82dd3` fmt + compiler optimization: `module.NAME` for a builtin module →
  single `OpGetBuiltin` (qualified builtin); shadowing local/global disables it.
- `8aedc67` strings + per-module `*ModuleSpec` set on each member; optimizer
  `allowedBuiltins[256]` bounds-checked (now >256 builtins).
- `e9ef527` time + camelCase members + global converters. **Naming convention**
  (todo L102): module functions/methods = lowerCamelCase (`time.now`,
  `time.durationString`), CONSTANTS = PascalCase (`time.Hour`, `time.January`,
  `time.RFC3339`, `time.Type`), value types `time`/`Location` (Location is
  UpperCamelCase per user); `strings`/`fmt`/`base64` members KEPT PascalCase for
  now (L94 will lowerCamelCase strings+fmt). The time↔Go converters + the
  `int(time)` override are registered GLOBALLY in `init()` via the `registry`
  package + `BuiltinObjects` (no VM) — `objects_go_time.go` removed.
  `registerBuiltinModule` pins a member `BuiltinObjType.builtinType` so its
  method-table identity is stable; `BuiltinObjType` gained `Module`; `FullName`
  qualifies module types (`time.time`, `time.Location`).
- `0ee4784` docs/samples/godoc: doc/builtins.md "Builtin Modules" section,
  modules.md note, regenerated docs/stdlib-time.md, samples drop `import`
  (+use_base64.gad). gaddoc gained a 3rd-arg module filter (multi-module root);
  Makefile generate-docs runs `gaddoc . <out> <module>` for time/fmt/strings.
- `690cde9` make-test fixes: moved //go:generate path in module_time_funcs.go;
  mkcallable `*Location`→"Location"; ide.go SA4006; Makefile fib smoke uses
  `go run ./cmd/gad`. **`make test` is GREEN.**

Module impl lives in root: `module_base64.go`, `module_fmt.go`+`_scan`,
`module_strings.go`, `module_time*.go`, shared `module_callable.go` (funcP*
helpers). stdlib/{time,strings,fmt,encoding/base64} are thin shims (ModuleInit
+ type aliases) delegating to root `XModule()` constructors. Their tests stay in
stdlib/<mod> (user said "skip moves"). Qualified members registered as builtins
"time.now", "base64.StdEncoding", etc. (sorted → deterministic indices).

NEXT (todo L78, big): time literals + new primitive types. Add scanner/parser
literals: `\d{8}D`→`time.Date` (alias of go uint); `(\d{8})?(_?\d{6})(.frac)?(Z…|NAME)T`
→`time.time`; `\d+(.\d+)?U`→unix `time.time`; go-duration string→`time.Duration`
(alias of go time.Duration). Compile to those types; encoder for Date/time;
constructors with `strToTime`/`strToDate`/`strToDuration`/`strToLocation`
methods; make time/Date/Duration primitive types; samples+docs. (Then L94:
strings+fmt → lowerCamelCase; L95 regexp/match docs; L102 conventions doc.)

--- (older detail below) ---

## ACTIVE WORK (2026-06-15): `gad fmt` + mixed/template mode (todo.md)

STATUS: Tasks 1–3 + transpile DONE + COMMITTED.
- Task 1 (mixed/template formatting + `begin/end`) → `ed73d1d`.
- Task 2 (godoc, `--template`+delimiter flags+config, `{%--`/`--%}` trim markers,
  `.gadt`, per-file `delimiter=[…]` config, doc/templates.md) → `f2fd892`,
  `46cfb8c`. Trim semantics: `-` keeps a boundary newline, `--` strips all.
- Task 3 (build tags `noide`/`nodebug`; `run` already the default) → `fa58a3e`.
  Optional subcommands self-register via init() into `optionalCommands`.
- Transpile (`--transpile` flag, `fmt.transpile` config, per-input_dir
  `transpile`, `.gadt`→sibling `.gad`, WriteStmts separator fix for transpiled
  write() calls, default `RawStrFuncStart="raw "`/`WriteFunc="write"`) → `7a36d9e`.

- NDJSON report + `--to-stdout`/`--boundary` streaming (todo.md L65) → `a2f4787`.
- `CodeStrLit` (`code … end` code-string literal) → committed below. New
  `token.CodeStr`; scanner `scanCodeStr` (block form: closing `end` at the
  opening line's indentation, deeper `end`s are body; inline `code … end`;
  falls back to a `code` identifier when no fence — non-breaking). Body is
  captured verbatim (compiles to `Str`, NOT template-parsed) and dedented to its
  least-indented line. Node `CodeStrLit` in literal.go; compiler emits
  `Str(Value())`. `code` added to codemirror+prism keyword lists (also fixed the
  stale prism do/then/done → begin/end). Tests: `TestScanner_ScanCodeStr`,
  `TestParseCodeStr`, `TestVMCodeStr`. Doc: strings-bytes-regex.md.

PREVIOUS TASK (todo.md ~L65): rework `gad fmt` reporting to PER-FILE NDJSON
(single-line JSON, one obj per line, keys `{input_dir?, file, error?}`; remove
YAML report support, drop `--report-format`). `input_dir` only when the file is
in a dir job; `file` is relative to that dir then. Add `--to-stdout`: stream
formatted results to stdout (no file writes) and, when `--report` is unset,
print the NDJSON report to stdout too. Add `--boundary BOUNDARY`: if unset,
generate a UUID and print `>> BOUNDARY` as the first stdout line. Stream frame
per file:
  -- BOUNDARY #FILE_INDEX [INPUT_DIR] FILE_NAME   (brackets only if in dir job)
  FORMATTED_FILE_RESULT
  -- BOUNDARY #FILE_INDEX
Plan: t.root!="" ⟺ "in input dir" (use t.root as input_dir, t.relPath() as
file). Store boundary + toStdout on fmtOptions; assign a global file index in
run(). Tests TestMarshalReport/TestValidateReportFormat must be replaced.
IDE codemirror finding (todo task 4, started): the DEFAULT `gad ide` serves the
build-free VANILLA UI (`cmd/gad/ideapp/app.js`) which uses a plain `<textarea>`
— there is NO CodeMirror there. CodeMirror (the React `Editor` +
`@gad-lang/codemirror-gad`) only runs in the React UI served via `gad ide
--static web/app/dist` or `-tags prod`. So "codemirror not working on ide"
is largely "the default UI has no codemirror". Fixed the stale keyword list
(`do`/`then`/`done`→`begin`) in web/codemirror-gad/src/keywords.ts. The big
IDE task needs browser-based iteration; remaining sub-features (file-tree
rename/menu, run/debug dialog, breakpoint dialog, evaluate panel, editors for
JSON/YAML/etc., tooltips, undo/redo, …) are best done with the IDE running.

NEXT (todo.md, top-down): big IDE enhancement task (codemirror fix, file-tree
rename + context menu run/format/transpile/remove, run/debug settings dialog,
breakpoint dialog with disabled/condition, builtin tooltips); then `gad fmt`
per-file JSON report + `--to-stdout` boundary stream; IDE "evaluate" panel;
CodeStrLit heredoc token; time+strings as builtin modules; date/time literals;
regexp `~`/`~~` + POSIX `/…/p` doc examples.

--- (superseded detail below) ---
Task 2 (godoc/`--template`/`--` markers/`.gadt`/docs) = IN PROGRESS:
  - godoc + `--template` flag + delimiter flags + `.gad.yaml` `template:` config
    = COMMITTED. `--template` runs config-less templates; delimiters default to
    `{%`/`%}`, overridable by flags or config; CLI flags win.
  - `{%--`/`--%}` markers (`-` keeps a single boundary newline, `--` strips all)
    = IN PROGRESS in the working tree: node side mostly done in
    `parser/node/stmt.go` (flags RemoveLeftAll/RemoveRightAll on MixedTextStmt;
    `trimmed()` helper with isMixedSpace; RemoveAllSpace on CodeBegin/CodeEnd;
    MixedValueStmt RemoveLeftAll/RemoveRightAll + leftMark/rightMark; String +
    WriteCode emit `--`). STILL TODO: the SCANNER (`scanner.go ScanCodeBlock`
    start `{%--`; `scanner_scan.go` end `--%}`) must detect the double dash and
    set token data `"remove-spaces-all"`; `parser/mixed.go` add
    `RemoveAllSpaces(t)`; the 6 parser propagation points (parser.go ~1957/1967
    CodeBegin/End, ~2068/2075 MixedValue, ~2107/2116 MixedText) set the All
    flags; `parser/node/expr.go` MixedTextExpr may need the All flags. `utils`
    import in stmt.go is now UNUSED — remove it. NOTE: redefining `-` to keep a
    newline changes runtime output of templates using `-`; update
    `samples/09_template.gad` to use `--` where full collapse is wanted, verify
    `gad run` output. Then parser+vm tests, docs+samples, README.
  - NEW (todo updated): run `.gadt` files in template mode automatically.
Task 3 (build tags for ide/debug; `run` as default) = pending.
LATER backlog (todo.md, growing): IDE file-tree rename/context-menu + run/debug
  settings dialog + breakpoint dialog + builtin tooltips; `gad fmt` per-file
  JSON report + `--to-stdout` boundary stream; IDE "evaluate" panel; CodeStrLit
  heredoc-style token (`code…end`); make stdlib time+strings builtin modules;
  date/time literal parsing; regexp operator (`~`/`~~`, POSIX `/…/p`) doc
  examples. (Work top-down; commit each task.)

### Task 1 detail — DONE + COMMITTED (`ed73d1d`)
1. **Mixed/template formatting** — DONE:
   - Parser already sets RemoveLeft/RightSpaces on MixedText from `{%-`/`-%}`
     (verified via parse tests); formatting preserves `Lit.Value` verbatim.
   - `WriteStmts` (coder.go) redesigned with sep kinds (newline/space/glue) +
     `inTag`: `{% … %}` tags render INLINE (no reflow); `%}` hugs preceding code
     with one space; template text/value segments are glued.
   - `BlockStmt.WriteCode` (stmt.go): a template body (`isMixedBlock`) renders
     inline as `do <segments> end`, injecting `do` when the opener is implicit.
   - `MixedValueStmt.WriteCode`: pads to `{%= expr %}` (and `{%- = expr -%}`).
   - `normalizeMixedEndTags` (coder.go) regex normalizes `{% end %}` spacing,
     applied in `Code()` (guarded to sources containing `{%`).
   - Block-keyword aliases reduced to `begin … end` (user requests, in order:
     removed `then`, removed `done`, then renamed `do`→`begin`). scanner_scan.go
     maps only `begin`→`{` and `end`→`}`. All tests + doc/control-flow.md use
     `begin … end`. `if a > 0 begin … end` works; `do`/`then`/`done` now error.
     BlockStmt injects `begin` for implicit template openers.
   - Test: `parser_test.go TestFormatMixedMode` (delimiters set to `{%`/`%}` —
     note `test.New` defaults MixedDelimiter to `‹ ›`). Fixture
     `samples/09_template.gad`: `gad fmt` output runs identical to the original.
2. **godoc + template CLI + `--` markers**: godoc for CodeBegin/End/MixedValue;
   `gad run`/STDIN non-interactive; `--template` flag (ParseMixed + ScanMixed |
   ScanConfigDisabled); `--template-start-delimiter`/`--template-end-delimiter`
   (stored in config `template`); new `{%--`/`--%}` forms (`-` strips blanks up
   to `\n` keeping it; `--` strips ALL blanks); tests + docs + README.
3. **Build tags** to exclude `ide`/`debug` subcommands; make `run` the default.

### `gad fmt` spacing fixes already done this session (UNCOMMITTED in working tree)
- **Stray closing-delimiter whitespace** fixed (`println(a‹TAB›)`→`println(a)`,
  `[1,2‹TAB›]`→`[1, 2]`): the closing `WritePrefix()` in `CallArgs`
  (`literal.go`), `ArrayExpr`/`DictExpr`/`FuncWithMethodsExpr` (`expr.go`),
  `GenDecl` (`stmt_decl.go`) is now guarded by the multiline flag; `CallArgs`
  computes `multiline = n>1 && CallParamsInNewLine` so single-arg calls stay
  inline.
- **Trailing whitespace on blank separator lines** fixed: `WriteStmts`
  (`coder.go`) emits a bare `\n` (not `\n<prefix>`) for the blank line; 6 stale
  goldens in `parser_test.go` (whitespace-only lines) were cleaned.
- Full `go test ./...` was green for these (non-template) changes.

### Mixed-mode investigation findings (key AST facts)
- Scanner (`scanner_scan.go:156-158`): `do`/`then` → `{` (LBrace);
  `done`/`end` → `}` (RBrace). Template block syntax = `do … end`.
- `{% for x in y %}BODY{% end %}` parses to a FLAT top-level
  `[CodeBegin, ForInStmt, CodeEnd, …]` where the **ForInStmt CONTAINS its body**:
  `Body` is a `BlockStmt` with `LBrace=""`, `RBrace="end"`, and
  `Stmts=[CodeEnd "%}", MixedText "BODY", CodeBegin "{%"]`. So the for-body
  block is delimited by the implicit `%}`/`{%` and closed by `end`.
- `ConfigStmt.WriteCode` now emits its trailing `\n`; CodeBegin/CodeEnd depth
  moved into `WriteStmts` (CodeEnd dedents before its separator).
- REMAINING BUG: `{% %}` tags are still REFLOWED (broken onto multiple lines),
  which is INVALID for control-flow openers (`{% for … do %}` must stay inline).
  Fix direction: render each `{% … %}` segment's gad code; keep the tag INLINE
  when single-line (required for for/if/end), expand only when multi-line (e.g.
  the `var (…)` block). Then apply the end/done normalization regex.
- The `do` keyword is already being emitted for the for body (good), but the
  reflow corrupts it. `samples/09_template.gad` is the test fixture; verify by
  `diff` of `gad run` original vs formatted (must be identical).

---

## NEW BATCH (current session)
`ia_todo.md` was reset with 3 new asks:
1. **Bytes from hex string** `h"ffccf1c2"` → `bytes` — **DONE** (committed below).
2. **Bytes from string/rawstring/heredoc/rawheredoc** `b"Hello"` → `bytes` —
   **DONE** (same commit; both share one `BytesLit` node).
3. **Recreate user docs in `./doc`** with examples for all gad features, split
   into multiple files — **DONE** (committed below).
4. **`gad` CLI → subcommands + `fmt` subcommand** (uses
   `github.com/moisespsena-go/command-context`) — **DONE** (committed below).

### CLI subcommands + `fmt` (item 4) — implementation
- `cmd/gad/cmd.go` (new) holds the command tree; `main.go` dispatches a known
  first arg (`run`/`fmt`/`help`) through command-context, else falls back to the
  legacy run/REPL path so `gad FILE`, `gad -`, bare `gad` (REPL) all still work.
  `parseFlags` was split into `registerRunFlags`/`apply` (signature kept for the
  existing tests).
- Formatting mirrors `Parser.FormattedCode`: parse via
  `parser.NewParserWithOptions(...).ParseFile()` then
  `node.Code(file.Stmts, CodeWithFlags(flags), CodeWithPrefix("\t"))`. Top-level
  depth 0 ⇒ no indent; nested blocks indent. (Output parenthesises binary exprs
  and isn't perfect gofmt — that's the project's Coder, intentional.)
- `fmt` flags: `--exclude/--include` (glob, comma+repeatable),
  `--exclude-re/--include-re` (regex, repeatable, NOT comma-split — regex may
  contain commas); filters test BOTH full path and base name; include wins.
  `--backup` + `--backup-format` (BASE_NAME→name w/o ext). `--jobs` (default
  NumCPU; each file/stdin = 1 job, each dir = 1 job; parallel up to N).
  `--out` (single input ⇒ output file; else output dir mirroring tree; inputs
  untouched; stdin⇒stdout). `--no-format` clears the whole Format flag; six
  `--no-*-in-new-line` clear individual bits. `--transpile-*` generated by
  reflection over `node.TranspileOptions` string fields (kebab-cased); any set ⇒
  `CodeTranspile`. `--report` + `--report-format` (yaml default / json) write a
  per-file status doc (`files` + `input_dirs` groups; `error` null on success).
- Config: `.gad.yaml` (default) with root key `fmt:`; `--config`/`--no-config`.
  Config values fill flags NOT set on the CLI (CLI wins). `input_dirs` list
  (path/includes/excludes/includes_re/excludes_re/backup/backup_format/report/
  report_format) — its globs MERGE with the global ones; `backup` defaults false
  per dir; `backup_format`/`report_format` default to the global value; a bare
  `gad fmt` with no args formats the configured input_dirs.
- Failure handling is gofmt-style: a failing file does NOT stop others; per-file
  errors go to stderr; exit code 2 on any failure (via `exitError` returned to
  `main`), else 0.
- Tests in `cmd/gad/cmd_test.go`; docs in `doc/formatting.md` (linked from index
  + getting-started). Dep `command-context` added (now direct in go.mod).

### CLI subcommands + `fmt` (item 4) — plan
- Add dep `github.com/moisespsena-go/command-context`; restructure
  `cmd/gad/main.go` so today's behavior becomes a subcommand (run script / REPL)
  and add a `fmt` subcommand.
- `fmt` args are files or dirs; a dir written `PATH/...` recurses. Hidden files
  are ignored and hidden dirs skipped. Flags:
  - `--exclude GLOB` / `--include GLOB`: repeatable and/or comma-separated globs;
    `--include` re-admits files matched by `--exclude`.
  - `--backup` (bool, default false) + `--backup-format` (default
    `BASE_NAME.backup.gad`): write the original to the backup path before
    rewriting.
- Formatting mirrors `parser/test/Parser.FormattedCode`: parse the file, then
  `node.Code(file.Stmts, CodeWithFlags(CodeWriteContextFlagFormat), …)`. The
  format flag is the OR of six sub-flags; expose a `--no-*-in-new-line` boolean
  per sub-flag that CLEARS that bit (default all on):
  `--no-array-item-in-new-line` (CodeWriteContextFlagFormatArrayItemInNewLine),
  `--no-dict-item-in-new-line`, `--no-key-value-array-item-in-new-line`,
  `--no-call-params-in-new-line`, `--no-parem-values-in-new-line` (note the
  source's "Parem" spelling), `--no-decl-item-in-new-line`.
- NOTE: top-level `File.BuildCode()` emits a compact `;`-joined single-line form
  (binary exprs parenthesised). The multi-line behavior comes from the format
  sub-flags + prefix; need to confirm the exact `node.Code(...)` option set that
  yields good file output before wiring `fmt`.
- Update user docs (`doc/getting-started.md` CLI section) and commit to main.

### User docs (item 3) — what was written
New `./doc/` guide, 13 example-driven markdown files, every snippet verified by
running the built CLI (`./.__tmp/gad`):
- `README.md` (index/TOC), `getting-started.md`, `values-and-types.md`,
  `variables-and-scopes.md`, `operators.md`, `control-flow.md`, `functions.md`,
  `collections.md`, `strings-bytes-regex.md` (covers the new `b"…"`/`h"…"`
  literals), `error-handling.md`, `modules.md`, `builtins.md`, `embedding.md`.
- Root `README.md` now links the guide; its TODO checkboxes for the two bytes
  items are ticked.
- **Verified facts worth remembering** (the old `docs/tutorial.md` is partly
  stale): error fields are lowercase selectors `.name`/`.message` (+ `.New`);
  builtin-error idents (`WrongNumArgumentsError`, …) have `.name` like
  `WrongNumberOfArgumentsError`. Module export is the `export` keyword (NOT an
  `exports` dict): `export x`, `export f(){…}`, `export {a:1}`. Imports use
  explicit paths `import("./mod.gad")`. Modules take params via
  `import("…"; k=v)` (first import only). Template strings are `#"… {expr} …"`.
  Octal is `0NN` (no `0o`/`0b`). `char`+int stays a `char`. `globals()` is a
  `syncDict`. Iterator builtin callback arity differs: `each(key,value)` but
  `map(value,key)`, `filter(value,key,iterable)`, `reduce(acc,value,key)`;
  `map`/`filter`/`keys`/`values` are LAZY — consume via for-in/comprehension or
  `collect`. Embedding API (verified to print `[2,4,6,8]`): `NewBuiltins()` +
  `NewSymbolTable(b.NameSet)` + `Compile(st, src, CompileOptions{})` +
  `NewVM(b.Build(), bc).RunOpts(&RunOpts{Globals, Args})`; Go funcs use
  `&gad.Function{FuncName:…, Value: func(Call)(Object,error)}` (field is
  `FuncName`, not `Name`).

### Bytes literals (items 1+2) — design
- **Scanner** (`parser/scanner_scan.go`): in the identifier case, a 1-letter
  `b`/`h` ident glued (no space) to a `"` or `` ` `` delimiter is a bytes-literal
  prefix. The underlying string is scanned with the existing `ScanString`/
  `ScanRawString` (so `"` may become a Heredoc, `` ` `` a RawHeredoc); the prefix
  is stashed on the PToken via `t.Set(bytesLitPrefixKey, prefix)`. A space breaks
  the literal (`b "x"` is ident+string → parse error), so existing `b`/`h`
  variables are unaffected.
- **AST** (`parser/node/literal.go`): `BytesLit{Prefix BytesLitPrefix, PrefixPos,
  Str Expr}`. `Prefix` is `BytesLitHex`("h") or `BytesLitRaw`("b"). `Str` is the
  inner `*StrLit/*RawStrLit/*HeredocLit/*RawHeredocLit`. `Bytes()` decodes:
  hex → `hex.DecodeString` (whitespace stripped first), raw → `[]byte(content)`.
  `String()`/`WriteCode()` re-emit `prefix + inner`.
- **Parser** (`parser/parser.go`): `ParseLiteral` checks the prefix flag up front
  and delegates to `ParseBytesLit`, which parses the inner literal and wraps it.
- **Compiler** (`compiler.go`): `case *node.BytesLit` → `c.addConstant(Bytes(b))`;
  invalid hex is a compile error (`invalid bytes literal: ...`). No new opcode,
  no encoder change (compiles to the existing `Bytes` object).
- **Tests**: `new_test.go` `TestVMBytesLit` (hex/raw/heredoc/whitespace/empty/
  concat/index + invalid-hex compile errors), `parser/parser_test.go`
  `TestParseBytesLit` + Code round-trips in `TestCodeNewNodes`.

---

5. **CodeMirror 6 Gad plugin + example web app** — **DONE** (committed below).
6. **`cmd/delve` Gad debugger + VS Code plugin + React debug plugin** — **TODO**
   (newest ask): a delve-like debugger for Gad, a vscode-go-like extension to
   drive it, and a React plugin using gad-codemirror to execute/debug source.

### CM6 plugin + web app (item 5) — implementation (all under `web/`)
- `web/gadbridge/` — shared Go core: `Format`/`Diagnose`/`Run` returning
  JSON-friendly structs with positioned `Diagnostic{Line,Column,Message,
  Severity}`. Parse via `NewParserWithOptions(...).ParseFile()`; format via the
  same `node.Code` path; run via compile + `NewVM(...).RunOpts{StdOut,StdErr}`.
  Errors→diagnostics handles `parser.ErrorList`, `*parser.Error` and
  `*gad.CompilerError` (`.FileSet.Position(.Node.Pos())`). Tested
  (`bridge_test.go`).
- `web/server/` — Go HTTP server: POST `/api/fmt|run|diagnose` (in-process via
  gadbridge), CORS, optional static SPA serving. Smoke-tested with curl.
- `web/wasm/` — `//go:build js && wasm`; installs `gadFormat`/`gadRun`/
  `gadDiagnose` globals (source→JSON string) + `gadReady`/`onGadReady`.
- `web/codemirror-gad/` — CM6 plugin (`@gad-lang/codemirror-gad`): StreamLanguage
  tokenizer (`language.ts`), keyword/builtin lists (`keywords.ts`), completion
  (`complete.ts`), async linter mapping line/col→offsets (`lint.ts`), `gad()`
  bundler (`index.ts`). Typechecks clean.
- `web/app/` — Vite+React: `Editor.tsx` (CM6 wrapper, Compartment reconfigures
  diagnose on backend switch), `Formatter` (right editor + LEFT viewer, live
  diagnostics, Format/Format&apply/Run), `Notebook.tsx` (interactive cells),
  backend abstraction (`backends/{server,wasm}.ts`) selectable in the header.
  Builds clean (`pnpm build`).
- Build: `web/app/scripts/build-wasm.sh` builds `gad.wasm` + copies Go's
  `wasm_exec.js` into `web/app/public` (must `chmod u+w` — module-cache copy is
  read-only). `.gitignore` excludes the 16MB wasm + wasm_exec.js (build
  artifacts). Node v26.3.0 + pnpm; workspace `pnpm-workspace.yaml` (esbuild build
  approved via `allowBuilds`).
- Makefile: `build` (=`build-cli`+`build-wasm`), `build-wasm`, `web`
  (install+dev), `web-server`, `web-build`, `web-install`. Docs in
  `web/README.md` + `web/codemirror-gad/README.md`.
- VERIFY UI in a browser next session (couldn't here): WASM load path
  (`/wasm_exec.js`, `/gad.wasm`), linter underlines, backend switch.

### PrismJS plugin + dark/light theme (follow-ups to item 5) — DONE
- `web/prism-gad/` (`@gad-lang/prism-gad`): PrismJS grammar (`registerGad(Prism)`
  → `Prism.languages.gad`); covers comments, string/heredoc/bytes forms, regex
  literals, keywords/atoms/builtins, `@`-specials, numbers, operators. Added to
  the pnpm workspace; typechecks clean.
- App "Highlight" tab (`Highlight.tsx`): static read-only snippets via Prism.
  Token colors come from CSS variables (no imported Prism theme) so they follow
  the app theme.
- Light/dark theme: `useTheme.ts` (localStorage `gad-theme` + prefers-color-
  scheme; sets `<html data-theme>`); pre-paint bootstrap script in `index.html`;
  header toggle; CSS variables for both themes in `styles.css`. CodeMirror uses
  `@codemirror/theme-one-dark` in dark via a theme Compartment (`Editor` takes a
  `dark` prop, threaded through Formatter/Notebook).
- App still builds clean (`pnpm build`).

### Debugger foundation (item 6) — DONE so far (committed below)
Decision (user): **separate debug VM** (not a guarded hot-loop hook), kept in
sync by a Go tool; full-stack scope.
- `cmd/update-delve` — command-context CLI (`gen`/`check`) that GENERATES
  `vm_loop_debug.go` from `vm_loop.go` by renaming `VM.loop`→`VM.loopDebug` and
  injecting `vm.dbg.Step(vm)` right after the instruction fetch
  (`op = Opcode(vm.curInsts[vm.ip])`, the unique anchor). `//go:generate` in
  `vm_debug.go`; `make gen-delve`/`check-delve`; `check-delve` wired into `lint`
  so drift fails CI.
- Production loop UNTOUCHED. `vm.go` gains a nil `dbg DebugStepper` field;
  `vm_run.go safeRun` dispatches `loopDebug()` vs `loop()` once per run (not per
  instruction).
- `vm_debug.go` (hand-written): `DebugStepper{ Step(vm *VM) }`,
  `SetDebugger/Debugger`, and accessors `DebugIP/DebugOpcode/DebugSourcePos`
  (FilePos line/col), `DebugFrames()` ([]DebugFrame{FuncName,Pos}),
  `DebugLocals()`, `DebugAbort()`.
- Tests (`vm_debug_test.go`): stepping records lines; debug run == normal run;
  frames depth + locals observed inside a function call. All green;
  `go test ./...` clean.

### Debugger ENGINE + `gad debug` CLI — DONE (committed below)
- `debug/` package: `Engine` implements `gad.DebugStepper`. `Step(vm)` decides
  to stop (pause / stop-on-entry / breakpoint at new line / step into|over|out by
  comparing frame depth to the depth captured at the last stop) and parks the VM
  on an unbuffered `stops`/`resume` channel handshake — the controller inspects
  `Frames()`/`Locals()` while parked, then `Continue/StepInto/StepOver/StepOut`.
  Stepping is line-granular (`lastLine` tracks the current source line).
  Tested (`debug/engine_test.go`): breakpoint+inspect+step, stop-on-entry,
  step-into/out depth changes, continue-to-completion.
- `gad debug [--break N,... ] [--stop-on-entry] FILE` (`cmd/gad/debug.go`):
  interactive delve-style REPL (c/n/s/o, b/clear, bt, locals, q). Verified
  manually: breakpoint → bt/locals → next → locals(updated) → continue.
  NOTE: locals show generic names `local0..` (slot→name mapping isn't retained
  in CompiledFunction; a future compiler-side debug-symbols enhancement could
  surface real names).
- The user renamed this from `cmd/delve`: it is the `debug` SUBCOMMAND of
  `cmd/gad` (added to `subcommandNames` + root). `cmd/update-delve` (the sync
  tool) is unrelated and keeps its name.

### DAP server — DONE (committed `a7d8be1`)
- `gad debug --dap` (`cmd/gad/dap.go`): stdio DAP server via `google/go-dap`.
  initialize/setBreakpoints/launch/threads/stackTrace/scopes/variables/
  continue/next/stepIn/stepOut/pause/disconnect/terminate +
  stopped/output/exited/terminated events. Program from the launch request.
  Tested in-process (`dap_test.go`). Fixed `DebugFrames` off-by-one
  (frameIndex starts at 1; current frame = frames[frameIndex-1]).

### VS Code extension — DONE (committed below)
- `editors/vscode-gad/`: registers the `gad` language (.gad) + a `gad` debugger
  type whose adapter is `gad debug --dap` (DebugAdapterExecutable; `gad.path`
  setting). package.json contributes languages/breakpoints/debuggers; a config
  provider defaults `program` to `${file}`. Compiles with `tsc` (out/ + node_
  modules gitignored).

### Web Run & Debug page — DONE (committed below) → DEBUGGER FULL-STACK COMPLETE
- `web/server/debug.go`: a request/response debug protocol (stdlib only, no WS):
  POST `/api/debug/start` (compile + run to first stop/end) and
  `/api/debug/command` (continue/next/stepIn/stepOut/pause → next stop/end).
  Each response carries state/reason/line, call stack, locals and new output
  (delta). Sessions kept in a map; removed on terminate. Tested
  (`web/server/debug_test.go`: breakpoint+locals+continue, stepping, compile
  error). curl-verified end-to-end through the real mux.
- `web/app`: `backends/debug.ts` + `Debug.tsx` "Debug" tab — breakpoints input,
  stop-on-entry, Start/Continue/Step Over/In/Out, call stack + locals + output
  panes, editor via gad-codemirror. (Server backend only; the WASM debug
  variant — VM goroutine + JS stop callback — is a possible future add.)
- Whole debugger stack now done: VM debug loop (generated, synced by
  cmd/update-delve) → engine → `gad debug` CLI → DAP → VS Code ext → web page.

### Debugger (ia_todo #17) — DONE (all 6 commits above), marked [x].

7. **`cmd/build-website`** (ia_todo #20) — **DONE** (committed below).
   Full static gad-lang website: language API + user docs (`./doc`, `docs/`)
   with client-side term search, dark/light, WASM playground examples,
   GitHub-Pages-ready; a GitHub Action to rebuild+publish per-commit
   (`/COMMIT-ID`) and to the release version on RELEASE.

### `cmd/build-website` (item 7) — implementation (DONE)
- `cmd/build-website/`: command-context CLI with `build` (`--out` default
  `dist/website`, `--repo .`, `--no-wasm`) and `serve` (`--out`, `--addr :8090`).
- `markdown.go` — dependency-free Markdown renderer (no goldmark; CLAUDE.md
  minimal-deps) for the doc subset: ATX headings (with slug IDs for TOC/anchors),
  fenced code, tables, ordered/unordered + NESTED lists, blockquotes, hr,
  paragraphs, inline code/bold/italic/links; `.md`→`.html` link rewrite
  (README.md→index.html). Returns `[]Heading` for the per-page TOC + search.
- `site.go` — `buildSite`: `collectPages` over curated `guideOrder` (`./doc`) +
  `refOrder` (`docs/`, prefixed `ref-`); README→index.html. Renders each page
  through `layoutTemplate` (sidebar nav + right-hand TOC of H2s), writes
  `search.json` (client index), CSS/JS assets, and (unless `--no-wasm`) builds
  `gad.wasm` (`GOOS=js GOARCH=wasm ./web/wasm`) + copies `wasm_exec.js` from
  GOROOT. 33 pages output; verified `exit 0` with and without WASM.
- `assets.go` — `layoutTemplate` (RELATIVE asset paths so the site works at any
  base path incl. `/<commit-id>/`), `playgroundBody`, `siteCSS` (light/dark via
  `[data-theme]` + pre-paint bootstrap), `themeJS`, `searchJS` (fuzzy
  title/text + keyboard nav), `playJS` (WASM run/format).
- BUG FIXED (caused exit 137 OOM): `renderList`'s nested-list recursion reused
  the inner call's last-consumed index then `continue`d WITHOUT advancing,
  re-entering the same line forever and growing the buffer unbounded. Fix:
  nested call advances `+1` past the last consumed line. Regression test
  `markdown_test.go` (`TestRenderNestedList` + terminates/blocks tests).
- `.github/workflows/website.yml` — on push to `main`, on release `published`,
  and `workflow_dispatch`: builds the site and deploys to gh-pages via
  `peaceiris/actions-gh-pages@v4` with `keep_files: true`, staging into
  `public/<sha>/` (+ `dev/` alias) for commits and `public/<tag>/` (+ `latest/`
  alias) for releases; a root `index.html` redirects to `latest/`. `concurrency:
  website` serialises commit vs release deploys.

### `cmd/build-website` (item 7) — plan
- Go command (command-context subcommands like `gad`): `build` (+ maybe
  `serve` preview). Renders `./doc/*.md` (and `docs/stdlib-*`, `builtins.md`)
  to themed HTML with a sidebar nav; generates a JSON search index + vanilla-JS
  client search; a Playground page loading `gad.wasm`+`wasm_exec.js` with a
  CodeMirror editor (CDN ESM to avoid a bundler). `--base-url` for GH Pages
  project paths (`/gad/`, `/gad/<commit>/`). Output to a `dist/` dir (gitignored).
- Markdown: render with a SMALL in-repo converter (no new dep — CLAUDE.md
  "keep dependencies minimal"; user interrupted a `go get goldmark`). The
  `./doc` files use a regular subset: headings, fenced ```code```, tables,
  lists, blockquotes, hr, links, inline code, bold.
- GitHub Action: build Go + wasm, render site, publish to gh-pages — per-commit
  under `/<commit>` and the release version on release.

### IDE task (ia_todo #23) refinement
- Also: allow panel RESIZING and GROUPING with a tabs layout (added by user).

### `gad ide` (ia_todo #23) — IMPLEMENTED so far (committed below)
Backend + CLI + bundled UI are done and tested; a React/CodeMirror frontend
(served via `--static`) is the remaining polish.
- `web/ide/` package — the IDE HTTP backend, transport-agnostic (`New(path)` →
  `*Server`; `Handler()`):
  - `ide.go`: server, workspace metadata (`/api/ide/workspace`), JSON/CORS
    helpers, optional static SPA serving.
  - `fs.go`: sandboxed workspace filesystem — `tree` (skips hidden +
    node_modules/dist/.git/.__tmp/vendor), `file` GET/PUT, `mkdir`, `delete`,
    `rename`. `resolve()` rejects path traversal outside Root.
  - `config.go`: `.gad.yaml` round-trip as a generic doc (preserves `fmt`, `ide`
    and any other keys); missing file → empty doc.
  - `run.go`: format/diagnose (reuse gadbridge), `modules` list, and `run` with
    options — args (`param (*args)`), per-module enable/disable (helper +
    `ModuleMap.Remove` for the always-on modules), safe mode, and saving
    stdout+stderr to a workspace file.
  - `debug.go`: the request/response debug manager MOVED here from
    web/server/debug.go (now exported `DebugManager`/`StartRequest`/etc.);
    web/server imports it (removed its duplicate + test). gadbridge gained an
    exported `ErrorDiagnostics`.
- `cmd/gad/ide.go` — `gad ide [--addr] [--static] [--no-open] [PATH]`. Default
  addr `0.0.0.0:17000`; `listenWithFallback` scans forward to the next free port
  (≤100) printing an `ALERT:` to STDERR; `browserHost` maps wildcard binds to
  127.0.0.1 for the opened URL. Serves the embedded UI (`//go:embed ideapp`) or
  `--static` build. Registered in `subcommandNames` + root.
- `cmd/gad/ideapp/` — bundled single-file UI (index.html + app.js, vanilla JS,
  no build step): file tree, multi-file tabs, Save/Format/Run/Debug, run+debug
  dialogs (args, module toggles, safe mode, save-output, breakpoints,
  stop-on-entry), call stack + locals panes, light/dark theme, resizable
  sidebar/output gutters, layout + per-file run config persisted to `.gad.yaml`
  `ide` key.
- Tests `web/ide/ide_test.go` (tree/read/write/traversal/delete/rename/config/
  format/diagnose/run+args/save-output/disabled-module/modules/debug-session) —
  all green. `make ide` rule. Docs: `gad ide` section in
  `doc/getting-started.md`.
- `samples/` — a workspace of runnable, commented examples (01–08 language tour,
  `modules/` source modules + imports, `stdlib/` builtin-module usage) plus
  `samples/.gad.yaml` (valid `fmt:` keys = `no-*-in-new-line`, + `ide:` layout).
  `make ide` defaults `DIR=samples`. All 13 verified with `.__tmp/gad run`; the
  module example relies on the IDE's per-file workdir to resolve `./mathx.gad`
  (it needs CWD=its dir under plain `gad run`). README + samples/README +
  getting-started document `gad ide`. NOTE: the IDE fmt settings dialog now
  writes the real inverted `no-*-in-new-line` keys so `gad fmt` accepts them.
  The canonical formatter drops comments, so samples are intentionally not
  auto-formatted.
- React + gad-codemirror IDE (the literal "in React" ask) — DONE: `web/app`
  gains `src/backends/ide.ts` (typed `/api/ide/*` client + `probeIde`) and
  `src/Ide.tsx` (the full IDE: tree, tabs, CodeMirror `Editor`, Save/Format/
  Run/Debug, run+debug dialogs, call stack + locals, settings, theme,
  self-contained `<IdeStyles>`). `src/main.tsx` probes the IDE backend on boot
  and renders `<Ide/>` when present, else the playground `<App/>` — so one Vite
  build serves both. Served via `gad ide --static web/app/dist` / `make
  ide-react`. `pnpm run build` clean; verified `gad ide --static` serves the
  built app + API. NOTE: dist/ is gitignored (build artifact), so the bundled
  vanilla UI remains the default out-of-the-box experience.
- Gutter breakpoints (React UI): `src/breakpointGutter.ts` is a CM6 breakpoint
  gutter — single click on the gutter sets a breakpoint, double-click removes it
  (1-based lines). `Editor.tsx` gains `breakpoints`/`onBreakpointsChange` props
  (controlled, reconciled via `setBreakpointsEffect`). `Ide.tsx` stores them in
  `.gad.yaml` `ide.breakpoints` ({path:[lines]}), feeds them to the debugger
  (the debug dialog no longer has a breakpoints field) and adds a **Breakpoints**
  bottom pane with *Current file* / *All* (grouped-per-file) tabs and per-row
  remove. `pnpm run build` clean.
- Debugger keybindings (React UI): defaults F9=resume/continue, F8=step over,
  F7=step into, Shift+F8=step out. A global keydown handler (active only while
  paused) maps chords→debug commands; `eventToKey` renders chords like
  "Shift+F8". A **⌨ Keys** dialog (`KeybindingsDialog`) rebinds them by capturing
  a keypress; saved under `.gad.yaml` `ide.keys`. Toolbar buttons show their
  shortcut. `pnpm run build` clean.
- #23 COMPLETE (bundled UI + React UI + samples + gutter breakpoints + keys).

### `gad ide` — later enhancement passes (uncommitted batch)
- **Debug decorations** (`web/app/src/debugDecorations.ts`): while paused the
  current line is highlighted, the current node (identifier at the stop column)
  is "super" highlighted, and hovering any identifier that is a current local
  shows a type/value tooltip. Editor gains `debugLine/debugColumn/locals` props
  and scrolls to the stop. Stop line/column come from the top frame.
- **Editor font size**: `Editor` `fontSize` prop via a theme compartment;
  toolbar A−/A+ control; persisted to `.gad.yaml` `ide.fontSize`.
- **Debugger keybindings** (already noted) stored in `ide.keys`.
- **Material UI**: `@mui/material` + icons + emotion added. `Ide.tsx` wrapped in
  `ThemeProvider`/`CssBaseline` (palette follows light/dark); header `AppBar`,
  toolbar `Button`s, and all three dialogs (run/debug, settings, keybindings)
  are MUI. Bundle ~860 KB (acceptable for an IDE).
- **Real local names** (backend): `CompiledFunction.LocalNames` (slot->name) is
  populated by the compiler. Names are recorded at definition time on the
  function-root `SymbolTable` (`localNames` map + `fnRoot()`), so nested-block
  locals are named too (params + body). Round-trips through the encoder (v1 tag
  9). VM `DebugFrame` carries per-frame `Locals`/`LocalNames`; `frameLocals`
  helper reads any frame. Engine `Frame.Locals` + `variablesOf`.
- **Call stack panel**: each frame shows function name + file:line:column and its
  OWN locals. Single click selects a frame and shows its locals in the Locals
  pane (with a 250 ms timer to distinguish); double click opens the file and
  moves the cursor to the position (`Editor.gotoLocation` + active-line
  highlight; bundled app sets the textarea caret). Mirrored in both UIs.
- **Production embed** (`web/app/embed_prod.go`, build tag `prod`;
  `embed_dev.go` for `!prod`): `//go:embed all:dist` packages the built React
  app. `gad ide` serves the embedded SPA (`spaFSServer`) when present, else the
  bundled UI, else `--static`. `make dist` (web-build + `go build -tags
  prod`). Verified: 36 MB binary serves the hashed React assets + API.
- Tests: `web/ide` `TestDebugFramesCarryLocals` (2 frames, named inner locals).
  Full `go test ./...` green; `pnpm run build` clean.

### REMAINING asks (ia_todo)
- #20 `cmd/build-website` (in progress).
- #23 `gad ide` subcommand: launch the React web app as a tabbed multi-file IDE
  (format/run/debug buttons, fmt settings ↔ `.gad.yaml`, movable/hideable panels
  saved to `.gad.yaml` `ide` key, per-file run/debug dialogs: params,
  enable/disable builtin modules, save STDOUT/STDERR to file).

### REMAINING asks (ia_todo)
- Build a full gad-lang website (dark/light, WASM examples, GitHub-Pages-ready)
  + a GitHub Action to auto-rebuild and publish it.

### CM6 plugin + web app (item 5) — plan
Deliverables (under `web/`):
- `web/codemirror-gad/` — a CodeMirror 6 plugin package (TS): Gad syntax
  highlighting, autocompletion (keywords + builtins), and a linter that turns
  `{line, column, message, severity}` diagnostics into CM6 `Diagnostic`s. The
  diagnose/format/run backend is injected so the same plugin works against the
  HTTP server OR the WASM module.
- A **diagnostics/format/run bridge** in Go reused by both backends:
  - HTTP server (`web/server/`, Go): endpoints `/fmt` (source→formatted +
    per-line/col errors) and `/run` (source→stdout/stderr). `gad` itself can
    `fmt -`/`run -` from stdin→stdout; the server reuses the gad packages
    in-process (parse+format via the same Coder; compile+run for execution).
  - WASM (`web/wasm/`, `//go:build js,wasm`): exposes `gadFormat`, `gadRun`,
    `gadDiagnose` to JS via syscall/js.
- React app (`web/app/`, Vite + pnpm, node v26.3.0): editor on the right, the
  formatted/output on the LEFT viewer; errors reported inline per line/col.
  Two examples: (a) backend-server-powered, (b) WASM-powered. Plus a
  notebook-like interactive execution example.

NOTE: must use pnpm (never npm/yarn) and `nvm use v26.3.0`. Reuse the parser
`ParseFile`/`node.Code` formatting path and the eval/VM run path; surface
parser error positions (line/column) for diagnostics. Explore existing
WASM/JS infra first (CLAUDE.md mentions a WASM playground ecosystem).

---

## PRIOR BATCH — ALL DONE
Every item in the previous `ia_todo.md` is implemented, each with parser +
compiler + VM tests (plus encoder tests where relevant), `make test` green, and
committed to `main`. Commits (newest first):

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

## PENDING (current asks)
- **README** (DONE, ready to commit): fixed the Go-embedding example to the
  current API — `gad.NewBuiltins()` + `NewSymbolTable(b.NameSet)` +
  `Compile(st, []byte, CompileOptions{})` + `NewVM(b.Build(), bc).RunOpts(&RunOpts{
  Globals: Dict{...}, Args: Args{Array{...}}})`. Also `gad.Map`->`gad.Dict`,
  `param ...args`->`param *args` (Go-style `...` is invalid gad), and
  Fibonacci `fib(arg0)`->`fib(int(arg0))` (CLI args are strings). Verified the
  Go example outputs `[2, 4, 6, 8]`.
- **User docs** (ia_todo): recreate comprehensive user docs in ./docs with
  examples/variations for ALL gad features (incl. the new ones); split into
  multiple files as needed.
- **Bytes literals** (ia_todo): `h"ffccf1c2"` -> bytes from hex; `b"Hello"`
  (and `b` + raw string / heredoc / rawheredoc) -> bytes from string content.
  Needs scanner prefix handling (like `raw`/template), AST node, compiler, VM
  tests. `typeof data == bytes`.

## DONE TASK (ia_todo.md)
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
