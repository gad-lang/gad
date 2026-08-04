# @gad-lang/prism-gad — API reference

Register the grammar(s), then highlight with Prism.

## Functions

| Function | Description |
| --- | --- |
| `registerGad(Prism)` | Install `Prism.languages.gad`. Run this **first** — the template and gadx grammars embed it. |
| `gadGrammarFor(sourceType, options?)` | Return the grammar for a dialect: `"gad"` (default), `"template"` (`.gadt`) or `"gadx"` (`.gadx`). `options` (delimiters / `preamble`) apply only to `"template"`. |
| `registerGadTemplate(Prism, delims?)` | Install `Prism.languages.gadt` (mixed template). `delims`: `{ start?, end?, preamble? }`, default `{% %}`. |
| `registerGadx(Prism)` | Install `Prism.languages.gadx` (indentation template). |
| `detectGadTemplate(source)` | Read a `# gad: mixed` directive: `{ mixed, start, end }`. |

## Usage

```ts
import Prism from "prismjs";
import { registerGad, gadGrammarFor } from "@gad-lang/prism-gad";

registerGad(Prism);
const grammar = gadGrammarFor(sourceType);            // "gad" | "template" | "gadx"
const html = Prism.highlight(code, grammar, sourceType);
```

Token colors come from your Prism theme (or your own `.token.*` CSS). Prism is
stateless, so template/mixed highlighting is an anchored approximation; for a full
state machine (and an editable buffer) use
[`@gad-lang/codemirror-gad`](../../codemirror-gad).

## Coverage

Comments, string / heredoc / bytes forms, `/regex/` literals, keywords, atoms,
builtins, `@`-prefixed specials, numbers and operators.
