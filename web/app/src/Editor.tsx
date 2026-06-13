import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { EditorState, Extension, Compartment } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { defaultKeymap, indentWithTab } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { basicSetup } from "codemirror";
import { gad, type DiagnoseFn } from "@gad-lang/codemirror-gad";
import { breakpointGutter, getBreakpointLines, setEditorBreakpoints } from "./breakpointGutter";

export interface EditorHandle {
  getValue(): string;
  setValue(value: string): void;
}

interface EditorProps {
  initialDoc: string;
  diagnose?: DiagnoseFn;
  dark?: boolean;
  onChange?: (value: string) => void;
  /** Initial breakpoint lines (1-based) and a callback when they change. */
  breakpoints?: number[];
  onBreakpointsChange?: (lines: number[]) => void;
}

/** Editor theme for the active light/dark mode. */
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
  { initialDoc, diagnose, dark = false, onChange, breakpoints, onBreakpointsChange },
  ref,
) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const gadCompartment = useRef(new Compartment());
  const themeCompartment = useRef(new Compartment());
  // Keep the latest breakpoint callback so the gutter (created once) can call it.
  const onBpRef = useRef(onBreakpointsChange);
  onBpRef.current = onBreakpointsChange;

  useImperativeHandle(ref, () => ({
    getValue: () => view.current?.state.doc.toString() ?? "",
    setValue: (value: string) => {
      const v = view.current;
      if (!v) return;
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } });
    },
  }));

  // Create the view once.
  useEffect(() => {
    if (!host.current) return;
    const extensions: Extension[] = [
      basicSetup,
      breakpointGutter((lines) => onBpRef.current?.(lines)),
      keymap.of([...defaultKeymap, indentWithTab]),
      themeCompartment.current.of(themeExtension(dark)),
      gadCompartment.current.of(gad({ diagnose })),
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
    // Intentionally run once; doc/diagnose updates are handled below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Reconfigure the Gad extension when the diagnose backend changes.
  useEffect(() => {
    view.current?.dispatch({
      effects: gadCompartment.current.reconfigure(gad({ diagnose })),
    });
  }, [diagnose]);

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

  return <div className="editor" ref={host} />;
});
