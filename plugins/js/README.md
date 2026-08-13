# Gad JS/TS editor libraries

Reusable JavaScript/TypeScript packages that add Gad language support to
in-browser editors. This directory is a **self-contained [bun](https://bun.sh)
workspace** — it compiles on its own, independently of the web app (`web/`) and
of the IDE extensions (`../ide`).

| Package | Name | What it is |
| --- | --- | --- |
| [`codemirror-gad`](codemirror-gad) | `@gad-lang/codemirror-gad` | CodeMirror 6 language support — highlighting, autocompletion and async diagnostics for `.gad` / `.gadt` / `.gadx`. |
| [`prism-gad`](prism-gad) | `@gad-lang/prism-gad` | PrismJS grammar for static, read-only Gad highlighting. |

Both packages are peer-dependency-only (CodeMirror / PrismJS are provided by the
host), so they add no runtime bloat to consumers.

## Build

```sh
cd plugins/js
bun install
bun run build       # tsc for both packages → their dist/
bun run typecheck   # type-check only, no emit
```

Each package can also be built on its own (`bun run --cwd codemirror-gad build`).

## How the web app consumes them

The web app and IDE components under `web/` list these packages as workspace
members (via `../plugins/js/*`), so `bun install` in `web/` links them into the
consumers and Vite/tsc resolve them from source. `make web-build` builds this
workspace first (`make plugins-js`) and then the app against the resulting
`dist/`.

## Keeping in sync with the language

The highlighting vocabulary (keywords, builtins, tokens) is regenerated from the
Gad compiler:

```sh
go run ./cmd/update-codemirror-plugin -w   # plugins/js/codemirror-gad
go run ./cmd/update-prism-plugin -w        # plugins/js/prism-gad
```

Run without `-w` for a dry run. See [`../README.md`](../README.md) for the full
plugin tree.

## Publish

```sh
bun run publish:dry   # dry-run for both packages
bun run publish       # publish @gad-lang/codemirror-gad and @gad-lang/prism-gad
```
