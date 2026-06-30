# IDE epic (`web/ide` backend + `web/app` React frontend)

Split from the original mega-bullet. Backend changes are unit-tested in
`web/ide`; frontend changes are typechecked (`tsc --noEmit`) — most need browser
verification for full sign-off.

## Done
- [x] Explorer: show/hide hidden files toggle (backend `?hidden=true` + eye button).
- [x] Locals panel: per-row copy-to-clipboard button.
- [x] Explorer: F2 / right-click context menu (open, run, format, transpile, rename, remove).
- [x] Remove confirmation dialog with recursive checkbox for non-empty dirs.
- [x] Run output: split into stdout/stderr files + combine flag (backend + API).
- [x] Explorer: "get file from web" backend (`/api/ide/fetch`, URL → workspace path).
- [x] Transpile: `gadbridge.Transpile` + `/api/ide/transpile`, config-driven `TranspileOptions`.
- [x] Settings dialog: Transpile options (writeFunc, rawStrFunc start/end → `.gad.yaml`).
- [x] Editor controls: reload-from-disk, undo, redo buttons.
- [x] Error dialog when a backend request fails (status line + modal, copyable).
- [x] Evaluate panel: add/edit/remove expressions, repr flag, output dialog, copy,
      re-eval on debug step. Backend `/api/ide/eval` (standalone, file as prelude).

## To do
- [x] gad codemirror plugin not working in the IDE: root cause was the default
      `gad ide` serving the build-free `cmd/gad/ideapp` (plain textarea, no
      CodeMirror) — the plugin only loaded in the React `web/app` (prod/--static
      build). Fixed by loading CodeMirror 6 + the gad language into the build-free
      app from a pinned ESM CDN, with a textarea fallback when offline.
- [ ] Run/Debug settings: one dialog with Run and Debug tabs; surface the split
      stdout/stderr file fields + combine flag in the Run tab.
- [ ] Explorer header: "get file from web" dialog (URL, output name, target dir).
- [x] Breakpoint condition + disabled: debug.Engine honours per-breakpoint
      Disabled flag and a Gad `condition` (evaluated in the paused frame's scope,
      pauses only when truthy); wired through /api/ide/debug/start
      (BreakpointSpecs). Breakpoints panel entry opens a dialog to edit
      disabled/condition (stored in .gad.yaml ide.breakpointMeta).
- [ ] Breakpoints panel: double-click an entry to navigate to its location
      (remove button + click-to-edit-condition done).
- [ ] Tooltip for builtin-value identifiers in the gad editor.
- [x] Evaluate panel: debug-session-aware eval. debug.Engine.EvalInFrame
      evaluates against the paused frame's locals; /api/ide/debug/eval exposes
      it; the panel routes through it while a session is paused (and re-evaluates
      on each step via a debugLoc effect), falling back to standalone eval
      otherwise.
- [ ] Multi-format editors: JSON, YAML, HTML, CSS, SCSS, JS (TS/JSX); plain-text
      fallback for other types.
- [ ] Tooltip: copy-to-clipboard button on the gad editor hover tooltip.
- [ ] codemirror plugin: editor features (autocomplete, etc.) inside template strings.
- [x] Right closable doc-comments panel: backend gadbridge.DocComments +
      /api/ide/doc extract `/?`/`/??`/`/???` docs (kind, title=following code
      line, content, line). Toolbar "Docs" toggle; panel lists entries (click to
      jump to the line), with a manual reload and auto-reload 5s after an edit.

- [x] STEP IN module import now opens the module file: the debugger reported
      imported-module frames as absolute `file:<abs>` paths (and the main file as
      `(main)`), and the UI never followed the stop into another file. Backend now
      normalizes debug file names to workspace-relative paths (DebugResponse.File
      + per-frame File via Server.normalizeDebugFile); the UI opens the stop's
      file and highlights the line there (debug.path follows res.file).
- [x] evaluate `1 + 1` showing the file's value instead of `2`: the file prelude
      ended in a top-level `return` which short-circuited the eval. Fixed with
      gadbridge.EvalSource (strips top-level returns, keeps definitions). repr's
      `‹int: 2›` is the correct repr form.
