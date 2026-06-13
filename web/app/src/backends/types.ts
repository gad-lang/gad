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

/**
 * GadBackend abstracts the source of Gad operations so the same UI works
 * against the Go HTTP server or the in-browser WebAssembly module.
 */
export interface GadBackend {
  readonly name: string;
  format(source: string): Promise<FormatResult>;
  run(source: string): Promise<RunResult>;
  diagnose(source: string): Promise<GadDiagnostic[]>;
}
