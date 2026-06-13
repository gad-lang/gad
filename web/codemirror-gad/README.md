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

## Exports

- `gad(options)` — bundled extension (language + completion + optional linter).
- `gadLanguageSupport()` / `gadLanguage` — highlighting only.
- `gadCompletion()` / `gadCompletionSource` — autocompletion.
- `gadLinter(diagnose, { delay })` — async diagnostics → CodeMirror lint.
- `keywords`, `builtins`, `atoms`, `constants` — the word lists.

The `diagnose` function is injected, so the plugin works against any backend
(a Go HTTP server, the Gad WebAssembly module, etc.). See the example app in
[`../app`](../) and the overview in [`../README.md`](../README.md).
