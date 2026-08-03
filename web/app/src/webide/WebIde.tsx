// WebIde — a fully in-browser, server-less IDE meant to be embedded (e.g. in the
// Gad documentation site). The sample tree is read-only; every user change
// (edits, new files/dirs, deletions) lives in LocalStorage via WebFS and can be
// reset. Running, documenting and debugging all go through the Gad WebAssembly
// module hosted in a Web Worker (no Go server involved).
import { useCallback, useMemo, useRef, useState } from "react";
import { Editor, type EditorHandle, type EditorLanguage } from "../Editor";
import { WebFS } from "../webfs";
import { wasmWorkerBackend, wasmDebugBackend, sharedClient } from "../backends/wasmWorker";
import type { RunResult } from "../backends/types";
import type { DebugResponse } from "../backends/debug";
import { renderDocMarkdown } from "../docMarkdown";
import { buildTree, langFor, type FileNode } from "./tree";

type Tab = "run" | "doc" | "debug";

function editorLangFor(path: string): EditorLanguage {
  if (path.endsWith(".giom")) return "giom";
  if (path.endsWith(".gadt")) return "gadt";
  if (path.endsWith(".json")) return "json";
  if (path.endsWith(".md")) return "markdown";
  return "gad";
}

export function WebIde({ dark = false }: { dark?: boolean }) {
  const fs = useMemo(() => new WebFS(), []);
  // Bumped after any fs mutation to re-derive the tree and badges.
  const [rev, setRev] = useState(0);
  const bump = useCallback(() => setRev((r) => r + 1), []);

  const files = useMemo(() => fs.listFiles(), [fs, rev]);
  const dirs = useMemo(() => fs.listDirs(), [fs, rev]);
  const tree = useMemo(() => buildTree(files, dirs), [files, dirs]);

  const [openPath, setOpenPath] = useState<string>(() => files[0] ?? "");
  const [tab, setTab] = useState<Tab>("run");

  const editorRef = useRef<EditorHandle>(null);
  const source = () => editorRef.current?.getValue() ?? "";

  // Persist edits to the LocalStorage overlay (creating an override for a
  // read-only sample; the base sample is never mutated).
  const onChange = useCallback(
    (value: string) => {
      if (openPath) fs.write(openPath, value);
    },
    [fs, openPath],
  );

  const openFile = (path: string) => {
    setOpenPath(path);
    setTab("run");
  };

  // --- tree mutations (all LocalStorage) ---
  const newFile = () => {
    const base = currentDir(openPath);
    const name = prompt("New file name (e.g. hello.gad):", "untitled.gad");
    if (!name) return;
    const path = base + name;
    if (!fs.createFile(path)) return alert("already exists: " + path);
    bump();
    openFile(path);
  };
  const newDir = () => {
    const base = currentDir(openPath);
    const name = prompt("New folder name:", "folder");
    if (!name) return;
    fs.createDir(base + name);
    bump();
  };
  const removeOpen = () => {
    if (!openPath) return;
    if (!confirm("Delete " + openPath + "?")) return;
    fs.remove(openPath);
    const remaining = fs.listFiles();
    setOpenPath(remaining[0] ?? "");
    bump();
  };
  const resetAll = () => {
    if (!confirm("Discard all your changes and restore the original samples?")) return;
    fs.reset();
    const remaining = fs.listFiles();
    setOpenPath(remaining[0] ?? "");
    bump();
  };

  // A pristine base sample shows a "sample" badge; anything with a LocalStorage
  // override (an edited sample or a user-created file) shows "edited".
  const pristine = openPath ? fs.readOnlyBase(openPath) : false;
  const edited = openPath ? !pristine : false;

  return (
    <div className="webide">
      <aside className="webide-side">
        <div className="webide-side-head">
          <span>Files</span>
          <span className="webide-side-actions">
            <button title="New file" onClick={newFile}>＋</button>
            <button title="New folder" onClick={newDir}>🗀</button>
            <button title="Delete open file" onClick={removeOpen} disabled={!openPath}>🗑</button>
            <button title="Reset all changes" onClick={resetAll} disabled={!fs.hasChanges()}>⟲</button>
          </span>
        </div>
        <div className="webide-tree">
          <TreeView node={tree} openPath={openPath} onOpen={openFile} fs={fs} rev={rev} />
        </div>
        {fs.hasChanges() && <div className="webide-note">Local changes stored in your browser.</div>}
      </aside>

      <main className="webide-main">
        <div className="webide-editor-head">
          <span className="webide-path">{openPath || "(no file)"}</span>
          {pristine && <span className="webide-badge ro">sample</span>}
          {edited && <span className="webide-badge mod">edited</span>}
        </div>
        <div className="webide-editor">
          {openPath ? (
            <Editor
              key={openPath}
              ref={editorRef}
              initialDoc={fs.read(openPath) ?? ""}
              language={editorLangFor(openPath)}
              dark={dark}
              onChange={onChange}
            />
          ) : (
            <p className="hint">Create or select a file to begin.</p>
          )}
        </div>
      </main>

      <section className="webide-panel">
        <nav className="tabs">
          <button className={tab === "run" ? "on" : ""} onClick={() => setTab("run")}>Run</button>
          <button className={tab === "doc" ? "on" : ""} onClick={() => setTab("doc")}>Doc</button>
          <button className={tab === "debug" ? "on" : ""} onClick={() => setTab("debug")}>Debug</button>
        </nav>
        <div className="webide-panel-body">
          {tab === "run" && <RunPanel source={source} />}
          {tab === "doc" && <DocPanel source={source} path={openPath} />}
          {tab === "debug" && <DebugPanel source={source} />}
        </div>
      </section>
    </div>
  );
}

