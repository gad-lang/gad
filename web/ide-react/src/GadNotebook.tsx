// GadNotebook — a column of independently runnable Gad cells, each with its own
// editor, dialect selector (GAD / GAD Template / GADx) and stdout/stderr/return
// output, executed through the injected GadRunner. The React counterpart of the
// Vuetify notebook, with a similar layout.
import { useRef, useState } from "react";
import type { DiagnoseFn } from "@gad-lang/codemirror-gad";
import { Editor, type EditorHandle, type EditorLanguage } from "./Editor";
import { PlaygroundStyles } from "./playgroundStyles";
import { DocPanel } from "./DocPanel";
import type { GadRunner, RunResult } from "./types";

type Dialect = "gad" | "gadt" | "gadx";
const LANG: Record<Dialect, EditorLanguage> = { gad: "gad", gadt: "gadt", gadx: "gadx" };
const SOURCE_TYPE: Record<Dialect, string> = { gad: "", gadt: "gadTemplate", gadx: "gadx" };

type TagEncode = "" | "json" | "yaml";

interface Cell {
  id: number;
  source: string;
  dialect: Dialect;
  tagEncode: TagEncode;
  result: RunResult | null;
  running: boolean;
}

let nextId = 1;
const newCell = (source = "", dialect: Dialect = "gad"): Cell => ({
  id: nextId++,
  source,
  dialect,
  tagEncode: "",
  result: null,
  running: false,
});

// One sample cell per source type (GAD / GAD Template / GADx), so the notebook
// opens with a runnable example of each dialect.
const SAMPLES: { dialect: Dialect; source: string }[] = [
  {
    dialect: "gad",
    source: `/**
# Squares

A documented module. Toggle the **Doc** button below to see its public API.
**/

/// squares holds n*n for each n in 1..5.
export squares = [n * n for n in [1, 2, 3, 4, 5]]
`,
  },
  {
    dialect: "gadt",
    source: `{%
/**
# Greeting Template

A mixed-mode template. Toggle **Doc** to see this comment rendered.
**/
var (name = "Gad", items = [1, 2, 3])
%}
<h1>Hello, {%= name %}!</h1>
<ul>
{% for i in items %}  <li>item {%= i %}</li>
{% end %}</ul>
`,
  },
  {
    dialect: "gadx",
    source: `/**
# Hello Gadx

An indentation-based HTML template; \`@main\` is its public entry.
**/
/** main renders the page. **/
@main
    h1 Hello Gadx
    ul
        @for i in [1, 2, 3]
            li item {= i }
`,
  },
];

export interface GadNotebookProps {
  runner: GadRunner;
  dark?: boolean;
}