- [~] tree navigator of container values (dict/array/module/…): backend
      gadbridge.InspectObject + /api/ide/inspect (frame-scoped or standalone);
      lazy-drill TreeNavigator + InspectDialog wired into the locals and evaluate
      panels. TODO: the tooltip surface (CodeMirror hover widget).
      add per entry button to goto declaration source. 
- [x] doc panel renders Markdown with gad code highlighting: docMarkdown.ts
      (dependency-free) renders headings/lists/blockquotes/inline code+bold and
      fenced code blocks highlighted with the prism-gad grammar (Prism.highlight).
- [~] colorize doc comments in the plugins: codemirror-gad (StreamLanguage),
      prism-gad (grammar) and the build-free ideapp tokenizer now recognize `/?`
      single + `/??`…`??` / `/???`…`???` block doc comments as a docComment token
      (verified: prism tokenizes them distinctly). TODO: highlight embedded source
      code and `>>>` result blocks inside doc comments (sub-language regions).
- [x] module/function values rendered absolute file paths (baked into their
      ToString) in locals/eval/inspect. The IDE now rewrites workspace-absolute
      paths (and `file:` prefixes) to workspace-relative in debug locals/frames,
      eval and inspect output (Server.relativizeValue via DebugManager hook).
      Test TestDebugModulePathsAreRelative.
- [ ] change input fields of evaluate and breakpoint condition to use codemirror plugin. 
- [x] FORMAT not preserving comments/doc comments: gadbridge.Format parsed
      without ParseComments and omitted CodeWithComments, so it dropped all
      comments. Now parses with ParseComments and re-emits via CodeWithComments
      (matching the CLI). Fixes the IDE, web server and WASM playground formatters.
      Regression test TestFormatPreservesComments.
- [x] separate STDERR/STDOUT streaming output: debug sessions now capture stdout
      and stderr into distinct buffers (DebugResponse.Stdout/Stderr deltas; run
      already separated them). Output pane is a stream-tagged log with a
      Combined/Split toggle; stderr is colorized red (combined inline, split as a
      second column). Test TestDebugSeparatesStdoutStderr.

## Editor plugins (separate)
(none — keyword/builtin sync commands shipped via cmd/update-*-plugin.)

- [~] VS Code plugin: cmd/update-vscode-plugin generates a TextMate grammar
      (was missing — no highlighting) from the shared vocabulary; package.json
      gains the grammars contribution, .gadt ext, and a gad.format.useConfig
      setting (gad.path existed). DAP gained EvaluateRequest (Watch/Console/hover
      via EvalInFrame) and conditional breakpoints. Tested: TestTextMateGrammar,
      TestDAPSession (evaluate). TODO: format-on-save / .gad.yaml-driven format
      command, and richer IDE-like panels (inspect/doc) inside VS Code.
- [ ] create plugin like vscode to JetBrains.

# Language

- [x] new doc-comment markers: SINGLE `/?`→`///` (`////` is a normal comment),
      BLOCK `/??`…`??`→`/**`…`**/`, ROOT_BLOCK `/???`…`???`→`/***`…`***/`. Scanner,
      parser attachment, formatter, gadbridge, doc generator, doctest, all four
      tokenizers (codemirror/prism/vscode-grammar/ideapp, verified with the real
      engines), docs, samples/16_doc_comments.gad and all doc tests updated.
      (Pre-existing unrelated regression: Class(...; fields=...) rejects `fields`
      → fails TestVMWith/TestVMBinaryIncDec/TestREPL on HEAD.)
- [x] create samples for `Heredoc` and`RawHeredoc`. `Template` of `Heredoc` and `RawHeredoc`. samples for singleline and multiline variations. update docs.
      Done: samples/21_heredocs.gad + doc/strings-bytes-regex.md (Heredocs +
      Template Strings sections). Bonus fix: backslash broke single-line raw
      heredocs (`` ```a\tb``` ``) — ReadAt no longer treats `\` as an escape
      (`a01abce`). Sample/docs `550745b`.
  Examples:
  - `s := """abc""de""" // abc""de`
  - `s := """\n\tabc""\n\tde\n""" // abc""\nde`
- [x] create detailed samples and docs form keyValueArray with functions, closures, typed ... (see `TestParseKeyValueArray` and `TestVMKeyValueArray`)
      Done: samples/22_key_value_array.gad + doc/values-and-types.md
      (keyValue/keyValueArray section): flags, func/closure values, typed keys,
      dup keys, flag/values/delete, indexing+mutation, spread, dict(). `0c51b90`.
