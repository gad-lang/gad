# Gad editor plugins

Editor and syntax-highlighting integrations for the Gad language family
(`.gad` scripts, `.gadt` templates, `.gadx` templates).

Each plugin now lives in its **own repository** and is included here as a **git
submodule**, so clone with `--recurse-submodules` (or run
`git submodule update --init --recursive` after cloning). Every plugin has its
own VuetifyJS site (docs + live demo) published on GitHub Pages.

## Plugins

| Plugin | Repository | Site | What it is |
| --- | --- | --- | --- |
| `@gad-lang/codemirror-gad` | [gad-lang/codemirror-gad](https://github.com/gad-lang/codemirror-gad) | [site ↗](https://gad-lang.github.io/codemirror-gad/) | CodeMirror 6 language support: highlighting, autocompletion and async diagnostics for `.gad` / `.gadt` / `.gadx`. |
| `@gad-lang/prism-gad` | [gad-lang/prism-gad](https://github.com/gad-lang/prism-gad) | [site ↗](https://gad-lang.github.io/prism-gad/) | PrismJS grammar for static, read-only Gad highlighting. |
| `vscode-gad` | [gad-lang/vscode-gad](https://github.com/gad-lang/vscode-gad) | [site ↗](https://gad-lang.github.io/vscode-gad/) | VS Code extension: TextMate grammars plus the language client. |
| `intellij-gad` | [gad-lang/intellij-gad](https://github.com/gad-lang/intellij-gad) | [site ↗](https://gad-lang.github.io/intellij-gad/) | IntelliJ Platform plugin for JetBrains IDEs (highlighting, run configs, DAP debugger, Gad Doc panel). |

## Layout (submodules)

```
plugins/ide/
├── vscode-gad/     # submodule → gad-lang/vscode-gad
└── intellij-gad/   # submodule → gad-lang/intellij-gad
web/plugins/js/
├── codemirror-gad/ # submodule → gad-lang/codemirror-gad  (member of the web bun workspace)
└── prism-gad/      # submodule → gad-lang/prism-gad        (member of the web bun workspace)
```

The two JS plugins are members of the `web/` **bun** workspace, so a single
`bun install` in `web/` links them — no cross-directory symlink. `make plugins-js`
(run by `make web-build`) builds each to its `dist/`, which `web/` consumes.

## Building

Each repo carries its own `Makefile` (bun-only tooling). From a plugin's
directory: `make help`, `make build` (JS) / `make compile` (VS Code) /
`make build` (IntelliJ, Gradle). See each repo's `README.md` / `CLAUDE.md`.

`make build-vscode-plugin` (in this repo) regenerates the grammar and moves the
`.vsix` into `dist/`.

## Keeping grammars in sync with the language

The plugins mirror the compiler's vocabulary (keywords, builtins, tokens). Three
Go tools regenerate the highlighting sources from the current language, into the
submodule working trees, and report the language commits since each plugin was
last updated:

```sh
go run ./cmd/update-codemirror-plugin -w   # web/plugins/js/codemirror-gad
go run ./cmd/update-prism-plugin -w        # web/plugins/js/prism-gad
go run ./cmd/update-vscode-plugin -w       # plugins/ide/vscode-gad
```

Run them without `-w` for a dry run (report only). Commit and push the changes in
the submodule, then bump the submodule pointer in this repo.