/** GadNotebook renders the runnable-cells notebook. */
export function GadNotebook({ runner, dark = false }: GadNotebookProps) {
  const [cells, setCells] = useState<Cell[]>(() => SAMPLES.map((s) => newCell(s.source, s.dialect)));
  // Live editor contents per cell (editors are uncontrolled; read on run).
  const contents = useRef<Record<number, string>>({});
  // Per-cell editor handles, so the Doc panel can navigate the matching editor.
  const editors = useRef<Record<number, EditorHandle | null>>({});
  // Per-cell Doc panel visibility, and a revision bumped on edit to keep it synced.
  const [docCells, setDocCells] = useState<Record<number, boolean>>({});
  const [docRev, setDocRev] = useState<Record<number, number>>({});
  const bumpDoc = (id: number) => setDocRev((r) => ({ ...r, [id]: (r[id] ?? 0) + 1 }));

  const update = (id: number, patch: Partial<Cell>) =>
    setCells((cs) => cs.map((c) => (c.id === id ? { ...c, ...patch } : c)));

  async function runCell(cell: Cell) {
    update(cell.id, { running: true });
    const src = contents.current[cell.id] ?? cell.source;
    let result: RunResult;
    try {
      const te = cell.dialect === "gadx" ? cell.tagEncode : "";
      result = await runner.run(src, SOURCE_TYPE[cell.dialect], te);
    } catch (e) {
      result = { ok: false, stdout: "", stderr: String(e), result: "", diagnostics: [] };
    }
    update(cell.id, { running: false, result, source: src });
  }
  const addCell = () => setCells((cs) => [...cs, newCell()]);
  const removeCell = (id: number) => setCells((cs) => (cs.length > 1 ? cs.filter((c) => c.id !== id) : cs));
  const diagnoseFor = (cell: Cell): DiagnoseFn | undefined =>
    runner.diagnose ? (src: string) => runner.diagnose!(src, SOURCE_TYPE[cell.dialect]) : undefined;

  return (
    <div className="gnb">
      <PlaygroundStyles />
      <p className="gp-muted">Each cell runs independently.</p>
      {cells.map((cell) => (
        <div className="gnb-cell" key={cell.id}>
          <div className={"gnb-cell-main" + (docCells[cell.id] ? " gnb-cell-main--doc" : "")}>
            <div className="gnb-editor">
              <Editor
                key={cell.id + ":" + cell.dialect}
                ref={(h) => {
                  editors.current[cell.id] = h;
                }}
                initialDoc={cell.source}
                language={LANG[cell.dialect]}
                dark={dark}
                diagnose={diagnoseFor(cell)}
                onChange={(v) => {
                  contents.current[cell.id] = v;
                  if (docCells[cell.id]) bumpDoc(cell.id);
                }}
              />
            </div>
            {runner.doc && docCells[cell.id] && (
              <div className="gnb-doc">
                <DocPanel
                  doc={runner.doc}
                  source={() => contents.current[cell.id] ?? cell.source}
                  sourceType={SOURCE_TYPE[cell.dialect]}
                  dark={dark}
                  revision={docRev[cell.id] ?? 0}
                  onNavigate={(line, col) => editors.current[cell.id]?.gotoLocation(line, col)}
                  header={
                    <button
                      type="button"
                      className="gp-btn gp-doc-ctl"
                      title="Close the doc panel"
                      onClick={() => setDocCells((d) => ({ ...d, [cell.id]: false }))}
                    >
                      ✕
                    </button>
                  }
                />
              </div>
            )}
          </div>
          {/* toolbar */}
          <div className="gnb-bar">
            <span className="gp-dialect">
              {(["gad", "gadt", "gadx"] as Dialect[]).map((d) => (
                <button
                  key={d}
                  type="button"
                  className={"gp-tab" + (cell.dialect === d ? " gp-tab--active" : "")}
                  onClick={() => update(cell.id, { dialect: d, source: contents.current[cell.id] ?? cell.source })}
                >
                  {d === "gad" ? "GAD" : d === "gadt" ? "GAD Template" : "GADx"}
                </button>
              ))}
            </span>
            {cell.dialect === "gadx" && (
              <label className="gp-tagenc" title="GADX output: render HTML, or encode the tag as JSON/YAML">
                Tag:
                <select value={cell.tagEncode} onChange={(e) => update(cell.id, { tagEncode: e.target.value as TagEncode })}>
                  <option value="">Render</option>
                  <option value="json">JSON</option>
                  <option value="yaml">YAML</option>
                </select>
              </label>
            )}
            <button type="button" className="gp-btn gp-btn--primary" disabled={cell.running} onClick={() => runCell(cell)}>▶ Run</button>
            {runner.doc && (
              <button
                type="button"
                className={"gp-btn" + (docCells[cell.id] ? " gp-btn--active" : "")}
                aria-pressed={!!docCells[cell.id]}
                title="Toggle the documentation panel for this cell"
                onClick={() => {
                  setDocCells((d) => ({ ...d, [cell.id]: !d[cell.id] }));
                  bumpDoc(cell.id);
                }}
              >
                Doc
              </button>
            )}
            <button type="button" className="gp-btn" onClick={() => removeCell(cell.id)}>Remove</button>
          </div>
          {cell.result && (
            <div className={"gnb-out" + (cell.result.ok ? "" : " gp-error")}>
              {cell.result.stdout && <pre className="gp-out">{cell.result.stdout}</pre>}
              {cell.result.stderr && <pre className="gp-out gad-ide__diag">{cell.result.stderr}</pre>}
              {cell.result.ok && cell.result.result && <div className="gp-return">↩ {cell.result.result}</div>}
              {(cell.result.diagnostics ?? []).map((d, i) => (
                <div className="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
              ))}
            </div>
          )}
        </div>
      ))}
      <button type="button" className="gp-btn" onClick={addCell}>+ Add cell</button>
    </div>
  );
}
