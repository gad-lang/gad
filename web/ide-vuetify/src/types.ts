import type { GadDiagnostic } from "@gad-lang/codemirror-gad";

/** FormatResult is the outcome of formatting (or transpiling) source. */
export interface FormatResult {
  ok: boolean;
  source: string;
  diagnostics: GadDiagnostic[];
}

/** RunResult is the outcome of compiling and executing source. */
export interface RunResult {
  ok: boolean;
  stdout: string;
  stderr: string;
  result: string;
  diagnostics: GadDiagnostic[];
}
