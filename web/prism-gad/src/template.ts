// @gad-lang/prism-gad/template — PrismJS grammar for Gad templates (`.gadt`).
//
//   import Prism from "prismjs";
//   import { registerGad } from "@gad-lang/prism-gad";
//   import { registerGadTemplate } from "@gad-lang/prism-gad/template";
//   registerGad(Prism);            // the embedded Gad grammar (required)
//   registerGadTemplate(Prism);    // installs Prism.languages.gadt
//   const html = Prism.highlight(src, Prism.languages.gadt, "gadt");
//
// A `.gadt` file is literal text with embedded Gad between delimiters:
//   {%  … %}  code block   {%= … %}  value expression
// Text outside the tags is left untouched. Delimiters default to `{%` / `%}`
// and can be overridden to match a project's `template:` config.
import type { Grammar } from "prismjs";
import { gadGrammar } from "./index";

/** Delimiters for the code/value tags. Defaults to `{%` / `%}`. */
export interface GadTemplateDelimiters {
  start?: string;
  end?: string;
}

/** Options for building the template grammar. */
export interface GadTemplateOptions extends GadTemplateDelimiters {
  /**
   * Highlight a leading **Gad preamble** for a `.gad` file that switches to
   * template mode part-way in with a `# gad: mixed` directive (as opposed to a
   * `.gadt` file, which is template from the first byte). Everything up to and
   * including the `# gad:` directive line is highlighted as Gad; the rest is
   * template. Because Prism is stateless, this is an anchored approximation and
   * only applies at the start of the source.
   */
  preamble?: boolean;
}

const DEFAULT_START = "{%";
const DEFAULT_END = "%}";

// The first `# gad: …` config directive line (see the parser's ParseConfigStmt).
const CONFIG_DIRECTIVE = /^[ \t]*#[ \t]*gad:[ \t]*([^\n]*)$/m;

// Escape a delimiter string for embedding into a RegExp source.
function esc(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * gadTemplateGrammar builds a PrismJS grammar for Gad templates using the given
 * delimiters (defaulting to `{%` / `%}`). Tag bodies are highlighted with the
 * embedded Gad grammar; the delimiters (and `=` value / `-`/`--` trim markers)
 * are tagged as punctuation. Everything else is literal text. When `preamble`
 * is set, a leading Gad region (up to the `# gad:` directive) is highlighted as
 * Gad — for a `.gad` file that enables mixed mode inline.
 */
export function gadTemplateGrammar(opts: GadTemplateOptions = {}): Grammar {
  const start = esc(opts.start || DEFAULT_START);
  const end = esc(opts.end || DEFAULT_END);
  return {
    // Anchored, lazy match up to and including the `# gad:` directive line,
    // highlighted with the full Gad grammar. Must come first so it wins at the
    // start of the source. Only present when a Gad preamble is expected.
    ...(opts.preamble
      ? {
          "gad-preamble": {
            pattern: /^[\s\S]*?#[ \t]*gad:[^\n]*/,
            inside: gadGrammar,
            alias: "gad",
          },
        }
      : {}),
    "gad-tag": {
      // Non-greedy body so the first end delimiter closes the tag.
      pattern: new RegExp(start + "[\\s\\S]*?" + end),
      greedy: true,
      inside: {
        "tag-delimiter": {
          pattern: new RegExp("^" + start + "[=-]{0,2}|-{0,2}" + end + "$"),
          alias: "punctuation",
        },
        // Highlight the tag body with the full Gad grammar.
        rest: gadGrammar,
      },
    },
  };
}

/**
 * registerGadTemplate installs the template grammar under Prism.languages.gadt.
 * Requires the Gad grammar (registerGad) to have been registered first, since
 * tag bodies reuse it. Pass `start`/`end` for custom delimiters and `preamble`
 * for a mixed `.gad` file.
 */
export function registerGadTemplate(
  Prism: { languages: Record<string, Grammar> },
  opts: GadTemplateOptions = {},
): void {
  Prism.languages.gadt = gadTemplateGrammar(opts);
}

/** The outcome of {@link detectGadTemplate}. */
export interface GadTemplateDetection {
  /** Whether a `# gad: mixed` directive enables template mode. */
  mixed: boolean;
  /** Custom start delimiter from `delimiter=[START, END]`, if any. */
  start?: string;
  /** Custom end delimiter from `delimiter=[START, END]`, if any. */
  end?: string;
}

/**
 * detectGadTemplate inspects a `.gad` source's first `# gad: …` config directive
 * (mirroring the parser's ParseConfigStmt): whether it enables `mixed` mode and
 * any `delimiter=[START, END]` override. Use it to pick and configure the right
 * grammar for a `.gad` file (a `.gadt` file is always a template).
 */
export function detectGadTemplate(source: string): GadTemplateDetection {
  const m = source.match(CONFIG_DIRECTIVE);
  if (!m) return { mixed: false };
  const body = m[1];
  const mixed = /\bmixed\b/.test(body) &&
    !/\bmixed[ \t]*=[ \t]*(?:false|no)\b/.test(body) &&
    !/\bno[_-]?mixed\b/.test(body);
  const d = body.match(/delimiter[ \t]*=[ \t]*\[[ \t]*(['"])(.*?)\1[ \t]*,[ \t]*(['"])(.*?)\3[ \t]*\]/);
  return { mixed, start: d?.[2], end: d?.[4] };
}
