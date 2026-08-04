// @gad-lang/prism-gad/gadx — PrismJS grammar for the Gadx template language.
//
//   import Prism from "prismjs";
//   import { registerGad } from "@gad-lang/prism-gad";
//   import { registerGadx } from "@gad-lang/prism-gad/gadx";
//   registerGad(Prism); // gadx embeds Gad in interpolations and `~~` blocks
//   registerGadx(Prism);
//   const html = Prism.highlight(code, Prism.languages.gadx, "gadx");

import type { Grammar, GrammarValue } from "prismjs";
import { gadGrammar } from "./gad-grammar";

/**
 * The Gadx PrismJS grammar. Gadx is indentation-based: a line may be a comment,
 * a `!!!` doctype, a `~~ … ~~` Gad code block, a `@`-control keyword, a
 * `+`component call, or a tag (`div.class#id[attr=…] text`). `{= … }` / `{ … }`
 * interpolations and the bodies of `~~` blocks and `[ … ]` attribute groups are
 * highlighted as embedded Gad by reusing the Gad grammar.
 */
export const gadxGrammar: Grammar = {
  comment: [
    {
      // `/* … */` block comments and `/** … **/` doc comments (gad convention),
      // recognized only at line start; may span multiple lines.
      pattern: /(^[ \t]*)\/\*[\s\S]*?\*\//m,
      lookbehind: true,
      greedy: true,
    },
    {
      // `//` and silent `//-` line comments.
      pattern: /\/\/.*/,
      greedy: true,
    },
  ],
  "code-block": {
    // `~~` … `~~` Gad source sections; the body is highlighted as Gad.
    pattern: /^[ \t]*~~[ \t]*$[\s\S]*?^[ \t]*~~[ \t]*$/m,
    greedy: true,
    inside: {
      "code-fence": { pattern: /^[ \t]*~~[ \t]*$/m, alias: "punctuation" },
      rest: gadGrammar,
    },
  },
  doctype: {
    pattern: /^[ \t]*!!!.*/m,
    alias: "important",
  },
  interpolation: {
    // `{= expr }` (buffered/escaped) and `{ expr }` (attribute/text).
    pattern: /\{=?[^{}]*\}/,
    greedy: true,
    inside: {
      "interpolation-punctuation": { pattern: /^\{=?|\}$/, alias: "punctuation" },
      rest: gadGrammar,
    },
  },
  keyword: {
    // Control keywords at line start: `@main`, `@if`, `@else`, `@for`, `@match`,
    // `@case`, `@var`, `@const`, `@enum`, `@global`, `@param`, `@func`, `@comp`,
    // `@slot`,
    // `@export`, `@assign`, `@import`, …
    pattern: /(^[ \t]*)@[A-Za-z_]\w*/m,
    lookbehind: true,
  },
  component: {
    // `+comp(...)` / `+mod.comp(...)` component calls.
    pattern: /(^[ \t]*)\+[A-Za-z_][\w.]*/m,
    lookbehind: true,
    alias: "function",
  },
  tag: {
    // The element name at the head of a tag line.
    pattern: /(^[ \t]*)[A-Za-z][\w-]*/m,
    lookbehind: true,
  },
  attributes: {
    // `[ … ]` attribute group; contents are Gad expressions.
    pattern: /\[[^\]]*\]/,
    greedy: true,
    inside: {
      punctuation: /[[\]]/,
      rest: gadGrammar,
    },
  },
  "attr-id": { pattern: /#[\w-]+/, alias: "selector" },
  "attr-class": { pattern: /\.[\w-]+/, alias: "selector" },
  string: gadGrammar.string as GrammarValue,
  number: gadGrammar.number as GrammarValue,
};

/** registerGadx installs the grammar under Prism.languages.gadx. Call
 * registerGad first so interpolations and `~~` blocks highlight embedded Gad. */
export function registerGadx(Prism: { languages: Record<string, Grammar> }): void {
  Prism.languages.gadx = gadxGrammar;
}
