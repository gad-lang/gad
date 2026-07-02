// @gad-lang/codemirror-gad — CodeMirror 6 support for the Gad language.
//
// Combine highlighting, autocompletion and async diagnostics with the `gad()`
// helper, or import the individual pieces.

import { LanguageSupport } from "@codemirror/language";
import { Extension } from "@codemirror/state";
import { gadCompletion } from "./complete";
import { gadLanguageSupport } from "./language";
import { GadTemplateDelimiters, gadTemplateLanguage } from "./template";
import { DiagnoseFn, gadLinter } from "./lint";
import { gadHoverTooltip } from "./hover";

export { gadLanguage, gadLanguageSupport } from "./language";
export { gadCompletion, gadCompletionSource } from "./complete";
export { gadLinter } from "./lint";
export type { GadDiagnostic, DiagnoseFn } from "./lint";
export type { GadTemplateDelimiters } from "./template";
export { keywords, builtins, atoms, constants } from "./keywords";
export { gadHoverTooltip } from "./hover";

export interface GadOptions {
  /** Enable autocompletion (default true). */
  completion?: boolean;
  /** Enable hover tooltips for builtins (default true). */
  hover?: boolean;
  /**
   * Async diagnostics source. When provided, a linter is installed that calls
   * it (e.g. the HTTP server or the WASM module). When omitted, no linting is
   * configured. Ignored in template mode (template text is not valid Gad).
   */
  diagnose?: DiagnoseFn;
  /** Lint debounce in ms (default 300). */
  lintDelay?: number;
  /**
   * Template (mixed) mode: highlight the source as a `.gadt` template — literal
   * text plus `{% … %}` / `{%= … %}` tags whose bodies are tokenized as Gad.
   */
  template?: boolean;
  /** Custom template tag delimiters (default `{%` / `%}`); only used when
   * `template` is set. */
  delimiters?: GadTemplateDelimiters;
  /** Start in a Gad preamble (for a `.gad` file whose `# gad: mixed` directive
   * enables template mode part-way in) rather than as template text from the
   * first byte (a `.gadt` file). Only used when `template` is set. */
  preamble?: boolean;
}

/**
 * gad returns a bundled extension: the language (highlighting), optional
 * autocompletion, optional hover tooltips for builtins, and an optional async
 * linter. Set `template` for `.gadt` (mixed) files, with `delimiters` to change
 * the `{%` / `%}` tags. Autocompletion and hover work inside tags too; the
 * linter is skipped in template mode.
 */
export function gad(options: GadOptions = {}): Extension {
  const ext: Extension[] = [
    options.template
      ? new LanguageSupport(gadTemplateLanguage(options.delimiters, options.preamble))
      : gadLanguageSupport(),
  ];
  if (options.completion !== false) ext.push(gadCompletion());
  if (options.hover !== false) ext.push(gadHoverTooltip());
  if (options.diagnose && !options.template) ext.push(gadLinter(options.diagnose, { delay: options.lintDelay }));
  return ext;
}
