# @gad-lang/codemirror-gad

CodeMirror 6 language support for the [Gad](https://github.com/gad-lang/gad)
scripting language: syntax highlighting, autocompletion and async diagnostics.

> **Status:** not yet published to npm. The install command below is the intended
> usage once the `@gad-lang` packages are released.

## Install

```sh
npm install @gad-lang/codemirror-gad codemirror
# or: bun add @gad-lang/codemirror-gad codemirror
```

`codemirror` (and its `@codemirror/*` modules) is a peer dependency.

## Usage

```ts
import { basicSetup } from "codemirror";
import { gad } from "@gad-lang/codemirror-gad";

new EditorView({
  extensions: [
    basicSetup,
    gad({
      // optional: async source of { line, column, message, severity }
      diagnose: async (source) => fetchDiagnostics(source),
    }),
  ],
  parent: document.body,
});
```

## Source type (`gad` / `template` / `gadx`)

The `sourceType` option selects the dialect to highlight:

- `"gad"` (default) — a plain `.gad` script.
- `"template"` — a `.gadt` **mixed** template: literal text plus `{% … %}` code
  blocks and `{%= … %}` value tags, with the tag bodies tokenized as Gad
  (completion and hover work inside tags too). The delimiters default to
  `{%` / `%}` and are configurable via `delimiters`.
- `"gadx"` — a `.gadx` indentation-based template (tags, `@`-control keywords,
  `+`component calls, `{= … }` interpolations and `~~` code blocks), with the
  embedded Gad highlighted, completed and linted.

```ts
import { gad } from "@gad-lang/codemirror-gad";

gad();                                              // .gad (default)
gad({ sourceType: "template", delimiters: { start: "{%", end: "%}" } }); // .gadt
gad({ sourceType: "gadx" });                        // .gadx
```

A `.gad` file can also enable template mode part-way in with a `# gad: mixed`
directive (after an optional Gad preamble). For that case add `preamble: true`,
so the leading Gad — comments and the `# gad:` directive — is highlighted as Gad
before template text begins:

```ts
gad({ sourceType: "template", preamble: true, delimiters }); // `.gad` + `# gad: mixed`
```

> **Migration:** the former boolean `template: true` is replaced by
> `sourceType: "template"`. `gadx(options)` remains as a convenience for
> `gad({ ...options, sourceType: "gadx" })`.

## Exports

- `gad(options)` — bundled extension (language + completion + optional linter).
  Set `sourceType: "template"` (plus optional `delimiters: { start, end }`) for
  `.gadt` mixed files or `sourceType: "gadx"` for `.gadx`; the linter is skipped
  for `"template"`.
- `gadx(options)` — convenience for `gad({ ...options, sourceType: "gadx" })`.
- `gadLanguageSupport()` / `gadLanguage` — highlighting only.
- `gadCompletion()` / `gadCompletionSource` — autocompletion.
- `gadLinter(diagnose, { delay })` — async diagnostics → CodeMirror lint.
- `keywords`, `builtins`, `atoms`, `constants` — the word lists.

The `diagnose` function is injected, so the plugin works against any backend
(a Go HTTP server, the Gad WebAssembly module, etc.). See the example app in
[`../app`](../) and the overview in [`../README.md`](../README.md).

## Demo

A standalone editor demo lives in [`example/`](example). Its sidebar is a tree
of the repository `samples/` directory built from the filesystem at startup;
clicking a `.gad` / `.gadt` / `.gadx` file opens it with the matching
`sourceType`. The dev server (`example/serve.ts`) reads the manifest and each
file's contents from disk on demand — nothing is bundled in.

```sh
bun install
bun run demo        # bun ./example/serve.ts — bundles the app and serves samples/
```

## Publishing

The package is published to npm under the public `@gad-lang` scope. It ships the
compiled output in `dist/` (built from `src/` by `tsc`); `prepublishOnly` rebuilds
it, and `files`/`exports` point npm consumers at `dist/index.js` + `dist/index.d.ts`.

```sh
bun install
bun run build            # emit dist/ (tsc: .js + .d.ts)
npm version <patch|minor|major>
bun publish --dry-run    # inspect the tarball first
bun publish              # publishConfig sets the public registry + access
```

`publishConfig` (in `package.json`) pins the public npm registry and
`access: public`, so no per-package `.npmrc` is required. The auth token is read
from the environment or your global `~/.npmrc`; **never commit a token** (this
repo's `.gitignore` ignores dotfiles). For CI, drop in a local `.npmrc`:

```ini
//registry.npmjs.org/:_authToken=${NPM_TOKEN}
```

## Documentation

- [API reference](./docs/api.md) — the `gad()` options, source types and exports.
