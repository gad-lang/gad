import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { EditorState, Extension, Compartment, StateEffect } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { defaultKeymap, indentWithTab, undo, redo } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { StreamLanguage } from "@codemirror/language";
import { basicSetup } from "codemirror";
import { gad, giom, type DiagnoseFn } from "@gad-lang/codemirror-gad";
import { json } from "@codemirror/lang-json";
import { html } from "@codemirror/lang-html";
import { css } from "@codemirror/lang-css";
import { javascript } from "@codemirror/lang-javascript";
import { markdown } from "@codemirror/lang-markdown";
import { yaml as yamlMode } from "@codemirror/legacy-modes/mode/yaml";
import { breakpointGutter, getBreakpointLines, setEditorBreakpoints } from "./breakpointGutter";
import { debugDecorations, setDebugLoc, type LocalVar } from "./debugDecorations";

export type EditorLanguage =
  | "gad"
  | "gadt"
  | "giom"
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

/** Template configuration for the `.gadt` / mixed `.gad` language: custom tag
 * delimiters, and `preamble` when the source is a `.gad` file that switches to
 * template mode part-way in via a `# gad: mixed` directive. */
export interface TemplateDelimiters {
  start?: string;
  end?: string;
  preamble?: boolean;
}

/** Return the CodeMirror Extension for the given language. */
function langExtension(lang: EditorLanguage, diagnose?: DiagnoseFn, tmpl?: TemplateDelimiters): Extension {
  switch (lang) {
    case "gad":
      return gad({ diagnose });
    case "gadt":
      return gad({ template: true, delimiters: { start: tmpl?.start, end: tmpl?.end }, preamble: tmpl?.preamble });
    case "giom":
      return giom({ diagnose });
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

export interface EditorHandle {
  getValue(): string;
  setValue(value: string): void;
  /** Move the cursor to a 1-based line/column, scroll to it and focus. */
  gotoLocation(line: number, column: number): void;
  /** Undo the last edit (history). */
  undo(): void;
  /** Redo the last undone edit (history). */
  redo(): void;
}

interface EditorProps {
  initialDoc: string;
  diagnose?: DiagnoseFn;
  /** Syntax language — defaults to "gad". */
  language?: EditorLanguage;
  /** Custom `.gadt` template delimiters (used when language is "gadt"). */
  templateDelimiters?: TemplateDelimiters;
  dark?: boolean;
  onChange?: (value: string) => void;
  /** Initial breakpoint lines (1-based) and a callback when they change. */
  breakpoints?: number[];
  onBreakpointsChange?: (lines: number[]) => void;
  /** Editor font size in pixels. */
  fontSize?: number;
  /** Current paused debug location (1-based line/column) and locals for tooltips. */
  debugLine?: number;
  debugColumn?: number;
  locals?: LocalVar[];
  /** Called when the user clicks "inspect" on a locals hover tooltip. */
  onInspectVar?: (name: string) => void;
}

/** Editor theme for the active light/dark mode. Template delimiters are tagged
 * `processingInstruction`, so the active theme's highlight style colours them —
 * no editor-level override is needed. */
function themeExtension(dark: boolean): Extension {
  return dark ? oneDark : [];
}

/**
 * Editor is a thin React wrapper around a CodeMirror 6 EditorView wired with the
 * Gad language plugin (highlighting + autocomplete + async diagnostics). The
 * diagnose source can change (e.g. switching backend) without recreating the
 * view, via a Compartment reconfigure.
 */
export const Editor = forwardRef<EditorHandle, EditorProps>(function Editor(
  {
    initialDoc,
    diagnose,
    language = "gad",
    templateDelimiters,
    dark = false,
    onChange,
    breakpoints,
    onBreakpointsChange,
    fontSize = 14,
    debugLine,
    debugColumn,
    locals,
    onInspectVar,
  },
  ref,
) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const langCompartment = useRef(new Compartment());
  const themeCompartment = useRef(new Compartment());
  const fontCompartment = useRef(new Compartment());
  // Keep latest diagnose available for the lang reconfigure effect.
  const diagnoseRef = useRef(diagnose);
  diagnoseRef.current = diagnose;
  // Keep latest template delimiters for the lang reconfigure effect.
  const tmplRef = useRef(templateDelimiters);
  tmplRef.current = templateDelimiters;
  // Keep the latest breakpoint callback so the gutter (created once) can call it.
  const onBpRef = useRef(onBreakpointsChange);
  onBpRef.current = onBreakpointsChange;
  // Latest locals, read lazily by the hover tooltip.
  const localsRef = useRef<Map<string, LocalVar>>(new Map());
  localsRef.current = new Map((locals ?? []).map((v) => [v.name, v]));
  // Latest inspect callback, read lazily so the extension stays stable.
  const onInspectVarRef = useRef(onInspectVar);
  onInspectVarRef.current = onInspectVar;

  const fontTheme = (px: number) =>
    EditorView.theme({ ".cm-scroller": { fontSize: `${px}px` }, ".cm-content": { fontSize: `${px}px` } });

  useImperativeHandle(ref, () => ({
    getValue: () => view.current?.state.doc.toString() ?? "",
    setValue: (value: string) => {
      const v = view.current;
      if (!v) return;
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } });
    },
    gotoLocation: (line: number, column: number) => {
      const v = view.current;
      if (!v) return;
      const ln = Math.min(Math.max(line, 1), v.state.doc.lines);
      const lineObj = v.state.doc.line(ln);
      const pos = lineObj.from + Math.min(Math.max(column - 1, 0), lineObj.length);
      v.dispatch({ selection: { anchor: pos }, scrollIntoView: true });
      v.focus();
    },
    undo: () => {
      const v = view.current;
      if (v) {
        undo(v);
        v.focus();
      }
    },
    redo: () => {
      const v = view.current;
      if (v) {
        redo(v);
        v.focus();
      }
    },
  }));

  // Create the view once.
  useEffect(() => {
    if (!host.current) return;
    const extensions: Extension[] = [
      basicSetup,
      breakpointGutter((lines) => onBpRef.current?.(lines)),
      debugDecorations(() => localsRef.current, () => onInspectVarRef.current),
      fontCompartment.current.of(fontTheme(fontSize)),
      keymap.of([...defaultKeymap, indentWithTab]),
      themeCompartment.current.of(themeExtension(dark)),
      langCompartment.current.of(langExtension(language, diagnose, templateDelimiters)),
      EditorView.updateListener.of((u) => {
        if (u.docChanged && onChange) onChange(u.state.doc.toString());
      }),
      EditorView.theme({ "&": { height: "100%" }, ".cm-scroller": { fontFamily: "monospace" } }),
    ];
    const v = new EditorView({
      state: EditorState.create({ doc: initialDoc, extensions }),
      parent: host.current,
    });
    view.current = v;
    if (breakpoints && breakpoints.length) setEditorBreakpoints(v, breakpoints);
    return () => v.destroy();
    // Intentionally run once; doc/language/diagnose updates are handled below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reconfigure the language extension when the language, diagnose backend or
  // template delimiters change.
  useEffect(() => {
    view.current?.dispatch({
      effects: langCompartment.current.reconfigure(langExtension(language, diagnoseRef.current, tmplRef.current)),
    });
  }, [language, diagnose, templateDelimiters?.start, templateDelimiters?.end, templateDelimiters?.preamble]);

  // Switch the editor theme when the light/dark mode changes.
  useEffect(() => {
    view.current?.dispatch({
      effects: themeCompartment.current.reconfigure(themeExtension(dark)),
    });
  }, [dark]);

  // Reconcile when breakpoints are changed externally (e.g. via the panel).
  useEffect(() => {
    const v = view.current;
    if (!v) return;
    const next = breakpoints ?? [];
    if (getBreakpointLines(v).join(",") !== next.join(",")) setEditorBreakpoints(v, next);
  }, [breakpoints]);

  // Apply the editor font size.
  useEffect(() => {
    view.current?.dispatch({ effects: fontCompartment.current.reconfigure(fontTheme(fontSize)) });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fontSize]);

  // Highlight (and scroll to) the current paused debug location.
  useEffect(() => {
    const v = view.current;
    if (!v) return;
    const loc = debugLine && debugLine > 0 ? { line: debugLine, col: debugColumn ?? 1 } : null;
    const effects: StateEffect<unknown>[] = [setDebugLoc.of(loc)];
    if (loc && loc.line <= v.state.doc.lines) {
      effects.push(EditorView.scrollIntoView(v.state.doc.line(loc.line).from, { y: "center" }));
    }
    v.dispatch({ effects });
  }, [debugLine, debugColumn]);

  return <div className="editor" ref={host} />;
});
