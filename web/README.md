# Gad Web: CodeMirror plugin + example app

This directory contains a CodeMirror 6 plugin for the Gad language and an
example web app (React) that formats, lints and runs Gad source. The same
features are powered by two interchangeable backends: a **Go HTTP server** and
an **in-browser WebAssembly** module.

```
web/
├── gadbridge/        Go package: Format / Diagnose / Run (shared core)
├── server/           Go HTTP server  (/api/fmt, /api/run, /api/diagnose)
├── wasm/             Go WASM module  (gadFormat / gadRun / gadDiagnose globals)
├── codemirror-gad/   CodeMirror 6 plugin: highlight + autocomplete + linter
├── prism-gad/        PrismJS grammar for static (read-only) highlighting
└── app/              React + Vite app (Formatter + Notebook + Highlight)
```

The app supports a light/dark theme (toggle in the header; it follows the OS
preference by default and is persisted). The CodeMirror editor and the PrismJS
tokens both follow the active theme.

## The CodeMirror plugin (`@gad-lang/codemirror-gad`)

```ts
import { gad } from "@gad-lang/codemirror-gad";

const extensions = [
  basicSetup,
  gad({
    // async diagnostics: return [{ line, column, message, severity }]
    diagnose: async (src) => myBackend.diagnose(src),
  }),
];
```

`gad(options)` bundles:

- **Highlighting** — a stream tokenizer for comments, the string/heredoc/bytes
  forms, numbers, char literals, keywords and builtins.
- **Autocompletion** — Gad keywords, atoms, constants and builtins.
- **Linting** — turns an injected async `diagnose(source)` into CodeMirror
  diagnostics, mapping 1-based line/column to document offsets.

The diagnose/format/run functions are injected, so the plugin is independent of
how Gad is executed (server or WASM).

## Running the example app

Requires Go (for the bridge/server/WASM) and Node v26.3.0 with **pnpm**.

From the repo root:

```sh
make web          # install deps, build gad.wasm, run the Vite dev server
```

Open the printed URL. The **WebAssembly** backend works out of the box. To use
the **Go server** backend, run it in another terminal:

```sh
make web-server   # API on :8080; Vite proxies /api to it
```

Production build (outputs `web/app/dist`, which the Go server can serve):

```sh
make web-build
make web-server   # now serves the built app + API on :8080
```

`make build-wasm` (re)builds just the WASM module + `wasm_exec.js` into
`web/app/public`.

## Examples in the app

- **Formatter** — editor on the right with live diagnostics (underlined as you
  type); the left viewer shows the formatted source or run output. `Format`,
  `Format & apply` and `Run` use the selected backend.
- **Notebook** — independently runnable cells, each showing stdout/stderr and
  the return value, for interactive exploration.
- **Highlight** — static, read-only Gad snippets rendered with the
  `@gad-lang/prism-gad` PrismJS grammar (for docs and blog-style code blocks).

Switch between the WebAssembly and Go-server backends with the selector in the
header.
