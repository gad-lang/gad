# IDE epic (`web/ide` backend + `web/app` React frontend)
- [x] change ui to use `dockview-react` package with dock/undock, move, resizable panels. 
  save all ui panel config (and positions and sizes) to config file.
  create button on top bar to reset panels config.
  Done: Ide.tsx refactored — IDE shell replaced by DockviewReact with 5 panel
  components (ExplorerPanel, EditorPanel, OutputPanel, DocsPanel, MdPreviewPanel)
  shared via IdeCtx. Layout serialised to config.ide.panels on every change via
  onDidLayoutChange→saveConfig. Restored via api.fromJSON on onReady. "Reset
  Panels" button in AppBar clears config.ide.panels + rebuilds default layout.
  dockview-react@7.0.2 installed; CSS theme vars mapped to existing --bg/--panel vars.
  Typecheck + vite build → exit 0.
- [x] change ui to use plugin for edit `.md` files and render it in left tab like `DOCS`.
  Done: Editor.tsx gains @codemirror/lang-markdown + "markdown" language case;
  langForPath maps .md/.mdx → "markdown". MdPreviewPanel renders renderDocMarkdown
  of the active .md file's content as a dockview tab in the left (Explorer) panel
  group. Panel auto-opens when a .md file becomes active; auto-closes when the
  active tab is no longer .md.
- [x] Gad identity icons + `.gadt` template support across the IDE and editor plugins.
      Done: assets/identity gad+gadt icon sets, wired into README/IDE header/explorer/
      favicon; fixed malformed xmlns (commit bbe37ba). Runnable samples/23_template.gadt
      (ec65fd3). Run/Debug honour `.gadt` mixed mode with delimiters from `.gad.yaml`
      `template:` + a Settings "Template" tab (041a7d2). Output panel renders stdout as
      JSON/HTML/Markdown, and JSON/HTML/MD source in a read-only CodeMirror with folding
      (46fdb03, 8122040). `.gadt` editor highlighting: merged gad({template,delimiters,
      preamble}) in @gad-lang/codemirror-gad + registerGadTemplate/detectGadTemplate in
      @gad-lang/prism-gad; `.gad` files with `# gad: mixed` detected from content and
      highlighted as templates; delimiters tagged `tagName` for theme-driven colour
      (3b36968). Evidence: tsc + `bun run build` → exit 0; tokenizer/detection runtime-
      tested on samples/09_template.gad.

# web/js projects
- [x] replace all runner of `pnpm` to `bun`, update docs and scripts.
      Done: pnpm→bun in Makefile (install/run/package), web/app/package.json scripts,
      editors/vscode-gad/package.json scripts (pnpm dlx→bunx), web/README.md,
      editors/vscode-gad/README.md, CLAUDE.md. Created web/package.json with bun
      workspaces + trustedDependencies (replaces pnpm-workspace.yaml, which was
      deleted). Removed pnpm-lock.yaml files; regenerated bun.lockb in both
      web/ and editors/vscode-gad/. `bun install` in both → exit 0.

# Language
- [x] update `github.com/moisespsena-go/command-context` dependency to `46d8492`. update usage for here applying Patterns.
      Done: go get @46d8492 (v0.0.0-20260630150637-46d849278485). Applied the
      README Patterns: ide cmd now uses an ideOptions struct via WithValue/Value
      (was closure pointers) + captures ctx.Flags() in a local var; debug cmd
      captures ctx.Flags() locally. The fixed Args.Max lets debug --dap use
      ctx.Args.Max(1) (removed the inverted-check workaround). go mod tidy.
- [x] when generate doc of `07_error_handling.gad`, the `err` variable is generating into constants section. put variables into "Variables" section.
      Done: docVar entry kind; `:=` value bindings and `var` decls bucket into a
      "Variables" section (exports/`const` stay Constants). docBuckets/bucketize;
      TOC + Exported/Internal writers updated. Test TestGenerateDocVariablesSection.
