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