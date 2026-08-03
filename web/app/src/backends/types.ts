import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import type { FormatResult, RunResult } from "@gad-lang/ide-react";

export type { FormatResult, RunResult } from "@gad-lang/ide-react";

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