- [x] change `class` declaration (Expr and Stmt) from syntaxe `class [NAME] [extends ...] {` to `class [NAME] { extends { Parent [: Alias], ... } ... }` (extends itens separated by `,` or `\n`. 
      Parent `alias` is optional, separated by `:`. `Parent` is `IdentExpr` or `SelectorExpr`, example: `class { extends { mod1.A, mod2.A: A2 } }` (`A2` is alias of `mod2.A`) ).
      format `WriteCode` extends section itens to new indented line.
      Done (commit c8e2f1e): extends-block syntax + alias via `:`; also reworked
      Class(name, define) (positional handler) and construction (cls + `new`
      ClassInitiator, Class.New(Call) Go API). Samples/tests/docs migrated.
- [x] rename module "core" to "gad". update samples, docs and tests.
- [x] create builtin functions `gad.binOp{OP_NAME}` (for binary operators), `gad.unOp{OP_NAME}` (for unary operators),
      `gad.selfAssigOp{OP_NAME}` (for self assign operators) removing first param `op Operator`. use call to `gad.binOp{OP_NAME}` insteadof `gad.binOp`,
      apply this rule for `gad.unOp` and `gad.selfAssigOp{OP_NAME}`. update methods implementations in tests, samples, doc, README.
      Done: per-op builtins gad.binOp{Op}/unOp{Op}/selfAssignOp{Op} dispatched via
      VM.callBinaryOp/UnaryOp/SelfAssignOp ([256]BuiltinType routing); selfAssignOp
      fallback routes through gad.binOp{Op} so user overloads back `x op= y`. The
      builtins now live at the static Builtin{Binary,Unary,SelfAssign}Operator{Op}
      enum keys (moved before BuiltinEnd_) so Builtins.build() clones them per VM
      (no cross-test/run leak). Removed the T{Binary,Unary,SelfAssign}Operator{Name}
      constants + deleted generated builtin_operators.go; cmd/mkoptypes no longer
      emits them (still regenerates the enum groups + op_api.go, verified
      idempotent). Updated doc/embedding.md, doc/operators.md (+ core→gad namespace
      anchor in control-flow.md), samples/17_unary_operators.gad (+regenerated
      samples/doc). Evidence: gofmt/vet clean, `go test ./...` → 0 failures.
- [x] `met` override modifier: `met ~name(…)` and `met name { ~(…){…} }` (also
      `met ~name { … }` applies to all block methods). Re-adds an existing method
      signature by replacing it instead of erroring.
      Done: new OpAddMethodOverride opcode; FuncExpr.Override / FuncMethod.Override
      AST fields (parsed from `~` after `met`/before block method params, rendered
      by WriteCode); compileAddMethodsExpr groups methods by override flag and emits
      OpAddMethod/OpAddMethodOverride; VM.xAddMethod(override) sets NamedArgs
      override=true into BuiltinAddMethodFunc. Also fixed cross-VM state leak: the
      per-operator gad.binOp{Op} builtins now use the static
      Builtin{Binary,Unary,SelfAssign}Operator{Op} enum keys (moved before
      BuiltinEnd_) so Builtins.build() clones them per VM — `met` overloads no
      longer pollute the global builtin across tests/runs.
      Evidence: `go test ./...` → 0 failures; `TestVMMethodOverride` (single/block/
      block-wide `~` + non-override duplicate → ErrNotIndexable) passes in both the
      default and unoptimized subtests.
- [x] remove `subcommandNames` from gad/cmd, it is unnecessary after dependency update.
      Done: dropped the subcommandNames map + isSubcommand; main() builds the root
      command and dispatches via root.IsSub(args[0]) (plus help/--help), using the
      new command-context Command.IsSub. registerCommand just appends factories.
- [x] change func-header to compile to bytecode constant insteadof call builtin (see `CompiledFunction` header for params, types and symbols),
      use `*Compiler.module` to get current `*ModuleSpec`.
      create encode/decode.
      Done: compileFuncHeaderExpr now builds a *FuncHeaderObject constant
      (OpConstant) instead of a FunctionHeader(...) builtin call. Param/return
      types are stored as compile-time symbols (TypedIdent.TypesSymbols, via the
      shared nameSymbolsOfTypedIdent) and resolved to ObjectType per-VM on
      IndexGet("types") through vm.GetSymbolValue (mirrors ReturnVars.VMTypes), so
      the shared constant stays immutable/thread-safe. Untyped params default to
      TAny. The header carries c.module for a module-qualified FullName
      (MODULE.Name + "." + name); FuncHeaderObject.ToString/String render the
      parsed syntax with FullName. Anonymous headers get an incremented `fh#N`
      name (Compiler.newFuncHeaderName, like newAnonymousFuncName). Added
      encoder/decoder for TypedIdent + FuncHeaderObject (encoder_v1_func_header.go,
      reusing the SymbolInfo array codec; module stored by name).
      Evidence: gofmt/vet clean, `go build ./...` and `go test ./...` → 0 failures;
      TestVMFuncHeaderExpr updated; TestFuncHeaderObjectEncoding round-trips.
