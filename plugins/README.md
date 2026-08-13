# Gad editor plugins

Editor and syntax-highlighting integrations for the Gad language family
(`.gad` scripts, `.gadt` templates, `.gadx` templates). Each plugin is
self-contained and builds independently of the web app and of the other
plugins.

## Layout

```
plugins/
├── js/                     # standalone bun workspace for the JS/TS libraries
│   ├── codemirror-gad/     # @gad-lang/codemirror-gad — CodeMirror 6 support
│   └── prism-gad/          # @gad-lang/prism-gad — PrismJS grammar
└── ide/                    # editor / IDE extensions
    ├── vscode-gad/         # VS Code extension (TextMate grammars + client)
    └── intellij-gad/       # IntelliJ Platform plugin (planned — see its PLAN.md)
```

| Plugin | Package | What it is |
| --- | --- | --- |
| [`js/codemirror-gad`](js/codemirror-gad) | `@gad-lang/codemirror-gad` | CodeMirror 6 language support: highlighting, autocompletion and async diagnostics for `.gad` / `.gadt` / `.gadx`. |
| [`js/prism-gad`](js/prism-gad) | `@gad-lang/prism-gad` | PrismJS grammar for static, read-only Gad highlighting. |
| [`ide/vscode-gad`](ide/vscode-gad) | `vscode-gad` | VS Code extension: TextMate grammars (`syntaxes/*.tmLanguage.json`) plus the language client. |
| [`ide/intellij-gad`](ide/intellij-gad) | `Gad Language` | IntelliJ Platform plugin for JetBrains IDEs (highlighting, run configs, DAP debugger). Planned — see [`PLAN.md`](ide/intellij-gad/PLAN.md). |

## Building

### JS plugins (`plugins/js`)

A self-contained [bun](https://bun.sh) workspace — it compiles on its own,
without the `web/` app:

```sh
cd plugins/js
bun install
bun run build       # tsc for codemirror-gad and prism-gad → their dist/
bun run typecheck   # type-check only
```

The web app (`web/`) consumes these packages as workspace members, so a
`make web-build` builds the JS plugins first (via `make plugins-js`) and then the
app against their `dist/`.

### VS Code extension (`plugins/ide/vscode-gad`)

```sh
go run ./cmd/update-vscode-plugin -w   # regenerate the Gad TextMate grammar
cd plugins/ide/vscode-gad
bun install
bun run package                        # compile + produce vscode-gad.vsix
```

`make build-vscode-plugin` runs these steps and moves the `.vsix` into `dist/`.

### IntelliJ Platform plugin (`plugins/ide/intellij-gad`)

Planned — a Gradle/Kotlin plugin for JetBrains IDEs (IntelliJ IDEA, GoLand,
WebStorm, …) reusing the `gad debug --dap` adapter and the VS Code TextMate
grammars. Scope and roadmap live in
[`ide/intellij-gad/PLAN.md`](ide/intellij-gad/PLAN.md); once it builds, it will
compile with `./gradlew buildPlugin`.

## Keeping grammars in sync with the language

The plugins mirror the compiler's vocabulary (keywords, builtins, tokens). Three
Go tools regenerate the highlighting sources from the current language and report
the language commits since each plugin was last updated:

```sh
go run ./cmd/update-codemirror-plugin -w   # plugins/js/codemirror-gad
go run ./cmd/update-prism-plugin -w        # plugins/js/prism-gad
go run ./cmd/update-vscode-plugin -w       # plugins/ide/vscode-gad
```

Run them without `-w` for a dry run (report only).
