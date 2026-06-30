// CodeMirror 6 debug decorations for the Gad IDE: while a debug session is
// paused, the current line is highlighted, the current node (the identifier at
// the stop position) is "super" highlighted, and hovering any identifier that
// is a current local shows its type and value in a tooltip.
import { StateEffect, StateField, type Extension, type Text } from "@codemirror/state";
import { Decoration, type DecorationSet, EditorView, hoverTooltip } from "@codemirror/view";

export interface LocalVar {
  name: string;
  type: string;
  value: string;
}

// setDebugLoc sets (or clears, with null) the current paused location.
export const setDebugLoc = StateEffect.define<{ line: number; col: number } | null>();

const lineDeco = Decoration.line({ class: "cm-debug-line" });
const nodeDeco = Decoration.mark({ class: "cm-debug-node" });

const isWordChar = (c: string) => /[A-Za-z0-9_]/.test(c);

/** wordAt returns the identifier range covering offset within line text, or null. */
function wordAt(text: string, offset: number): { start: number; end: number; word: string } | null {
  let i = Math.min(Math.max(offset, 0), text.length);
  if (i >= text.length || !isWordChar(text[i])) {
    if (i > 0 && isWordChar(text[i - 1])) i -= 1;
    else return null;
  }
  let s = i;
  let e = i;
  while (s > 0 && isWordChar(text[s - 1])) s--;
  while (e < text.length && isWordChar(text[e])) e++;
  if (e <= s) return null;
  return { start: s, end: e, word: text.slice(s, e) };
}

function buildDecorations(doc: Text, loc: { line: number; col: number } | null): DecorationSet {
  if (!loc || loc.line < 1 || loc.line > doc.lines) return Decoration.none;
  const line = doc.line(loc.line);
  const ranges = [lineDeco.range(line.from)];
  const w = wordAt(line.text, loc.col - 1);
  if (w) ranges.push(nodeDeco.range(line.from + w.start, line.from + w.end));
  return Decoration.set(ranges, true);
}

const debugDecoField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(deco, tr) {
    deco = deco.map(tr.changes);
    for (const e of tr.effects) if (e.is(setDebugLoc)) deco = buildDecorations(tr.state.doc, e.value);
    return deco;
  },
  provide: (f) => EditorView.decorations.from(f),
});

/**
 * debugDecorations builds the extension. getLocals supplies the current locals
 * (by name) for the hover tooltip; it is read lazily so it always reflects the
 * latest paused frame. onInspect, when provided, adds an inspect button that
 * opens the tree navigator for the hovered variable.
 */
export function debugDecorations(
  getLocals: () => Map<string, LocalVar>,
  getInspect?: () => ((name: string) => void) | undefined,
): Extension {
  return [
    debugDecoField,
    hoverTooltip((view, pos) => {
      const line = view.state.doc.lineAt(pos);
      const w = wordAt(line.text, pos - line.from);
      if (!w) return null;
      const v = getLocals().get(w.word);
      if (!v) return null;
      return {
        pos: line.from + w.start,
        end: line.from + w.end,
        above: true,
        create() {
          const dom = document.createElement("div");
          dom.className = "cm-locals-tooltip";
          const text = document.createElement("span");
          text.textContent = `${w.word}: ${v.type} = ${v.value}`;
          const copy = document.createElement("button");
          copy.className = "cm-locals-tooltip-copy";
          copy.title = "Copy value";
          copy.textContent = "⎘";
          copy.onclick = (e) => {
            e.stopPropagation();
            void navigator.clipboard?.writeText(v.value).catch(() => {});
          };
          dom.append(text, copy);
          const onInspect = getInspect?.();
          if (onInspect) {
            const insp = document.createElement("button");
            insp.className = "cm-locals-tooltip-inspect";
            insp.title = "Inspect (tree navigator)";
            insp.textContent = "⊕";
            insp.onclick = (e) => {
              e.stopPropagation();
              onInspect(w.word);
            };
            dom.append(insp);
          }
          return { dom };
        },
      };
    }),
    EditorView.baseTheme({
      ".cm-debug-line": { backgroundColor: "rgba(255, 200, 0, 0.18)" },
      ".cm-debug-node": {
        backgroundColor: "rgba(255, 160, 0, 0.45)",
        outline: "1px solid rgba(255, 140, 0, 0.9)",
        borderRadius: "2px",
      },
      ".cm-locals-tooltip": {
        display: "flex",
        alignItems: "center",
        gap: "6px",
        padding: "2px 6px",
        fontFamily: "ui-monospace, monospace",
        fontSize: "0.85em",
      },
      ".cm-locals-tooltip-copy, .cm-locals-tooltip-inspect": {
        background: "none",
        border: "none",
        cursor: "pointer",
        padding: "0 2px",
        fontSize: "1em",
        lineHeight: 1,
        opacity: "0.6",
        color: "inherit",
        "&:hover": { opacity: "1" },
      },
    }),
  ];
}
