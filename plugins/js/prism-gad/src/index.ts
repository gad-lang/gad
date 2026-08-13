// @gad-lang/prism-gad — PrismJS grammar for the Gad scripting language.
//
//   import Prism from "prismjs";
//   import { registerGad } from "@gad-lang/prism-gad";
//   registerGad(Prism);
//   const html = Prism.highlight(code, Prism.languages.gad, "gad");

import type { Grammar, Environment } from "prismjs";
import { gadGrammar } from "./gad-grammar";
import { gadTemplateGrammar, type GadTemplateOptions } from "./template";
import { gadxGrammar } from "./gadx";


/** registerGad installs the grammar under Prism.languages.gad. */
export function registerGad(Prism: {
  languages: Record<string, Grammar>;
  hooks?: { add(name: string, cb: (env: Environment) => void): void };
}): void {
  Prism.languages.gad = gadGrammar;
}

/**
 * Which Gad dialect a grammar highlights:
 * - `"gad"` (default): a plain `.gad` script.
 * - `"template"`: a `.gadt` mixed template (`{% … %}` / `{%= … %}` tags).
 * - `"gadx"`: a `.gadx` indentation-based template.
 */
export type GadSourceType = "gad" | "template" | "gadx";

/**
 * gadGrammarFor returns the Prism grammar for the given source dialect, so a
 * consumer can pick the right handler from a single `sourceType` (the analog of
 * codemirror-gad's `gad({ sourceType })`). `options` (delimiters / `preamble`)
 * apply only to `"template"`.
 *
 * ```ts
 * const grammar = gadGrammarFor(sourceType);
 * const html = Prism.highlight(code, grammar, sourceType);
 * ```
 */
export function gadGrammarFor(
  sourceType: GadSourceType = "gad",
  options: GadTemplateOptions = {},
): Grammar {
  switch (sourceType) {
    case "template":
      return gadTemplateGrammar(options);
    case "gadx":
      return gadxGrammar;
    default:
      return gadGrammar;
  }
}

// Re-export the core Gad grammar (its definition moved to ./gad-grammar to break
// the template/gadx ↔ index import cycle).
export { gadGrammar } from "./gad-grammar";

export {
  gadTemplateGrammar,
  registerGadTemplate,
  detectGadTemplate,
  type GadTemplateDelimiters,
  type GadTemplateOptions,
  type GadTemplateDetection,
} from "./template";

export { gadxGrammar, registerGadx } from "./gadx";
