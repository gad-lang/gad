// Framework-agnostic CodeMirror 6 setup for the Gad IDE, shared by the Vue
// <GadEditor> wrapper. It builds the per-language extension and a factory that
// creates a fully-wired EditorView (Gad syntax + diagnostics, breakpoint gutter,
// debug decorations), plus small helpers to reconfigure it reactively.
import { Compartment, EditorState, type Extension } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { defaultKeymap, indentWithTab, redo as cmRedo, undo as cmUndo } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { StreamLanguage } from "@codemirror/language";
import { basicSetup } from "codemirror";
import { gad, type DiagnoseFn } from "@gad-lang/codemirror-gad";
import { json } from "@codemirror/lang-json";
import { html } from "@codemirror/lang-html";
import { css } from "@codemirror/lang-css";
import { javascript } from "@codemirror/lang-javascript";
import { markdown } from "@codemirror/lang-markdown";
import { yaml as yamlMode } from "@codemirror/legacy-modes/mode/yaml";
import { breakpointGutter, setEditorBreakpoints } from "./breakpointGutter";
import { debugDecorations, setDebugLoc, type LocalVar } from "./debugDecorations";

export type EditorLanguage =
  | "gad"
  | "gadt"
  | "gadx"
  | "json"
  | "yaml"
  | "html"
  | "css"
  | "scss"
  | "javascript"
  | "typescript"
  | "jsx"
  | "tsx"
  | "markdown"
  | "text";

/** Custom `.gadt` / mixed-mode template tag delimiters. */
export interface TemplateDelimiters {
  start?: string;
  end?: string;
  preamble?: boolean;
}

/** langOf maps a file extension to an EditorLanguage. */
export function langOf(path: string): EditorLanguage {
  const ext = path.slice(path.lastIndexOf(".") + 1).toLowerCase();
  switch (ext) {
    case "gad": return "gad";
    case "gadt": return "gadt";
    case "gadx": return "gadx";
    case "json": return "json";
    case "yaml": case "yml": return "yaml";
    case "html": case "htm": return "html";
    case "css": return "css";
    case "scss": return "scss";
    case "js": case "mjs": case "cjs": return "javascript";
    case "ts": case "mts": case "cts": return "typescript";
    case "jsx": return "jsx";
    case "tsx": return "tsx";
    case "md": case "mdx": case "markdown": return "markdown";
    default: return "text";
  }
}

/** Return the CodeMirror Extension for the given language. */
export function langExtension(lang: EditorLanguage, diagnose?: DiagnoseFn, tmpl?: TemplateDelimiters): Extension {
  switch (lang) {
    case "gad":
      return gad({ sourceType: "gad", diagnose });
    case "gadt":
      return gad({ sourceType: "template", delimiters: { start: tmpl?.start, end: tmpl?.end }, preamble: tmpl?.preamble });
    case "gadx":
      return gad({ sourceType: "gadx", diagnose });
    case "json":
      return json();
    case "yaml":
      return StreamLanguage.define(yamlMode);
    case "html":
      return html();
    case "css":
    case "scss":
      return css();
    case "javascript":
      return javascript();
    case "typescript":
      return javascript({ typescript: true });
    case "jsx":
      return javascript({ jsx: true });
    case "tsx":
      return javascript({ jsx: true, typescript: true });
    case "markdown":
      return markdown();
    default:
      return []; // plain text — no syntax
  }
}

export interface EditorOptions {
  parent: HTMLElement;
  doc: string;
  language: EditorLanguage;
  dark: boolean;
  diagnose?: DiagnoseFn;
  onChange?: (value: string) => void;
  onBreakpointsChange?: (lines: number[]) => void;
  onBreakpointContext?: (line: number) => void;
  getLocals?: () => Map<string, LocalVar>;
}

/** GadEditorView wraps an EditorView with compartment-based reconfiguration. */
export class GadEditorView {
  readonly view: EditorView;
  private langComp = new Compartment();
  private themeComp = new Compartment();
  private opts: EditorOptions;

  constructor(opts: EditorOptions) {
    this.opts = opts;
    const state = EditorState.create({
      doc: opts.doc,
      extensions: [
        basicSetup,
        keymap.of([indentWithTab, ...defaultKeymap]),
        this.langComp.of(langExtension(opts.language, opts.diagnose)),
        this.themeComp.of(opts.dark ? oneDark : []),
        breakpointGutter(
          (lines) => opts.onBreakpointsChange?.(lines),
          (line) => opts.onBreakpointContext?.(line),
        ),
        debugDecorations(() => opts.getLocals?.() ?? new Map()),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) opts.onChange?.(u.state.doc.toString());
        }),
      ],
    });
    this.view = new EditorView({ state, parent: opts.parent });
  }

  getValue(): string {
    return this.view.state.doc.toString();
  }

  setValue(value: string): void {
    this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: value } });
  }

  setLanguage(language: EditorLanguage, diagnose?: DiagnoseFn): void {
    this.opts.language = language;
    this.view.dispatch({ effects: this.langComp.reconfigure(langExtension(language, diagnose)) });
  }

  setDark(dark: boolean): void {
    this.view.dispatch({ effects: this.themeComp.reconfigure(dark ? oneDark : []) });
  }

  setBreakpoints(lines: number[]): void {
    setEditorBreakpoints(this.view, lines);
  }

  /** setDebugLine highlights the paused line (1-based), or clears it with 0. */
  setDebugLine(line: number, column = 1): void {
    this.view.dispatch({ effects: setDebugLoc.of(line >= 1 ? { line, col: column } : null) });
  }

  undo(): void {
    cmUndo(this.view);
    this.view.focus();
  }

  redo(): void {
    cmRedo(this.view);
    this.view.focus();
  }

  /** gotoLine moves the cursor to a 1-based line and scrolls it into view. */
  gotoLine(line: number, column = 1): void {
    const doc = this.view.state.doc;
    if (line < 1 || line > doc.lines) return;
    const l = doc.line(line);
    const pos = Math.min(l.from + Math.max(0, column - 1), l.to);
    this.view.dispatch({ selection: { anchor: pos }, scrollIntoView: true });
    this.view.focus();
  }

  destroy(): void {
    this.view.destroy();
  }
}

export type { LocalVar, DiagnoseFn };