// currentDir returns the directory prefix (with trailing "/") of a path, or "".
function currentDir(path: string): string {
  const slash = path.lastIndexOf("/");
  return slash === -1 ? "" : path.slice(0, slash + 1);
}

function TreeView({
  node,
  openPath,
  onOpen,
  fs,
  rev,
  depth = 0,
}: {
  node: FileNode;
  openPath: string;
  onOpen: (p: string) => void;
  fs: WebFS;
  rev: number;
  depth?: number;
}) {
  return (
    <div>
      {node.children.map((c) =>
        c.dir ? (
          <TreeDir key={c.path} node={c} openPath={openPath} onOpen={onOpen} fs={fs} rev={rev} depth={depth} />
        ) : (
          <div
            key={c.path}
            className={"webide-file" + (c.path === openPath ? " on" : "")}
            style={{ paddingLeft: 8 + depth * 14 }}
            onClick={() => onOpen(c.path)}
            title={c.path}
          >
            {c.name}
            {!fs.readOnlyBase(c.path) && <span className="webide-dot" title="modified / user file" />}
          </div>
        ),
      )}
    </div>
  );
}

function TreeDir(props: {
  node: FileNode;
  openPath: string;
  onOpen: (p: string) => void;
  fs: WebFS;
  rev: number;
  depth: number;
}) {
  const { node, depth } = props;
  const [open, setOpen] = useState(true);
  return (
    <div>
      <div
        className="webide-dir"
        style={{ paddingLeft: 8 + depth * 14 }}
        onClick={() => setOpen((o) => !o)}
      >
        <span className="webide-twist">{open ? "▾" : "▸"}</span>
        {node.name}
      </div>
      {open && <TreeView {...props} depth={depth + 1} />}
    </div>
  );
}

