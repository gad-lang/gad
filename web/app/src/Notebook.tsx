import { useCallback, useRef, useState } from "react";
import { Editor, type EditorHandle } from "./Editor";
import type { GadBackend, RunResult } from "./backends/types";

interface Cell {
  id: number;
  source: string;
  result: RunResult | null;
  running: boolean;
}

let nextId = 1;
function newCell(source = ""): Cell {
  return { id: nextId++, source, result: null, running: false };
}

const SAMPLE = [
  `total := 0\nfor i in [1, 2, 3, 4] {\n  total += i\n}\nprintln("sum =", total)\nreturn total`,
  `squares := [n * n for n in [1, 2, 3, 4, 5]]\nprintln(squares)\nreturn squares`,
];

/**
 * Notebook is the interactive-execution example: a column of independently
 * runnable cells, each showing its stdout/stderr/return value.
 */
export function Notebook({ backend }: { backend: GadBackend }) {
  const [cells, setCells] = useState<Cell[]>(() => SAMPLE.map((s) => newCell(s)));
  const handles = useRef(new Map<number, EditorHandle | null>());

  const runCell = useCallback(
    async (id: number) => {
      const source = handles.current.get(id)?.getValue() ?? "";
      setCells((cs) => cs.map((c) => (c.id === id ? { ...c, running: true } : c)));
      let result: RunResult;
      try {
        result = await backend.run(source);
      } catch (e) {
        result = { ok: false, stdout: "", stderr: String(e), result: "", diagnostics: [] };
      }
      setCells((cs) => cs.map((c) => (c.id === id ? { ...c, result, running: false } : c)));
    },
    [backend],
  );

  const addCell = () => setCells((cs) => [...cs, newCell()]);
  const removeCell = (id: number) =>
    setCells((cs) => (cs.length > 1 ? cs.filter((c) => c.id !== id) : cs));

  return (
    <div className="notebook">
      <p className="hint">
        Each cell runs independently via the <strong>{backend.name}</strong> backend.
      </p>
      {cells.map((cell) => (
        <div className="cell" key={cell.id}>
          <div className="cell-editor">
            <Editor
              ref={(h) => handles.current.set(cell.id, h)}
              initialDoc={cell.source}
              diagnose={backend.diagnose}
            />
          </div>
          <div className="cell-bar">
            <button onClick={() => runCell(cell.id)} disabled={cell.running}>
              {cell.running ? "Running…" : "Run ▶"}
            </button>
            <button className="ghost" onClick={() => removeCell(cell.id)}>
              Remove
            </button>
          </div>
          {cell.result && <CellOutput result={cell.result} />}
        </div>
      ))}
      <button className="add-cell" onClick={addCell}>
        + Add cell
      </button>
    </div>
  );
}

function CellOutput({ result }: { result: RunResult }) {
  return (
    <div className={`cell-output ${result.ok ? "" : "error"}`}>
      {result.stdout && <pre className="stdout">{result.stdout}</pre>}
      {result.stderr && <pre className="stderr">{result.stderr}</pre>}
      {result.ok && result.result && <div className="return">⇦ {result.result}</div>}
      {result.diagnostics.map((d, i) => (
        <div className="diag" key={i}>
          {d.line}:{d.column} {d.message}
        </div>
      ))}
    </div>
  );
}
