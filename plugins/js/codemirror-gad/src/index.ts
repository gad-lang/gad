// @gad-lang/codemirror-gad — CodeMirror 6 support for the Gad language.
//
// Combine highlighting, autocompletion and async diagnostics with the `gad()`
// helper, or import the individual pieces.

import { LanguageSupport } from "@codemirror/language";
import { Extension } from "@codemirror/state";
import { gadCompletion } from "./complete";
import { gadLanguageSupport } from "./language";
import { gadxLanguageSupport } from "./gadx";
import { GadTemplateDelimiters, gadTemplateLanguage } from "./template";
import { DiagnoseFn, gadLinter } from "./lint";
import { gadHoverTooltip } from "./hover";

export { gadLanguage, gadLanguageSupport } from "./language";
export { gadxLanguage, gadxLanguageSupport, gadxToken } from "./gadx";
export type { GadxState } from "./gadx";
export { gadCompletion, gadCompletionSource } from "./complete";
export { gadLinter } from "./lint";
export type { GadDiagnostic, DiagnoseFn } from "./lint";
export type { GadTemplateDelimiters } from "./template";
export { keywords, builtins, atoms, constants } from "./keywords";
export { gadHoverTooltip } from "./hover";

/**
 * Which Gad dialect to highlight:
 * - `"gad"` (default): a plain `.gad` script.
 * - `"template"`: a `.gadt` mixed template — literal text plus `{% … %}` /
 *   `{%= … %}` tags whose bodies are tokenized as Gad.
 * - `"gadx"`: a `.gadx` template — indentation-based tags, `@`-control keywords,
 *   `+`component calls and `{= … }` interpolations, with embedded Gad.
 */
export type GadSourceType = "gad" | "template" | "gadx";

export interface GadOptions {
  /** Enable autocompletion (default true). */
  completion?: boolean;
  /** Enable hover tooltips for builtins (default true). */
  hover?: boolean;
  /**
   * Async diagnostics source. When provided, a linter is installed that calls
   * it (e.g. the HTTP server or the WASM module). When omitted, no linting is
   * configured. Ignored for `sourceType: "template"` (template text is not valid
   * Gad); in `"gadx"` mode it applies to the embedded Gad code.
   */
  diagnose?: DiagnoseFn;
  /** Lint debounce in ms (default 300). */
  lintDelay?: number;
  /**
   * Source dialect, selecting the highlighter/handler: `"gad"` (default),
   * `"template"` or `"gadx"`. Replaces the former boolean `template` option.
   */
  sourceType?: GadSourceType;
  /** Custom template tag delimiters (default `{%` / `%}`); only used when
   * `sourceType` is `"template"`. */
  delimiters?: GadTemplateDelimiters;
  /** Start in a Gad preamble (for a `.gad` file whose `# gad: mixed` directive
   * enables template mode part-way in) rather than as template text from the
   * first byte (a `.gadt` file). Only used when `sourceType` is `"template"`. */
  preamble?: boolean;
}

/**
 * gad returns a bundled extension: the language (highlighting) for the requested
 * `sourceType`, optional autocompletion, optional hover tooltips for builtins,
 * and an optional async linter. Set `sourceType: "template"` for `.gadt` (mixed)
 * files — with `delimiters` to change the `{%` / `%}` tags — or `"gadx"` for
 * `.gadx` templates. Autocompletion and hover work inside tags/interpolations
 * too; the linter is skipped for `"template"`.
 */
export function gad(options: GadOptions = {}): Extension {
  const sourceType: GadSourceType = options.sourceType ?? "gad";

  let language: Extension;
  switch (sourceType) {
    case "template":
      language = new LanguageSupport(gadTemplateLanguage(options.delimiters, options.preamble));
      break;
    case "gadx":
      language = gadxLanguageSupport();
      break;
    default:
      language = gadLanguageSupport();
  }

  const ext: Extension[] = [language];
  if (options.completion !== false) ext.push(gadCompletion());
  if (options.hover !== false) ext.push(gadHoverTooltip());
  if (options.diagnose && sourceType !== "template") {
    ext.push(gadLinter(options.diagnose, sourceType, { delay: options.lintDelay }));
  }
  return ext;
}

/** Options for {@link gadx}; a subset of {@link GadOptions} (no template-only
 * `delimiters`/`preamble`). Equivalent to `gad({ ...options, sourceType: "gadx" })`. */
export type GadxOptions = Omit<GadOptions, "sourceType" | "delimiters" | "preamble">;

/**
 * gadx returns a bundled extension for `.gadx` templates. It is a convenience
 * wrapper for `gad({ ...options, sourceType: "gadx" })`: the Gadx language
 * (indentation-based tags, `.class`/`#id`, `[attr]` groups, `@`-control
 * keywords, `+`component calls and `{= … }` interpolations), with optional Gad
 * autocompletion, hover tooltips and async diagnostics that apply to the
 * embedded Gad code inside interpolations and `~~` blocks.
 */
export function gadx(options: GadxOptions = {}): Extension {
  return gad({ ...options, sourceType: "gadx" });
}