- [x] change `meti` parser to parser function header without param name, parsing `(int)`
      as `(_ int)` insteadof untyped param `int`. apply this rule to parse `FuncHeader` (A func-header declaration value `<…>`).
      update godoc, doc, samples, and tests. update tests and docs.
      compiles to bytecode constant insteadof call builtin (see `CompiledFunction` header for params, types and symbols),
      use `*Compiler.module` to get current `*ModuleSpec`.
      create encode/decode. update tests and docs.
      if is anonymous, compile with name `meti#N` (see compiler of FuncHeader).
      Done: new MultiParenExpr.ToFuncHeaderParams rewrites a bare positional entry
      to an unnamed typed param (`(int)` -> `(_ int)`, incl. selectors); applied by
      parseInterfaceHeader + ParseFuncHeaderExpr only (regular `func(...)` params
      unchanged). `meti { … }` now compiles to a *MethodInterface bytecode constant
      (buildMethodInterfaceInstance, headers via the shared buildFuncHeaderObject)
      instead of a MethodInterface(...) builtin call; anonymous → `meti#N`
      (Compiler.newMethodInterfaceName). Renamed MethodInterfaceInstance ->
      MethodInterface (no alias; Type() → TMethodInterface). Added encoder/decoder
      for MethodInterface (reuses the FuncHeaderObject codec). Docs
      (doc/method-interfaces.md) + tests (parser + VM + encoder round-trip) updated.
      Evidence: gofmt/vet clean, `go build ./...` and `go test ./...` → 0 failures;
      samples/12_method_interfaces.gad runs.
- [x] create parser for `interface` Expr and Stmt.
      Done (staged commits 763a5ee→3fbfc25): token.Interface keyword; AST
      InterfaceExpr/Stmt/InterfaceMemberExpr/InterfaceMethodExpr with extends,
      typed fields, get/set/prop accessors and methods (single `name(sig)` or
      block `name { sig, … }` — the `parse` example is just a block-form method,
      meti-style without the keyword); WriteCode + parser tests. Objects Interface
      (TInterface in gad, no constructor) + InterfaceField/Prop/Method with
      IndexGet + fluid WithField/WithGetter/WithSetter/WithMethod. Compiler lowers
      to a *Interface constant (buildInterface; types as symbols via
      nameSymbolsOfTypedIdent; getters/setters -> InterfaceProp, methods ->
      InterfaceMethod; extends -> parent symbols; module from c.module), anonymous
      -> ifaces#N; statement/export forms bind a const. Union types in header
      params (`(int|uint)` -> `(_ int|uint)`). Encode/decode for all four
      constants (member Iface back-refs restored on decode). doc/method-interfaces.md
      section + samples/24_interfaces.gad. Evidence: gofmt/vet clean,
      `go build ./...` and `go test ./...` -> 0 failures; sample runs.
      NOTE: `<false>`/`<true>` boolean returns (used in the parse example) need a
      small ParseFuncReturnTypes addition — deferred (see the meti follow-ups).
  - syntaxe:

      ```gad
      MyInterface := interface [NAME] {
        extends { ... } // like class extends, but without alias
        
        // fields
        fistName
        lastName str // typed field, allow many types `int|uint`
        birthday time.calendarTime
      
        // getters
        get fullName
        get yeadsOld uint|int // typed getter
      
        // setters
        set fullName
        set yeadsOld uint|int // typed setter
        
        // properties, is a shortcut form of getter and setter
        prop pFullName
        prop pYeadsOld uint|int
        
        // methods, like `meti`
        authorName() // header must <authorName()>
        authorName2() <str> // with return type
        update(int|uint) <str> // func header <(_ int|uint)<str>>
      
        // method "parse" with `meti` declaration (without `meti`) insteadof MetiToken field, preserve compile source positions
        parse {
          (str) // func header <(_ str)>
          (str,int) // func header <(_ str, _ int)>
          (v int|uint)<false> // func header <(v int|uint) <false>>
        }
      }
      ```    

  - Stmt variation `interface X {...}`, compiles to `const X = interface X { ... }`
  - allow doc comments and export `export interface X { ... }`
  - create new Object `&InterfaceProp{ Iface *Interface; Name string; Getter *FunctionHeader; Setters[]*FunctionHeader }`
  - create new Object `&InterfaceField{ Iface *Interface; Name string; TypesSymbols ParamType; Types ObjectTypes }` (see `gad.Param`)
  - create new Object `&InterfaceMethod{ Iface *Interface; Name string; Headers[]*FunctionHeader }`
  - compiles to new constant `*gad.Interface{module *ModuleSpec; Fields []*InterfaceField; Props []*InterfaceProp; ...}` (type is new object type "Interface" in "gad" module, without constructor) (see `CompiledFunction` header for params, types and symbols, use `*Compiler.module` to get current `*ModuleSpec`).
  - compile getters to const `&InterfaceProp{Name, Getter, ...}`
  - compile setters to const `&InterfaceProp{Name, Setters, ... }`
  - compile methods to const `&InterfaceMethod{Name, Headers, ... }`
  - if is anonymous, compile with name `ifaces#N` (see compiler of FuncHeader).
  - create methods for `*gad.Interface` for fluid construction appending fields/getters/setters/methods (methods is *MethodInterface)
  - format like this task example.
  - create encode/decode.
  - create expansive docs, tests and examples.