- [x] create parser for `enum`.
  - DONE: Enum keyword, EnumExpr/EnumStmt AST, parser (expr/stmt/export),
    compiler value engine + compile-time expr evaluator (increment, signs,
    bit, `_`, refs like `All = Read | Write`), EnumValue `.name/.value/.index/
    .enum`, encoder/decoder (`5683ec9`, `5789943`). Formatter doc-claiming +
    a block-doc grouping fix, godoc, samples/20_enum.gad, doc/enums.md
    (`29fdc21`). Tests: TestParseEnum, TestVMEnum, TestEnum*Encoding*.
    NOTE: 3 spec examples mutually contradict (sign/type of `ReadOnly,-Write`
    vs `ReadOnly,-Write=1`; `-_`; `-Write=1`→Delete) — implemented a
    consistent left-to-right model (14/17 examples match); `bit _` left as
    "advance bit position, not added". Revisit if exact cases needed.
  - Expr syntaxe `enum { [bit] IDENT [= Expr] [, IDENT [= Expr ]...] }`, usage `x := enum { ... }`.
  - Stmt syntaxe like Expr, but have Ident `enum IDENT { ... }`, it compiles to `const IDENT = enum { ... }`.
    exports variant `export enum IDENT { ... }` compiles to `enum IDENT { ...}; export IDENT`.
  - items sep is `,` (comma) or new lines `\n`.
  - `enum` compiles to Compiler.constants `NewEnum(name, compiler.module).AddValue(NAME, VALUE)`
  - `enum` values must accept `int|uint` value types.
  - implements encoder/decoder and tests for enum with fields: `Module.Index` (like `CompileFuction`), `Name` and `Fields`.
  - if value of field isn't set, default value is `SIG ((PREV_FIELD ?? 1u) + 1)` when `SIGN` is a prev field sign `+` (default) or `-`.
  - bitwise mode activation with `bit` (`ident`). `enum { bit A }` (activates bitwise mode to current field and nexts).
    if value of field isn't set, default value is `1 << ((PREV_FIELD ?? 1u) + 1)`.
  - `enum` and your fields accept doc strings.
  - field `_` isn't compilable, but must set prev value of next field. `enum` accept many `_` fields. 
  - examples:
    - `enum { ReadOnly, Write }` (`ReadOnly=1u, Write=2u`).
    - `enum { +ReadOnly, Write }` (`ReadOnly=1, Write=2`), `+FIELD` or `-FIELD` take it as signed integer `int` with sign `+` or `-`.
    - `enum { -ReadOnly, Write }` (`ReadOnly=-1, Write=-2`).
    - `enum { ReadOnly, -Write, Delete }` (`ReadOnly=1, Write=-2, Delete=-3`), `-FIELD` take all values as signed integer `int` and set `FIELD` value to negative.
    - `enum { -ReadOnly, Write, +List, Delete }` (`ReadOnly=-1, Write=-2, List=3, Delete=4`).
    - `enum { -ReadOnly, Write, List=1, Delete }` (`ReadOnly=-1, Write=-2, List=1, Delete=2`).
    - `enum { -ReadOnly, Write, List=1u, Delete }` (`ReadOnly=-1, Write=-2, List=1u, Delete=2u`).
    - `enum { ReadOnly = 10, Write }` (`ReadOnly=10, Write=11`).
    - `enum { ReadOnly, Write, All = ReadOnly + Write }` (`ReadOnly=1u, Write=2u, All=3u`).
    - `enum { bit List, Detail, Create, Edit, Delete, Read=List|Detail, Write=Create|Eit }` (`List=1<<0, Detail=1<<1, Create=1<<2, Edit=1<<3, Delete=1<<4, Read=List|Detail, Write=Create|Eit`).
    - `enum { _, ReadOnly, Write }` (`_=1u, ReadOnly=2u, Write=3u`).
    - `enum { _ = 10u, ReadOnly, Write }` (`_=10u, ReadOnly=11, Write=12`).
    - `enum { _ = -1, ReadOnly, Write }`, (`_=-1, ReadOnly=-2, Write=-3`).
    - `enum { bit _, List, Detail }` (`_ = 1u << 1, List = `).
    - `enum { bit _ = 10, List, Detail }`, first field is `_` ignore it, but List starts at  `1<<11` instead of `1<<12`.
    - `enum { ReadOnly, _, Write }` (`ReadOnly=1u, Write=3u`).
    - `enum { ReadOnly, _ = 6, Write }` (`ReadOnly=1u, Write=7`).
    - `enum { ReadOnly, -_, Write }` (`ReadOnly=1u, Write=-2`).
    - `enum { ReadOnly, -Write=1, Delete }` (`ReadOnly=1u, Write=-1, Delete=-1`).
  - create format for here where putting all fields into new indented line without comma sep:
    ```gad
    enum { ReadOnly, Write, Execute = 10 }
    ```
    to
    ```gad
    enum { 
        ReadOnly
        Write
        Execute = 10
    }
    ```
  - the doc strings describes type and has a table of fields with values and your doc string. 
  - create go doc, expansive samples with doc comments, docs, parser/compiler/vm tests for indexGet, iterable, str, repr
