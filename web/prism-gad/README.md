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

For an interactive editor with autocompletion and live diagnostics, use
[`@gad-lang/codemirror-gad`](../codemirror-gad) instead. See the example app in
[`../README.md`](../README.md).
