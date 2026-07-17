// @gad-lang/prism-gad/giom — PrismJS grammar for the Giom template language.
//
//   import Prism from "prismjs";
//   import { registerGad } from "@gad-lang/prism-gad";
//   import { registerGiom } from "@gad-lang/prism-gad/giom";
//   registerGad(Prism); // giom embeds Gad in interpolations and `~~` blocks
//   registerGiom(Prism);
//   const html = Prism.highlight(code, Prism.languages.giom, "giom");

import type { Grammar, GrammarValue } from "prismjs";
import { gadGrammar } from "./index";

/**
 * The Giom PrismJS grammar. Giom is indentation-based: a line may be a comment,
 * a `!!!` doctype, a `~~ … ~~` Gad code block, a `@`-control keyword, a
 * `+`component call, or a tag (`div.class#id[attr=…] text`). `{= … }` / `{ … }`
 * interpolations and the bodies of `~~` blocks and `[ … ]` attribute groups are
 * highlighted as embedded Gad by reusing the Gad grammar.
 */
export const giomGrammar: Grammar = {
  comment: {
    // `//` and silent `//-` line comments.
    pattern: /\/\/.*/,
    greedy: true,
  },
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
    // `@case`, `@var`, `@const`, `@enum`, `@global`, `@func`, `@comp`, `@slot`,
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

/** registerGiom installs the grammar under Prism.languages.giom. Call
 * registerGad first so interpolations and `~~` blocks highlight embedded Gad. */
export function registerGiom(Prism: { languages: Record<string, Grammar> }): void {
  Prism.languages.giom = giomGrammar;
}
