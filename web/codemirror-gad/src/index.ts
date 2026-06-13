// @gad-lang/codemirror-gad — CodeMirror 6 support for the Gad language.
//
// Combine highlighting, autocompletion and async diagnostics with the `gad()`
// helper, or import the individual pieces.

import { Extension } from "@codemirror/state";
import { gadCompletion } from "./complete";
import { gadLanguageSupport } from "./language";
import { DiagnoseFn, gadLinter } from "./lint";

export { gadLanguage, gadLanguageSupport } from "./language";
export { gadCompletion, gadCompletionSource } from "./complete";
export { gadLinter } from "./lint";
export type { GadDiagnostic, DiagnoseFn } from "./lint";
export { keywords, builtins, atoms, constants } from "./keywords";

export interface GadOptions {
  /** Enable autocompletion (default true). */
  completion?: boolean;
  /**
   * Async diagnostics source. When provided, a linter is installed that calls
   * it (e.g. the HTTP server or the WASM module). When omitted, no linting is
   * configured.
   */
  diagnose?: DiagnoseFn;
  /** Lint debounce in ms (default 300). */
  lintDelay?: number;
}

/**
 * gad returns a bundled extension: the language (highlighting), optional
 * autocompletion and an optional async linter.
 */
export function gad(options: GadOptions = {}): Extension {
  const ext: Extension[] = [gadLanguageSupport()];
  if (options.completion !== false) ext.push(gadCompletion());
  if (options.diagnose) ext.push(gadLinter(options.diagnose, { delay: options.lintDelay }));
  return ext;
}
