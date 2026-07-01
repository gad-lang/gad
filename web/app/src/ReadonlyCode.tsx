import { useEffect, useRef } from "react";
import { Compartment, EditorState, type Extension } from "@codemirror/state";
import { EditorView, keymap, lineNumbers } from "@codemirror/view";
import {
  codeFolding,
  defaultHighlightStyle,
  foldGutter,
  foldKeymap,
  syntaxHighlighting,
} from "@codemirror/language";
import { oneDark } from "@codemirror/theme-one-dark";
import { json } from "@codemirror/lang-json";
import { html } from "@codemirror/lang-html";
import { markdown } from "@codemirror/lang-markdown";

/** Supported read-only viewer languages. */
export type ReadonlyLanguage = "json" | "html" | "markdown";

function langExtension(lang: ReadonlyLanguage): Extension {
  switch (lang) {
    case "json":
      return json();
    case "html":
      return html();
    case "markdown":
      return markdown();
    default:
      return [];
  }
}

/**
 * ReadonlyCode is a minimal, non-editable CodeMirror 6 viewer used to render
 * run output (e.g. stdout as JSON) with syntax highlighting. Unlike Editor it
 * carries no history, breakpoints, autocomplete or diagnostics — it is a
 * read-only presentation surface whose document is replaced when `value`
 * changes and whose theme reconfigures on `dark`.
 */
export function ReadonlyCode({ value, language, dark }: { value: string; language: ReadonlyLanguage; dark?: boolean }) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const themeCompartment = useRef(new Compartment());

  // Create the view once.
  useEffect(() => {
    if (!host.current) return;
    const v = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          lineNumbers(),
          langExtension(language),
          syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
          // Expand/collapse for the language's block ranges (JSON objects/arrays,
          // HTML elements, …) via a fold gutter and the fold keymap.
          codeFolding(),
          foldGutter(),
          keymap.of(foldKeymap),
          themeCompartment.current.of(dark ? oneDark : []),
          EditorView.editable.of(false),
          EditorState.readOnly.of(true),
          EditorView.theme({ "&": { height: "100%" }, ".cm-scroller": { overflow: "auto" } }),
        ],
      }),
    });
    view.current = v;
    return () => {
      v.destroy();
      view.current = null;
    };
    // Recreate on language change; value/theme are handled by the effects below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [language]);

  // Replace the document when the value changes.
  useEffect(() => {
    const v = view.current;
    if (!v) return;
    if (v.state.doc.toString() !== value) {
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value } });
    }
  }, [value]);

  // Reconfigure the theme on light/dark toggle.
  useEffect(() => {
    view.current?.dispatch({ effects: themeCompartment.current.reconfigure(dark ? oneDark : []) });
  }, [dark]);

  return <div ref={host} className="readonly-code" style={{ height: "100%", overflow: "hidden" }} />;
}
