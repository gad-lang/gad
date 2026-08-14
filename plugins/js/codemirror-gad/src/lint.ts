import { linter, Diagnostic as CMDiagnostic } from "@codemirror/lint";
import { EditorView } from "@codemirror/view";
import { Extension } from "@codemirror/state";
import type { GadSourceType } from "./index";

/** A positioned diagnostic as returned by the Gad backend (1-based line/col). */
export interface GadDiagnostic {
  line: number;
  column: number;
  message: string;
  severity: "error" | "warning";
}

/**
 * Async source of diagnostics for a Gad document. `sourceType` is the dialect
 * the document is edited as ("gad" | "gadx"), so the backend parses it with the
 * matching front-end instead of always as plain Gad (which would flag valid
 * `.gadx` syntax like `@comp` / `+comp(...)` as errors).
 */
export type DiagnoseFn = (
  source: string,
  sourceType?: GadSourceType,
) => Promise<GadDiagnostic[]> | GadDiagnostic[];

/**
 * Convert a 1-based line/column to an absolute document offset, clamping to
 * valid bounds so a stale or off-by-one position never throws.
 */
function offsetOf(view: EditorView, line: number, column: number): number {
  const doc = view.state.doc;
  const lineNo = Math.min(Math.max(line, 1), doc.lines);
  const l = doc.line(lineNo);
  const col = Math.min(Math.max(column - 1, 0), l.length);
  return l.from + col;
}

/**
 * gadLinter wires an async diagnose function into CodeMirror's lint system. The
 * diagnose function typically calls the HTTP server (/api/diagnose) or the WASM
 * module (gadDiagnose). Diagnostics are debounced by CodeMirror's linter.
 */
export function gadLinter(
  diagnose: DiagnoseFn,
  sourceType?: GadSourceType,
  config?: { delay?: number },
): Extension {
  return linter(
    async (view): Promise<CMDiagnostic[]> => {
      const source = view.state.doc.toString();
      let diags: GadDiagnostic[];
      try {
        diags = await diagnose(source, sourceType);
      } catch (e) {
        // Surface backend failures as a single document-level error.
        return [
          {
            from: 0,
            to: Math.min(1, view.state.doc.length),
            severity: "error",
            message: `diagnostics unavailable: ${String(e)}`,
          },
        ];
      }

      return diags.map((d): CMDiagnostic => {
        const from = offsetOf(view, d.line, d.column);
        const line = view.state.doc.lineAt(from);
        const to = Math.min(line.to, from + 1);
        return {
          from,
          to: to > from ? to : from,
          severity: d.severity === "warning" ? "warning" : "error",
          message: d.message,
        };
      });
    },
    { delay: config?.delay ?? 300 },
  );
}