- [x] change `meti` parser allow shortcut form (one method): `meti<...>` when `<...>` is a func-header declaration. 
      parses `meti<(v)>` as `meti<(_ v)>` (`_` param with type `v`, not untyped `v` param).
      Done (commit 1ce093b): implemented as `met<…>` (per follow-up: reuse the `met`
      token, result is still a MethodInterface). `met <header>` -> a one-method
      MethodInterface, e.g. `met<(v)>` == `meti { (_ v) }`; the `(v)` -> `(_ v)`
      type rule applies. A Shortcut flag on MethodInterfaceExpr preserves the
      `met<…>` form in String()/WriteCode. Parser (TestParseMethodInterface) + VM
      (TestVMMethodInterface) tests; gofmt/vet clean, `go test ./...` -> 0 failures.
- [ ] change typed ident and param parser to parse param type of method interface. add exaplained tests and docs for it
  of funcs/closures/methods/funcHeaders/properties etc.... examples: `func x(cb meti{(int)<float>} ) {...}`, `met x(iOrCb int|meti{(int)<float>}) {...}`,
      STAGE 1 (parser) done (commit a95881c): the type parser accepts `meti{…}` /
      `interface{…}` / `met<…>` structural literals wherever a type is read (typed
      idents, params, func-headers, unions); isTypeStart + parseType handle the
      literal forms; parser tests (TestParseTypeMethodInterface). Compiling such a
      type is NOT yet supported — nameSymbolsOfTypedIdent returns a clear error
      (was a nil-deref panic). REMAINING: stage 2 (lower a literal type to a
      constant type reference — needs a constant-scope symbol or a type-ref model
      change), stage 3 (runtime structural `implements` type-checking in the
      TypeAssertion/ParamsTypes machinery), stage 4 (docs/examples). Best done in a
      fresh session — it is a type-system feature.
- [x] change parser of `met<...>` to allow multiples headers `met<(int), (float)<str> [, ...]>`, when format,
    if has muliples headers, put it int new indented line without comma. parses allow multiples itens separated by new line without `,` (its optional, no required in this case).
      Done (commit 983d018): parseMetShortcut parses 1+ bracket-less headers
      between `<…>`, separated by commas or newlines (either optional, ExprLevel
      makes newlines skippable). WriteCode formats several headers one per indented
      line without commas (idempotent); a single header stays inline `met<(_ v)>`;
      String() keeps the compact comma form. Parser test extended; `go test ./...`
      -> 0 failures.
- [ ] check cmd/update-*-plugin to accept all language changes.
    update vscode plugin to allow single "run" and "debug".
    create example page for codemirror and prismjs plugins.
