// A CodeMirror 6 breakpoint gutter for the Gad IDE. A single click on the
// gutter (next to the line numbers) sets a breakpoint on that line; a
// double-click removes it. Breakpoints are reported as 1-based line numbers.
import { RangeSet, StateEffect, StateField, type Extension } from "@codemirror/state";
import { EditorView, GutterMarker, gutter } from "@codemirror/view";

// setBreakpointsEffect replaces all breakpoints with the given line numbers
// (used to load/reconcile from external state such as the IDE config).
export const setBreakpointsEffect = StateEffect.define<number[]>();

// toggleEffect adds/removes a single breakpoint at a document position.
const toggleEffect = StateEffect.define<{ pos: number; on: boolean }>();

class BreakpointMarker extends GutterMarker {
  toDOM() {
    const s = document.createElement("span");
    s.className = "cm-bp-dot";
    s.textContent = "●";
    return s;
  }
}
const marker = new BreakpointMarker();

const breakpointState = StateField.define<RangeSet<GutterMarker>>({
  create() {
    return RangeSet.empty;
  },
  update(set, tr) {
    set = set.map(tr.changes);
    for (const e of tr.effects) {
      if (e.is(toggleEffect)) {
        set = e.value.on
          ? set.update({ add: [marker.range(e.value.pos)] })
          : set.update({ filter: (from) => from !== e.value.pos });
      } else if (e.is(setBreakpointsEffect)) {
        const doc = tr.state.doc;
        const marks = [...new Set(e.value)]
          .filter((l) => l >= 1 && l <= doc.lines)
          .sort((a, b) => a - b)
          .map((l) => marker.range(doc.line(l).from));
        set = RangeSet.of(marks);
      }
    }
    return set;
  },
});

/** getBreakpointLines returns the current breakpoints as sorted 1-based lines. */
export function getBreakpointLines(view: EditorView): number[] {
  const lines: number[] = [];
  const it = view.state.field(breakpointState).iter();
  while (it.value) {
    lines.push(view.state.doc.lineAt(it.from).number);
    it.next();
  }
  return lines.sort((a, b) => a - b);
}

/** setEditorBreakpoints replaces the editor's breakpoints with the given lines. */
export function setEditorBreakpoints(view: EditorView, lines: number[]) {
  view.dispatch({ effects: setBreakpointsEffect.of(lines) });
}

function hasBreakpoint(view: EditorView, pos: number): boolean {
  let found = false;
  view.state.field(breakpointState).between(pos, pos, () => {
    found = true;
  });
  return found;
}

/**
 * breakpointGutter builds the gutter extension. onChange is called with the new
 * line set whenever the user adds or removes a breakpoint.
 */
export function breakpointGutter(onChange: (lines: number[]) => void): Extension {
  const fire = (view: EditorView) => onChange(getBreakpointLines(view));
  return [
    breakpointState,
    gutter({
      class: "cm-breakpoint-gutter",
      markers: (v) => v.state.field(breakpointState),
      initialSpacer: () => marker,
      domEventHandlers: {
        // Single click adds a breakpoint; double click removes it.
        mousedown(view, line) {
          if (!hasBreakpoint(view, line.from)) {
            view.dispatch({ effects: toggleEffect.of({ pos: line.from, on: true }) });
            fire(view);
          }
          return true;
        },
        dblclick(view, line) {
          if (hasBreakpoint(view, line.from)) {
            view.dispatch({ effects: toggleEffect.of({ pos: line.from, on: false }) });
            fire(view);
          }
          return true;
        },
      },
    }),
    EditorView.baseTheme({
      ".cm-breakpoint-gutter": { width: "1.1em", cursor: "pointer" },
      ".cm-breakpoint-gutter .cm-bp-dot": { color: "#e5484d", paddingLeft: "2px" },
    }),
  ];
}
