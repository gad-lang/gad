# IDE epic (`web/ide` backend + `web/app` React frontend)
- [~] change ui to use `dockview-react` package with dock/undock, move, resizable panels. 
  save all ui panel config (and positions and sizes) to config file.
  create button on top bar to reset panels config.
- [~] change ui to use plugin for edit `.md` files and render it in left tab like `DOCS`.

# web/js projects
- [ ] replace all runner of `pnpm` to `bun`, update docs and scripts.

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
- [ ] change `class` declaration (Expr and Stmt) from syntaxe `class [NAME] [extends ...] {` to `class [NAME] { extends { Parent [: Alias], ... } ... }` (extends itens separated by `,` or `\n`. 
      Parent `alias` is optional, separated by `:`. `Parent` is `IdentExpr` or `SelectorExpr`, example: `class { extends { mod1.A, mod2.A: A2 } }` (`A2` is alias of `mod2.A`) ).
      format `WriteCode` extends section itens to new indented line.
- [ ] rename module "core" to "gad". update samples, docs and tests.
- [ ] create builtin functions `gad.binOp{OP_NAME}` (for binary operators), `gad.unOp{OP_NAME}` (for unary operators),
      `gad.selfAssigOp{OP_NAME}` (for self assign operators) removing first param `op Operator`. use call to `gad.binOp{OP_NAME}` insteadof `gad.binOp`,
      apply this rule for `gad.unOp` and `gad.selfAssigOp{OP_NAME}`. update methods implementations in tests, samples, doc, README.