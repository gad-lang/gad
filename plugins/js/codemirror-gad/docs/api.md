# @gad-lang/codemirror-gad — API reference

## `gad(options?)`

Returns a CodeMirror 6 `Extension` for the Gad language.

```ts
import { gad } from "@gad-lang/codemirror-gad";

gad(options?): Extension
```

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `sourceType` | `"gad" \| "template" \| "gadx"` | `"gad"` | Dialect to parse/highlight. |
| `diagnose` | `DiagnoseFn` | — | Async source of diagnostics (linting). |
| `delimiters` | `{ start?: string; end?: string }` | `{% %}` | Template tag delimiters (`sourceType: "template"`). |
| `preamble` | `boolean` | `false` | Highlight a leading Gad preamble before a `# gad: mixed` template. |

```ts
type DiagnoseFn = (source: string) => Promise<GadDiagnostic[]> | GadDiagnostic[];
interface GadDiagnostic { line: number; column: number; message: string; severity: "error" | "warning" }
```

Positions are 1-based (matching Gad's parser).

## Source types

- **`"gad"`** — a plain `.gad` script.
- **`"template"`** — a `.gadt` mixed template (literal text + `{% … %}` code and
  `{%= … %}` value tags; tag bodies are tokenized as Gad with completion/lint).
- **`"gadx"`** — a `.gadx` indentation template (tags, `@`-control keywords,
  `+`component calls, `{= … }` interpolations, `~~` code blocks) with the embedded
  Gad highlighted, completed and linted.

## Exports

- `gad(options?)` — the language extension.
- `type DiagnoseFn`, `type GadDiagnostic` — the diagnostics contract.