- [x] create parser for `class`.
  - DONE: Class keyword, ClassExpr/ClassStmt AST, parser (expr/stmt/export),
    compiler lowering to `Class(name; define=(Type,define)=>define(;extends,
    fields,properties,methods,new))` with `this` injection, formatter, godoc,
    samples/19_class_syntax.gad, doc/classes.md (`c587c1e`, `7067dcb`,
    `267fd13`, `5f4aa85`). Fixed NewClassFunc define-callback (`c587c1e`) and a
    VM ParamTypes panic (`7964f84`). Tests: TestVMClassSyntax, TestParseClass.
    NOTE: methods get typed `this Type`; property accessors and constructors
    get an UNTYPED `this` — typed there resolves `Type` to the instance at
    invocation time (no dispatch value anyway). A method can't reference its
    own class by name during its initializer (same as hand-written Class()).
  - items sep is `,` (comma) or new lines `\n`.
    Expr syntaxe, auto insert `this` param as first param of properties, constructors and methods:
    ```gad
    /// this is my class example 
    class [extends A, B] {
        withoutValueField
        withValueField = 1
        valueFieldWithComputedValue = (= 1)
    
        // properties, auto insert `this` param as first param
        props {
            a() => 1 /// a is must getter like `ClosureExpr` declaration
      
            b = this.a /// b is must getter (shortcut version of `a()`)
      
            b(v) { this._b = v } /// b is must single setter
      
            c(v) => this._c = v /// c is must single setter like `ClosureExpr` declaration
      
            /// like `prop` declaration without `prop` keyword       
            d {
                () => this._d /// is a getter
                (v int) => this._d = v /// is a setter if int value
                  
                /// setter of str value
                (v str) { this._d = v }
            }
        }
    
        /// single constructuor
        new(;_b=0,_c,**fields) => this(;**fields)
      
        // or with methods declaration (bellow), not both
      
        /// constructor with methods, like `FuncWithMethodsExpr` declaration without `func` keyword
        new {
           (b int) => this(;_b=b)
           (b int, c int) {
                this(;_b=b, _c=c)
           }
        }
    
        // methods declarations
        methods {
          /// single method like `ClosureExpr` declaration
          done() => this._done = true
      
          /// shortcut version of `done()`
          done = this._done = true
      
          start() {
              this._started = true
          }
      
          /// method with methods, like `FuncWithMethodsExpr` declaration without `func` keyword
          build {
              () => this._builded = true
              (v int) this._builded = v
              (v int, x int) this._builded = [v, x]
          }
        }
    }
    ```
  - Stmt syntaxe `class IDENT [extends A, B ...] { ... }`, it compiles to `const IDENT = Class("IDENT"; define(Type, define) => define(; new=..., methods=..., fields=... ))`.
    Stmt syntaxe, auto insert `this Type` param as first param of properties, constructors and methods.
    exports variant `export class IDENT { ... }` compiles to `class IDENT {...}; export IDENT`.
  - the bellow example is the code format model.
  - the doc strings describes class with subsections "fields" (fields table with Name, type, value, description (doc string of field) columns),
    "constructor" and "methods".

- [ ] change doc generator to group consts, classes, enums, functions, methods, properties in sections. create table of contents.