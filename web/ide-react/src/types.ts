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
