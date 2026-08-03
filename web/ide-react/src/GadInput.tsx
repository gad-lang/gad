import { useEffect, useRef } from "react";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap, placeholder } from "@codemirror/view";
import { defaultKeymap } from "@codemirror/commands";
import { oneDark } from "@codemirror/theme-one-dark";
import { gad } from "@gad-lang/codemirror-gad";

interface GadInputProps {
  value: string;
  onChange: (value: string) => void;
  /** Called when the user presses Enter. */
  onSubmit?: () => void;
  dark?: boolean;
  placeholder?: string;
  className?: string;
}

/**
 * GadInput is a single-line CodeMirror editor pre-configured with the Gad
 * language plugin (syntax highlighting + autocomplete). Use it in place of a
 * plain <TextField> where a Gad expression is expected. Enter triggers onSubmit.
 */
export function GadInput({ value, onChange, onSubmit, dark, placeholder: hint, className }: GadInputProps) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const onSubmitRef = useRef(onSubmit);
  onSubmitRef.current = onSubmit;

  useEffect(() => {
    if (!host.current) return;
    const v = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [
          gad({}),
          dark ? oneDark : [],
          hint ? placeholder(hint) : [],
          EditorView.theme({
            "&": {
              border: "1px solid rgba(0,0,0,.23)",
              borderRadius: "4px",
              padding: "4px 8px",
              cursor: "text",
              minHeight: "36px",
              display: "flex",
              alignItems: "center",
            },
            ".cm-scroller": { overflow: "hidden" },
            ".cm-content": { fontFamily: "ui-monospace, monospace", fontSize: "0.875rem", padding: 0 },
            ".cm-line": { padding: 0 },
            "&.cm-editor.cm-focused": { outline: "2px solid #1976d2", outlineOffset: "-1px", borderColor: "#1976d2" },
            ".cm-placeholder": { color: "rgba(0,0,0,.4)", fontStyle: "italic" },
          }),
          keymap.of([
            { key: "Enter", run: () => { onSubmitRef.current?.(); return true; } },
            ...defaultKeymap,
          ]),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) onChangeRef.current(u.state.doc.toString());
          }),
        ],
      }),
      parent: host.current,
    });
    view.current = v;
    return () => v.destroy();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sync when value changes externally (e.g. loading a saved expression).
  useEffect(() => {
    const v = view.current;
    if (!v) return;
    const current = v.state.doc.toString();
    if (current !== value) v.dispatch({ changes: { from: 0, to: current.length, insert: value } });
  }, [value]);

  return <div ref={host} className={className} style={{ width: "100%" }} />;
}
