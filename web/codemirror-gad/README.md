# @gad-lang/codemirror-gad

CodeMirror 6 language support for the [Gad](https://github.com/gad-lang/gad)
scripting language: syntax highlighting, autocompletion and async diagnostics.

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

## Templates (`.gadt`)

Set `template: true` to highlight Gad **template** (mixed) files: literal text
plus `{% … %}` code blocks and `{%= … %}` value tags, with the tag bodies
tokenized as Gad (completion and hover work inside tags too). The delimiters
default to `{%` / `%}` and are configurable via `delimiters`.

```ts
import { gad } from "@gad-lang/codemirror-gad";

new EditorView({
  extensions: [basicSetup, gad({ template: true, delimiters: { start: "{%", end: "%}" } })],
  parent: document.body,
});
```

A `.gad` file can also enable template mode part-way in with a `# gad: mixed`
directive (after an optional Gad preamble). For that case add `preamble: true`,
so the leading Gad — comments and the `# gad:` directive — is highlighted as Gad
before template text begins:

```ts
gad({ template: true, preamble: true, delimiters }); // for `.gad` + `# gad: mixed`
```

## Exports

- `gad(options)` — bundled extension (language + completion + optional linter).
  Set `template: true` (plus optional `delimiters: { start, end }`) for `.gadt`
  mixed files; the linter is skipped in template mode.
- `gadLanguageSupport()` / `gadLanguage` — highlighting only.
- `gadCompletion()` / `gadCompletionSource` — autocompletion.
- `gadLinter(diagnose, { delay })` — async diagnostics → CodeMirror lint.
- `keywords`, `builtins`, `atoms`, `constants` — the word lists.

The `diagnose` function is injected, so the plugin works against any backend
(a Go HTTP server, the Gad WebAssembly module, etc.). See the example app in
[`../app`](../) and the overview in [`../README.md`](../README.md).
