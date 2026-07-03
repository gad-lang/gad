# @gad-lang/prism-gad

A [PrismJS](https://prismjs.com/) grammar for the
[Gad](https://github.com/gad-lang/gad) scripting language — for static,
read-only syntax highlighting (docs, blogs, code blocks).

```ts
import Prism from "prismjs";
import { registerGad } from "@gad-lang/prism-gad";

registerGad(Prism);
const html = Prism.highlight(code, Prism.languages.gad, "gad");
```

It covers comments, the string/heredoc/bytes forms, `/regex/` literals,
keywords, atoms, builtins, `@`-prefixed specials, numbers and operators. Token
colors are supplied by your Prism theme (or your own `.token.*` CSS).

## Templates (`.gadt`)

`registerGadTemplate(Prism, delims?)` installs a `gadt` grammar for Gad template
(mixed) files — literal text plus `{% … %}` / `{%= … %}` tags whose bodies use
the embedded Gad grammar. Delimiters default to `{%` / `%}` and are
configurable. It reuses the Gad grammar, so `registerGad` must run first.

```ts
import { registerGad, registerGadTemplate } from "@gad-lang/prism-gad";

registerGad(Prism);
registerGadTemplate(Prism, { start: "{%", end: "%}" }); // delimiters optional
const html = Prism.highlight(src, Prism.languages.gadt, "gadt");
```

### Mixed `.gad` files (`# gad: mixed`)

A `.gad` file can enable template mode inline with a `# gad: mixed` directive
(after an optional Gad preamble). Use `detectGadTemplate(source)` to read the
directive — whether it enables `mixed` and any `delimiter=[START, END]` — and
`preamble: true` to highlight the leading Gad (comments + the `# gad:` line) as
Gad before the template text:

```ts
import { detectGadTemplate, registerGadTemplate } from "@gad-lang/prism-gad";

const { mixed, start, end } = detectGadTemplate(source);
if (mixed) {
  registerGadTemplate(Prism, { start, end, preamble: true });
  const html = Prism.highlight(source, Prism.languages.gadt, "gadt");
}
```

Prism is stateless, so the preamble is an anchored approximation (it applies at
the start of the source); for a full state machine use the CodeMirror plugin.

For an interactive editor with autocompletion and live diagnostics, use
[`@gad-lang/codemirror-gad`](../codemirror-gad) instead. See the example app in
[`../README.md`](../README.md).

## Demo

A standalone highlighting demo lives in [`example/`](example), with three tabs —
a plain `.gad` script, a `.gadt` template, and a `# gad: mixed` `.gad` file
(routed to the template grammar via `detectGadTemplate`):

```sh
bun install
bun run demo        # serves example/index.html
# or: bun run demo:build   # writes a static bundle to example/dist
```