function RunPanel({ source }: { source: () => string }) {
  const [res, setRes] = useState<RunResult | null>(null);
  const [busy, setBusy] = useState(false);
  const run = async () => {
    setBusy(true);
    try {
      setRes(await wasmWorkerBackend.run(source()));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div>
      <div className="webide-panel-actions">
        <button onClick={run} disabled={busy}>Run ▶</button>
      </div>
      {res && (
        <div className={res.ok ? "" : "error"}>
          {res.stdout && <pre className="stdout">{res.stdout}</pre>}
          {res.stderr && <pre className="stderr">{res.stderr}</pre>}
          {res.ok && res.result && <div className="return">⇦ {res.result}</div>}
          {res.diagnostics?.map((d, i) => (
            <div className="diag" key={i}>{d.line}:{d.column} {d.message}</div>
          ))}
        </div>
      )}
    </div>
  );
}

function DocPanel({ source, path }: { source: () => string; path: string }) {
  const [html, setHtml] = useState<string>("");
  const [busy, setBusy] = useState(false);
  const gen = async () => {
    setBusy(true);
    try {
      const r = await sharedClient().doc(source(), langFor(path));
      setHtml(r.error ? `<p class="diag">${escapeHtml(r.error)}</p>` : renderDocMarkdown(r.markdown ?? ""));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div>
      <div className="webide-panel-actions">
        <button onClick={gen} disabled={busy}>Generate docs</button>
      </div>
      {html ? (
        <div className="doc-md" dangerouslySetInnerHTML={{ __html: html }} />
      ) : (
        <p className="hint">Extract documentation from the open file.</p>
      )}
    </div>
  );
}

function DebugPanel({ source }: { source: () => string }) {
  const [bpText, setBpText] = useState("");
  const [session, setSession] = useState<string | null>(null);
  const [snap, setSnap] = useState<DebugResponse | null>(null);
  const [output, setOutput] = useState("");
  const [busy, setBusy] = useState(false);

  const apply = useCallback((r: DebugResponse) => {
    setSnap(r);
    if (r.output) setOutput((o) => o + r.output);
    if (r.state === "terminated" || r.state === "error") setSession(null);
    else if (r.session) setSession(r.session);
  }, []);

  const bps = () =>
    bpText
      .split(",")
      .map((p) => parseInt(p.trim(), 10))
      .filter((n) => !Number.isNaN(n));

  const start = async () => {
    setBusy(true);
    setOutput("");
    setSnap(null);
    try {
      apply(await wasmDebugBackend.start(source(), bps(), false));
    } finally {
      setBusy(false);
    }
  };
  const cmd = async (c: "continue" | "next" | "stepIn" | "stepOut") => {
    if (!session) return;
    setBusy(true);
    try {
      apply(await wasmDebugBackend.command(session, c));
    } finally {
      setBusy(false);
    }
  };

  const stopped = snap?.state === "stopped";
  return (
    <div className="debug-panel">
      <div className="debug-controls">
        <label>
          Breakpoints (lines):{" "}
          <input value={bpText} onChange={(e) => setBpText(e.target.value)} disabled={!!session} placeholder="e.g. 2, 5" />
        </label>
      </div>
      <div className="debug-buttons">
        <button onClick={start} disabled={busy}>{session ? "Restart" : "Start ▶"}</button>
        <button onClick={() => cmd("continue")} disabled={!stopped || busy}>Continue</button>
        <button onClick={() => cmd("next")} disabled={!stopped || busy}>Step Over</button>
        <button onClick={() => cmd("stepIn")} disabled={!stopped || busy}>Step In</button>
        <button onClick={() => cmd("stepOut")} disabled={!stopped || busy}>Step Out</button>
      </div>
      {snap && (
        <div className="debug-state">
          {snap.state === "stopped" && <div>stopped ({snap.reason}) at {snap.line}:{snap.column}</div>}
          {snap.state === "terminated" && <div className="return">terminated — returned {snap.result || "nil"}</div>}
          {snap.state === "error" && <div className="stderr">compile error</div>}
          {snap.diagnostics?.map((d, i) => (
            <div className="diag" key={i}>{d.line}:{d.column} {d.message}</div>
          ))}
          {stopped && (
            <>
              <h4>Locals</h4>
              <ul className="debug-list">
                {snap.locals?.length ? (
                  snap.locals.map((v, i) => <li key={i}>{v.name} = {v.value} <span className="muted">({v.type})</span></li>)
                ) : (
                  <li className="muted">(none)</li>
                )}
              </ul>
            </>
          )}
        </div>
      )}
      {output && <pre className="stdout">{output}</pre>}
    </div>
  );
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c] as string);
}
