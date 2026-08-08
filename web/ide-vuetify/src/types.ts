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
 * run and (optionally) diagnose. `sourceType` selects the dialect — "" / "gad"
 * for plain Gad, "gadTemplate" for a `.gadt` mixed template, "gadx" for Gadx. Any
 * implementation works — a Go server, the Gad WASM module, etc. */
/** DocMode selects what the Doc panel generates and how it shows it (see the
 * React counterpart for the full list). Default "render-md". */
export type DocMode = "render-md" | "md" | "render-html" | "html" | "json" | "yaml";

/** DocResult is what a GadRunner.doc call returns; exactly one of markdown /
 * html / text is set, matching the requested mode. */
export interface DocResult {
  ok: boolean;
  mode: DocMode;
  markdown?: string;
  html?: string;
  text?: string;
  error?: string;
}

export interface GadRunner {
  name?: string;
  format(source: string, sourceType?: string): Promise<FormatResult>;
  /** Run source. For a gadx `sourceType`, `tagEncode` ("json"/"yaml") encodes the
   * returned tag as JSON/YAML instead of rendering it as HTML. */
  run(source: string, sourceType?: string, tagEncode?: string): Promise<RunResult>;
  diagnose?: (source: string, sourceType?: string) => Promise<GadDiagnostic[]> | GadDiagnostic[];
  /** Generate documentation for source in the given mode. When absent, the Doc
   * panel/toggle is hidden. */
  doc?: (source: string, sourceType: string, mode: DocMode) => Promise<DocResult>;
}
