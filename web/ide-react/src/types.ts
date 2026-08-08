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

/** DocMode selects what the Doc panel generates and how it shows it:
 * - "render-md":  the doc Markdown, rendered to HTML for reading (the default).
 * - "md":         the generated Markdown source (highlighted).
 * - "render-html": the doc HTML (goldmark), rendered.
 * - "html":       the generated HTML source (highlighted).
 * - "json"/"yaml": the doc encoded (prose, sections, snippets), shown as source. */
export type DocMode = "render-md" | "md" | "render-html" | "html" | "json" | "yaml";

/** DocResult is what a GadRunner.doc call returns. Exactly one of markdown / html
 * / text is set, matching the requested mode. */
export interface DocResult {
  ok: boolean;
  mode: DocMode;
  markdown?: string; // render-md, md
  html?: string; // render-html, html
  text?: string; // json, yaml
  error?: string;
}

/** GadRunner is the minimal backend the Playground and Notebook drive: format,
 * run and (optionally) diagnose. `sourceType` selects the dialect — "" / "gad"
 * for plain Gad, "gadTemplate" for a `.gadt` mixed template, "gadx" for Gadx. Any
 * implementation works — a Go server, the Gad WASM module, etc. */
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
