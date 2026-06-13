import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { EditorState, Extension, Compartment } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { defaultKeymap, indentWithTab } from "@codemirror/commands";
import { basicSetup } from "codemirror";
import { gad, type DiagnoseFn } from "@gad-lang/codemirror-gad";

export interface EditorHandle {
  getValue(): string;
  setValue(value: string): void;
}

interface EditorProps {
  initialDoc: string;
  diagnose?: DiagnoseFn;
  onChange?: (value: string) => void;
}

/**
 * Editor is a thin React wrapper around a CodeMirror 6 EditorView wired with the
 * Gad language plugin (highlighting + autocomplete + async diagnostics). The
 * diagnose source can change (e.g. switching backend) without recreating the
 * view, via a Compartment reconfigure.
 */
export const Editor = forwardRef<EditorHandle, EditorProps>(function Editor(
  { initialDoc, diagnose, onChange },
  ref,
) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const gadCompartment = useRef(new Compartment());

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
      keymap.of([...defaultKeymap, indentWithTab]),
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

  return <div className="editor" ref={host} />;
});
