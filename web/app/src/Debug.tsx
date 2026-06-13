import { useCallback, useRef, useState } from "react";
import { Editor, type EditorHandle } from "./Editor";
import { debugBackend, type DebugResponse } from "./backends/debug";

const SAMPLE = `f := func(x) {
  y := x * 2
  return y
}
a := 1
b := f(20)
println("b =", b)
return a + b
`;

function parseBreakpoints(s: string): number[] {
  return s
    .split(",")
    .map((p) => parseInt(p.trim(), 10))
    .filter((n) => !Number.isNaN(n));
}

/**
 * Debug is the "Run & Debug" page. It drives the server-side debugger
 * (/api/debug/*): set breakpoints, start, and step while inspecting the call
 * stack, locals and output. Requires the Go server (make web-server).
 */
export function Debug({ dark }: { dark: boolean }) {
  const editorRef = useRef<EditorHandle>(null);
  const [bpText, setBpText] = useState("2, 7");
  const [stopOnEntry, setStopOnEntry] = useState(false);
  const [session, setSession] = useState<string | null>(null);
  const [snap, setSnap] = useState<DebugResponse | null>(null);
  const [output, setOutput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const apply = useCallback((r: DebugResponse) => {
    setSnap(r);
    if (r.output) setOutput((o) => o + r.output);
    if (r.state === "terminated" || r.state === "error") {
      setSession(null);
    } else if (r.session) {
      setSession(r.session);
    }
  }, []);

  const start = useCallback(async () => {
    setBusy(true);
    setError(null);
    setOutput("");
    setSnap(null);
    try {
      const r = await debugBackend.start(
        editorRef.current?.getValue() ?? "",
        parseBreakpoints(bpText),
        stopOnEntry,
      );
      apply(r);
    } catch (e) {
      setError(String(e) + " — is the Go server running? (make web-server)");
    } finally {
      setBusy(false);
    }
  }, [apply, bpText, stopOnEntry]);

  const cmd = useCallback(
    async (command: "continue" | "next" | "stepIn" | "stepOut") => {
      if (!session) return;
      setBusy(true);
      try {
        apply(await debugBackend.command(session, command));
      } catch (e) {
        setError(String(e));
        setSession(null);
      } finally {
        setBusy(false);
      }
    },
    [apply, session],
  );

  const stopped = snap?.state === "stopped";

  return (
    <div className="split">
      <section className="pane left">
        <div className="pane-head">Debug</div>
        <div className="pane-body debug-panel">
          <div className="debug-controls">
            <label>
              Breakpoints (lines):{" "}
              <input
                value={bpText}
                onChange={(e) => setBpText(e.target.value)}
                disabled={!!session}
                placeholder="e.g. 2, 7"
              />
            </label>
            <label>
              <input
                type="checkbox"
                checked={stopOnEntry}
                onChange={(e) => setStopOnEntry(e.target.checked)}
                disabled={!!session}
              />{" "}
              stop on entry
            </label>
          </div>
          <div className="debug-buttons">
            <button onClick={start} disabled={busy}>
              {session ? "Restart" : "Start ▶"}
            </button>
            <button onClick={() => cmd("continue")} disabled={!stopped || busy}>
              Continue
            </button>
            <button onClick={() => cmd("next")} disabled={!stopped || busy}>
              Step Over
            </button>
            <button onClick={() => cmd("stepIn")} disabled={!stopped || busy}>
              Step In
            </button>
            <button onClick={() => cmd("stepOut")} disabled={!stopped || busy}>
              Step Out
            </button>
          </div>

          {error && <div className="diag">{error}</div>}

          {snap && (
            <div className="debug-state">
              <div className="debug-status">
                {snap.state === "stopped" && (
                  <span>
                    stopped ({snap.reason}) at line {snap.line}:{snap.column}
                  </span>
                )}
                {snap.state === "terminated" && (
                  <span className="return">terminated — returned {snap.result || "nil"}</span>
                )}
                {snap.state === "error" && <span className="stderr">compile error</span>}
              </div>

              {snap.diagnostics?.map((d, i) => (
                <div className="diag" key={i}>
                  {d.line}:{d.column} {d.message}
                </div>
              ))}

              {stopped && (
                <>
                  <h4>Call stack</h4>
                  <ul className="debug-list">
                    {snap.frames?.map((f, i) => (
                      <li key={i}>
                        {f.name} <span className="muted">@ {f.line}:{f.column}</span>
                      </li>
                    ))}
                  </ul>
                  <h4>Locals</h4>
                  <ul className="debug-list">
                    {snap.locals?.length ? (
                      snap.locals.map((v, i) => (
                        <li key={i}>
                          {v.name} = {v.value} <span className="muted">({v.type})</span>
                        </li>
                      ))
                    ) : (
                      <li className="muted">(none)</li>
                    )}
                  </ul>
                </>
              )}
            </div>
          )}

          {output && (
            <>
              <h4>Output</h4>
              <pre className="stdout">{output}</pre>
            </>
          )}
        </div>
      </section>

      <section className="pane right">
        <div className="pane-head">Source</div>
        <div className="pane-body">
          <Editor ref={editorRef} initialDoc={SAMPLE} dark={dark} />
        </div>
      </section>
    </div>
  );
}
