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

/** GadRunner is the minimal backend the Playground and Notebook drive: format,
 * run and (optionally) diagnose. Any implementation works — a Go server, the Gad
 * WASM module, etc. */
export interface GadRunner {
  name?: string;
  format(source: string): Promise<FormatResult>;
  run(source: string): Promise<RunResult>;
  diagnose?: (source: string) => Promise<GadDiagnostic[]> | GadDiagnostic[];
}
