// Shared result shapes for Gad operations, used across the IDE api and backends.
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";

export interface FormatResult {
  ok: boolean;
  source: string;
  diagnostics: GadDiagnostic[];
}

export interface RunResult {
  ok: boolean;
  stdout: string;
  stderr: string;
  result: string;
  diagnostics: GadDiagnostic[];
}

/** GadRunner is the minimal backend the Playground and Notebook drive: format,
 * run and (optionally) diagnose. `sourceType` selects the dialect — "" / "gad"
 * for plain Gad, "gadTemplate" for a `.gadt` mixed template, "gadx" for Gadx. Any
 * implementation works — a Go server, the Gad WASM module, etc. */
export interface GadRunner {
  name?: string;
  format(source: string, sourceType?: string): Promise<FormatResult>;
  run(source: string, sourceType?: string): Promise<RunResult>;
  diagnose?: (source: string, sourceType?: string) => Promise<GadDiagnostic[]> | GadDiagnostic[];
}
