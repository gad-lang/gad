import { useMemo, useRef, useState } from "react";
import { Editor, type EditorHandle } from "./Editor";
import { Notebook } from "./Notebook";
import { Highlight } from "./Highlight";
import { Debug } from "./Debug";
import { useTheme } from "./useTheme";
import { serverBackend } from "./backends/server";
import { wasmBackend } from "./backends/wasm";
import type { GadBackend, FormatResult, RunResult } from "./backends/types";

const BACKENDS: Record<string, GadBackend> = {
  wasm: wasmBackend,
  server: serverBackend,
};

const SAMPLE = `// edit me — errors are underlined as you type
param *args

greet := func(name; greeting="Hello") {
  return greeting + ", " + name
}

squares := [n*n for n in [1,2,3,4] if n>1]
println(greet("Gad"), squares)
return squares
`;

type Tab = "format" | "notebook" | "debug" | "highlight";

export function App() {
  const [backendKey, setBackendKey] = useState<keyof typeof BACKENDS>("wasm");
  const [tab, setTab] = useState<Tab>("format");
  const [theme, toggleTheme] = useTheme();
  const backend = BACKENDS[backendKey];
  const dark = theme === "dark";

  return (
    <div className="app">
      <header>
        <h1>Gad Playground</h1>
        <div className="controls">
          <button className="theme-toggle" onClick={toggleTheme} title="Toggle light/dark">
            {dark ? "☀ Light" : "☾ Dark"}
          </button>
          <label>
            Backend:{" "}
            <select value={backendKey} onChange={(e) => setBackendKey(e.target.value)}>
              <option value="wasm">WebAssembly (in-browser)</option>
              <option value="server">Go server (/api)</option>
            </select>
          </label>
          <nav className="tabs">
            <button className={tab === "format" ? "on" : ""} onClick={() => setTab("format")}>
              Formatter
            </button>
            <button className={tab === "notebook" ? "on" : ""} onClick={() => setTab("notebook")}>
              Notebook
            </button>
            <button className={tab === "debug" ? "on" : ""} onClick={() => setTab("debug")}>
              Debug
            </button>
            <button className={tab === "highlight" ? "on" : ""} onClick={() => setTab("highlight")}>
              Highlight
            </button>
          </nav>
        </div>
      </header>

      {tab === "format" && <Formatter backend={backend} dark={dark} />}
      {tab === "notebook" && <Notebook backend={backend} dark={dark} />}
      {tab === "debug" && <Debug dark={dark} />}
      {tab === "highlight" && <Highlight />}

      <footer>
        Editor uses <code>@gad-lang/codemirror-gad</code>. Diagnostics and formatting come from the{" "}
        <strong>{backend.name}</strong> backend.
      </footer>
    </div>
  );
}

function Formatter({ backend, dark }: { backend: GadBackend; dark: boolean }) {
  const editorRef = useRef<EditorHandle>(null);
  const [left, setLeft] = useState<{ kind: "format" | "run"; fmt?: FormatResult; run?: RunResult } | null>(
    null,
  );
  const [busy, setBusy] = useState(false);

  // A stable diagnose reference per backend so the editor's linter is not
  // reconfigured on every render.
  const diagnose = useMemo(() => backend.diagnose, [backend]);

  const doFormat = async (apply: boolean) => {
    setBusy(true);
    try {
      const fmt = await backend.format(editorRef.current?.getValue() ?? "");
      setLeft({ kind: "format", fmt });
      if (apply && fmt.ok) editorRef.current?.setValue(fmt.source);
    } finally {
      setBusy(false);
    }
  };

  const doRun = async () => {
    setBusy(true);
    try {
      const run = await backend.run(editorRef.current?.getValue() ?? "");
      setLeft({ kind: "run", run });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="split">
      <section className="pane left">
        <div className="pane-head">Output</div>
        <div className="pane-body">
          {!left && <p className="hint">Format or run the source on the right.</p>}
          {left?.kind === "format" && left.fmt && <FormatView fmt={left.fmt} />}
          {left?.kind === "run" && left.run && <RunView run={left.run} />}
        </div>
      </section>

      <section className="pane right">
        <div className="pane-head">
          Source
          <span className="actions">
            <button onClick={() => doFormat(false)} disabled={busy}>
              Format
            </button>
            <button onClick={() => doFormat(true)} disabled={busy}>
              Format &amp; apply
            </button>
            <button onClick={doRun} disabled={busy}>
              Run ▶
            </button>
          </span>
        </div>
        <div className="pane-body">
          <Editor ref={editorRef} initialDoc={SAMPLE} diagnose={diagnose} dark={dark} />
        </div>
      </section>
    </div>
  );
}

function FormatView({ fmt }: { fmt: FormatResult }) {
  if (!fmt.ok) {
    return (
      <div className="diags">
        {fmt.diagnostics.map((d, i) => (
          <div className="diag" key={i}>
            {d.line}:{d.column} {d.message}
          </div>
        ))}
      </div>
    );
  }
  return <pre className="formatted">{fmt.source}</pre>;
}

function RunView({ run }: { run: RunResult }) {
  return (
    <div className={run.ok ? "" : "error"}>
      {run.stdout && <pre className="stdout">{run.stdout}</pre>}
      {run.stderr && <pre className="stderr">{run.stderr}</pre>}
      {run.ok && run.result && <div className="return">⇦ {run.result}</div>}
      {run.diagnostics.map((d, i) => (
        <div className="diag" key={i}>
          {d.line}:{d.column} {d.message}
        </div>
      ))}
    </div>
  );
}
