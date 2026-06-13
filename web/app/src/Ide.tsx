import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Editor, type EditorHandle } from "./Editor";
import { useTheme } from "./useTheme";
import {
  ideApi,
  type DebugResponse,
  type ModuleInfo,
  type RunConfig,
  type TreeNode,
  type Workspace,
} from "./backends/ide";

interface OpenTab {
  path: string;
  content: string;
  saved: boolean;
  runCfg: RunConfig;
}

type Pane = "output" | "stack" | "locals" | "breakpoints";

const emptyRunCfg = (): RunConfig => ({ args: [], disabled: [], safe: false, saveOut: "" });

/** The multi-file React IDE served by `gad ide`. */
export function Ide({ workspace }: { workspace: Workspace }) {
  const [theme, toggleTheme] = useTheme();
  const dark = theme === "dark";

  const [tree, setTree] = useState<TreeNode | null>(null);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [tabs, setTabs] = useState<OpenTab[]>([]);
  const [active, setActive] = useState(-1);
  const [pane, setPane] = useState<Pane>("output");
  const [output, setOutput] = useState("");
  const [stack, setStack] = useState<DebugResponse["frames"]>([]);
  const [locals, setLocals] = useState<DebugResponse["locals"]>([]);
  const [status, setStatus] = useState("");
  const [debug, setDebug] = useState<{ session: string; path: string } | null>(null);
  const [dialog, setDialog] = useState<null | { kind: "run" | "debug"; tab: OpenTab }>(null);
  const [settings, setSettings] = useState(false);
  const [bpScope, setBpScope] = useState<"current" | "all">("current");

  const editorRef = useRef<EditorHandle>(null);
  const activeTab = active >= 0 ? tabs[active] : null;

  const refreshTree = useCallback(async () => setTree(await ideApi.tree()), []);

  useEffect(() => {
    (async () => {
      try {
        setConfig(await ideApi.config());
        setModules(await ideApi.modules());
        await refreshTree();
        if (workspace.openFile) openFile(workspace.openFile);
      } catch (e) {
        setStatus("failed to start: " + e);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const runCfgFor = (path: string): RunConfig => {
    const ide = (config.ide as Record<string, unknown>) || {};
    const run = (ide.run as Record<string, RunConfig>) || {};
    return { ...emptyRunCfg(), ...(run[path] || {}) };
  };

  // Breakpoints live in .gad.yaml under ide.breakpoints as { path: [lines] }.
  const allBreakpoints = (): Record<string, number[]> =>
    ((config.ide as Record<string, unknown>)?.breakpoints as Record<string, number[]>) || {};
  const bpFor = (path?: string): number[] => (path ? allBreakpoints()[path] || [] : []);

  function setBreakpoints(path: string, lines: number[]) {
    setConfig((c) => {
      const ide = { ...((c.ide as Record<string, unknown>) || {}) };
      const bps = { ...((ide.breakpoints as Record<string, number[]>) || {}) };
      if (lines.length) bps[path] = [...lines].sort((a, b) => a - b);
      else delete bps[path];
      ide.breakpoints = bps;
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }

  async function openFile(path: string) {
    const existing = tabs.findIndex((t) => t.path === path);
    if (existing >= 0) {
      setActive(existing);
      return;
    }
    const data = await ideApi.read(path);
    setTabs((ts) => {
      const next = [...ts, { path, content: data.content, saved: true, runCfg: runCfgFor(path) }];
      setActive(next.length - 1);
      return next;
    });
  }

  function closeTab(i: number) {
    setTabs((ts) => {
      const next = ts.filter((_, idx) => idx !== i);
      setActive((a) => (a >= next.length ? next.length - 1 : a === i ? Math.max(0, i - 1) : a > i ? a - 1 : a));
      return next;
    });
  }

  function onEdit(value: string) {
    setTabs((ts) => ts.map((t, i) => (i === active ? { ...t, content: value, saved: false } : t)));
  }

  async function save() {
    if (!activeTab) return;
    const content = editorRef.current?.getValue() ?? activeTab.content;
    await ideApi.write(activeTab.path, content);
    setTabs((ts) => ts.map((t, i) => (i === active ? { ...t, content, saved: true } : t)));
    setStatus("saved " + activeTab.path);
  }

  async function format() {
    if (!activeTab) return;
    const content = editorRef.current?.getValue() ?? activeTab.content;
    const res = await ideApi.format(content);
    if (res.ok) {
      editorRef.current?.setValue(res.source);
      setTabs((ts) => ts.map((t, i) => (i === active ? { ...t, content: res.source, saved: false } : t)));
      setStatus("formatted");
    } else {
      showDiagnostics(res.diagnostics);
    }
  }

  function showDiagnostics(diags: { line: number; column: number; message: string }[]) {
    setOutput((diags || []).map((d) => `${d.line}:${d.column} ${d.message}`).join("\n"));
    setPane("output");
  }

  async function ensureSaved(tab: OpenTab): Promise<string> {
    const content = editorRef.current?.getValue() ?? tab.content;
    if (!tab.saved) {
      await ideApi.write(tab.path, content);
      setTabs((ts) => ts.map((t) => (t.path === tab.path ? { ...t, content, saved: true } : t)));
    }
    return content;
  }

  async function doRun(tab: OpenTab, cfg: RunConfig) {
    persistRunCfg(tab.path, cfg);
    const content = await ensureSaved(tab);
    setStatus("running…");
    setPane("output");
    try {
      const res = await ideApi.run({
        path: tab.path,
        source: content,
        args: cfg.args,
        disabled: cfg.disabled,
        safe: cfg.safe,
        saveOut: cfg.saveOut,
      });
      let s = "";
      if (res.stdout) s += res.stdout;
      if (res.stderr) s += res.stderr;
      if (res.ok && res.result) s += "\n⇦ " + res.result + "\n";
      (res.diagnostics || []).forEach((d) => (s += `${d.line}:${d.column} ${d.message}\n`));
      setOutput(s || "(no output)");
      setStatus(res.ok ? "done" : "error");
    } catch (e) {
      setOutput(String(e));
      setStatus("error");
    }
  }

  async function startDebug(tab: OpenTab, stopOnEntry: boolean) {
    const content = await ensureSaved(tab);
    setStatus("debugging…");
    setOutput("");
    try {
      const res = await ideApi.dbgStart({ source: content, breakpoints: bpFor(tab.path), stopOnEntry });
      applyDebug(res, tab.path);
    } catch (e) {
      setOutput(String(e));
      setStatus("error");
    }
  }

  async function dbgCommand(command: string) {
    if (!debug) return;
    if (command === "stop") {
      setDebug(null);
      setStatus("stopped");
      return;
    }
    const res = await ideApi.dbgCmd(debug.session, command);
    applyDebug(res, debug.path);
  }

  function applyDebug(res: DebugResponse, path: string) {
    if (res.output) setOutput((o) => o + res.output);
    if (res.state === "stopped") {
      setDebug({ session: res.session!, path });
      setStack(res.frames || []);
      setLocals(res.locals || []);
      setStatus(`stopped (${res.reason}) at line ${res.line}`);
      setPane("stack");
    } else if (res.state === "terminated") {
      if (res.result) setOutput((o) => o + "\n⇦ " + res.result + "\n");
      if (res.error) setOutput((o) => o + "\n" + res.error + "\n");
      setDebug(null);
      setStatus("program exited");
    } else {
      setOutput(res.error || "debug error");
      if (res.diagnostics) showDiagnostics(res.diagnostics);
      setDebug(null);
    }
  }

  function persistRunCfg(path: string, cfg: RunConfig) {
    setConfig((c) => {
      const ide = { ...((c.ide as Record<string, unknown>) || {}) };
      const run = { ...((ide.run as Record<string, RunConfig>) || {}) };
      run[path] = cfg;
      ide.run = run;
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }

  const diagnose = useMemo(() => ideApi.diagnose, []);

  return (
    <div className="ide" data-theme={theme}>
      <IdeStyles />
      <header className="ide-header">
        <span className="brand">Gad IDE</span>
        <span className="ws" title={workspace.root}>
          {workspace.root}
        </span>
        <span className="spacer" />
        <button onClick={() => setSettings(true)}>⚙ Settings</button>
        <button onClick={toggleTheme}>{dark ? "☀" : "☾"}</button>
      </header>

      <div className="ide-main">
        <aside className="ide-sidebar">
          <div className="side-head">
            <span>Explorer</span>
            <button
              onClick={async () => {
                const name = prompt("New file path (relative to workspace):", "untitled.gad");
                if (!name) return;
                await ideApi.mkfile(name);
                await refreshTree();
                openFile(name);
              }}
            >
              +
            </button>
          </div>
          <div className="tree">
            {tree?.children?.map((n) => (
              <TreeView key={n.path} node={n} activePath={activeTab?.path} onOpen={openFile} />
            ))}
          </div>
        </aside>

        <section className="ide-center">
          <div className="tabbar">
            {tabs.map((t, i) => (
              <div key={t.path} className={"tab" + (i === active ? " active" : "")} onClick={() => setActive(i)}>
                <span>
                  {t.path.split("/").pop()}
                  {t.saved ? "" : " •"}
                </span>
                <span
                  className="x"
                  onClick={(e) => {
                    e.stopPropagation();
                    closeTab(i);
                  }}
                >
                  ✕
                </span>
              </div>
            ))}
          </div>

          <div className="toolbar">
            <button onClick={save} disabled={!activeTab}>
              Save
            </button>
            <button onClick={format} disabled={!activeTab}>
              Format
            </button>
            <button onClick={() => activeTab && setDialog({ kind: "run", tab: activeTab })} disabled={!activeTab}>
              Run ▶
            </button>
            <button onClick={() => activeTab && setDialog({ kind: "debug", tab: activeTab })} disabled={!activeTab}>
              Debug 🐞
            </button>
            {debug && (
              <span className="dbgbar">
                <button onClick={() => dbgCommand("continue")}>Continue</button>
                <button onClick={() => dbgCommand("next")}>Step Over</button>
                <button onClick={() => dbgCommand("stepIn")}>Step In</button>
                <button onClick={() => dbgCommand("stepOut")}>Step Out</button>
                <button onClick={() => dbgCommand("stop")}>Stop</button>
              </span>
            )}
            <span className="spacer" />
            <span className="status">{status}</span>
          </div>

          <div className="editor-host">
            {activeTab ? (
              <Editor
                key={activeTab.path}
                ref={editorRef}
                initialDoc={activeTab.content}
                diagnose={diagnose}
                dark={dark}
                onChange={onEdit}
                breakpoints={bpFor(activeTab.path)}
                onBreakpointsChange={(lines) => setBreakpoints(activeTab.path, lines)}
              />
            ) : (
              <div className="empty">Open a file from the explorer</div>
            )}
          </div>

          <div className="panes">
            <div className="pane-tabs">
              {(["output", "stack", "locals", "breakpoints"] as Pane[]).map((p) => (
                <button key={p} className={pane === p ? "on" : ""} onClick={() => setPane(p)}>
                  {p === "output"
                    ? "Output"
                    : p === "stack"
                      ? "Call stack"
                      : p === "locals"
                        ? "Locals"
                        : "Breakpoints"}
                </button>
              ))}
              <span className="spacer" />
              {pane === "output" && <button onClick={() => setOutput("")}>Clear</button>}
            </div>
            {pane === "output" && <pre className="pane-body">{output}</pre>}
            {pane === "stack" && (
              <div className="pane-body">
                {(stack || []).map((f, i) => (
                  <div key={i} className="frame">
                    {f.name || "main"} <span className="muted">line {f.line}</span>
                  </div>
                ))}
              </div>
            )}
            {pane === "locals" && (
              <div className="pane-body">
                <table className="locals">
                  <tbody>
                    {(locals || []).map((v, i) => (
                      <tr key={i}>
                        <td>{v.name}</td>
                        <td className="muted">{v.type}</td>
                        <td>{v.value}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {pane === "breakpoints" && (
              <div className="pane-body">
                <div className="bp-scope">
                  <button className={bpScope === "current" ? "on" : ""} onClick={() => setBpScope("current")}>
                    Current file
                  </button>
                  <button className={bpScope === "all" ? "on" : ""} onClick={() => setBpScope("all")}>
                    All
                  </button>
                </div>
                {bpScope === "current" ? (
                  <BreakpointList
                    path={activeTab?.path}
                    lines={bpFor(activeTab?.path)}
                    onRemove={(line) =>
                      activeTab && setBreakpoints(activeTab.path, bpFor(activeTab.path).filter((l) => l !== line))
                    }
                  />
                ) : (
                  <BreakpointGroups
                    all={allBreakpoints()}
                    onOpen={openFile}
                    onRemove={(file, line) => setBreakpoints(file, (allBreakpoints()[file] || []).filter((l) => l !== line))}
                  />
                )}
              </div>
            )}
          </div>
        </section>
      </div>

      {dialog && (
        <RunDialog
          kind={dialog.kind}
          tab={dialog.tab}
          modules={modules}
          onCancel={() => setDialog(null)}
          onRun={(cfg) => {
            setDialog(null);
            doRun(dialog.tab, cfg);
          }}
          onDebug={(cfg, entry) => {
            setDialog(null);
            persistRunCfg(dialog.tab.path, cfg);
            startDebug(dialog.tab, entry);
          }}
        />
      )}

      {settings && (
        <SettingsDialog
          config={config}
          onClose={() => setSettings(false)}
          onSave={async (next) => {
            setConfig(next);
            await ideApi.saveConfig(next);
            setSettings(false);
            setStatus("settings saved");
          }}
        />
      )}
    </div>
  );
}

function TreeView({
  node,
  activePath,
  onOpen,
}: {
  node: TreeNode;
  activePath?: string;
  onOpen: (p: string) => void;
}) {
  const [open, setOpen] = useState(true);
  if (node.dir) {
    return (
      <div>
        <div className="node" onClick={() => setOpen((o) => !o)}>
          {open ? "📂" : "📁"} {node.name}
        </div>
        {open && (
          <div className="children">
            {node.children?.map((c) => (
              <TreeView key={c.path} node={c} activePath={activePath} onOpen={onOpen} />
            ))}
          </div>
        )}
      </div>
    );
  }
  return (
    <div className={"node file" + (node.path === activePath ? " active" : "")} onClick={() => onOpen(node.path)}>
      📄 {node.name}
    </div>
  );
}

function BreakpointList({
  path,
  lines,
  onRemove,
}: {
  path?: string;
  lines: number[];
  onRemove: (line: number) => void;
}) {
  if (!path) return <div className="muted">No file open.</div>;
  if (!lines.length) return <div className="muted">No breakpoints in {path.split("/").pop()}.</div>;
  return (
    <ul className="bp-list">
      {lines.map((l) => (
        <li key={l}>
          <span>line {l}</span>
          <button className="x" title="Remove" onClick={() => onRemove(l)}>
            ✕
          </button>
        </li>
      ))}
    </ul>
  );
}

function BreakpointGroups({
  all,
  onOpen,
  onRemove,
}: {
  all: Record<string, number[]>;
  onOpen: (path: string) => void;
  onRemove: (file: string, line: number) => void;
}) {
  const files = Object.keys(all).filter((f) => (all[f] || []).length);
  if (!files.length) return <div className="muted">No breakpoints set.</div>;
  return (
    <div>
      {files.sort().map((file) => (
        <div key={file} className="bp-group">
          <div className="bp-file" onClick={() => onOpen(file)} title="Open file">
            {file}
          </div>
          <ul className="bp-list">
            {[...all[file]].sort((a, b) => a - b).map((l) => (
              <li key={l}>
                <span>line {l}</span>
                <button className="x" title="Remove" onClick={() => onRemove(file, l)}>
                  ✕
                </button>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}

function RunDialog({
  kind,
  tab,
  modules,
  onCancel,
  onRun,
  onDebug,
}: {
  kind: "run" | "debug";
  tab: OpenTab;
  modules: ModuleInfo[];
  onCancel: () => void;
  onRun: (cfg: RunConfig) => void;
  onDebug: (cfg: RunConfig, stopOnEntry: boolean) => void;
}) {
  const [args, setArgs] = useState(tab.runCfg.args.join("\n"));
  const [disabled, setDisabled] = useState<string[]>(tab.runCfg.disabled);
  const [safe, setSafe] = useState(tab.runCfg.safe);
  const [saveOut, setSaveOut] = useState(tab.runCfg.saveOut);
  const [entry, setEntry] = useState(false);

  const toggle = (name: string) =>
    setDisabled((d) => (d.includes(name) ? d.filter((n) => n !== name) : [...d, name]));

  const cfg = (): RunConfig => ({
    args: args.split("\n").map((s) => s.trim()).filter(Boolean),
    disabled,
    safe,
    saveOut: saveOut.trim(),
  });

  return (
    <div className="modal-bg" onClick={(e) => e.target === e.currentTarget && onCancel()}>
      <div className="modal">
        <h3>
          {kind === "debug" ? "Debug" : "Run"} {tab.path}
        </h3>
        <label className="row">
          Arguments (one per line)
          <textarea rows={3} value={args} onChange={(e) => setArgs(e.target.value)} />
        </label>
        {kind === "debug" && (
          <label className="ck">
            <input type="checkbox" checked={entry} onChange={(e) => setEntry(e.target.checked)} /> Stop on entry
            <span className="muted"> (set breakpoints by clicking the gutter)</span>
          </label>
        )}
        <div className="row">
          Builtin modules (checked = enabled)
          <div className="mods">
            {modules.map((m) => (
              <label key={m.name} className="ck">
                <input type="checkbox" checked={!disabled.includes(m.name)} onChange={() => toggle(m.name)} />{" "}
                {m.name}
                {m.unsafe ? " (unsafe)" : ""}
              </label>
            ))}
          </div>
        </div>
        <label className="ck">
          <input type="checkbox" checked={safe} onChange={(e) => setSafe(e.target.checked)} /> Safe mode (disable
          unsafe modules)
        </label>
        <label className="row">
          Save stdout+stderr to file (optional)
          <input value={saveOut} onChange={(e) => setSaveOut(e.target.value)} placeholder="output.log" />
        </label>
        <div className="actions">
          <button onClick={onCancel}>Cancel</button>
          {kind === "debug" ? (
            <button onClick={() => onDebug(cfg(), entry)}>Start Debug</button>
          ) : (
            <button onClick={() => onRun(cfg())}>Run</button>
          )}
        </div>
      </div>
    </div>
  );
}

const NEWLINE_FLAGS: [string, string][] = [
  ["no-array-item-in-new-line", "Array items on new lines"],
  ["no-dict-item-in-new-line", "Dict items on new lines"],
  ["no-call-params-in-new-line", "Call params on new lines"],
];

function SettingsDialog({
  config,
  onClose,
  onSave,
}: {
  config: Record<string, unknown>;
  onClose: () => void;
  onSave: (next: Record<string, unknown>) => void;
}) {
  const fmt = (config.fmt as Record<string, unknown>) || {};
  // Checked = expanded layout (default, no key); unchecked writes no-…: true.
  const [expanded, setExpanded] = useState<Record<string, boolean>>(
    Object.fromEntries(NEWLINE_FLAGS.map(([k]) => [k, fmt[k] !== true])),
  );
  const [backup, setBackup] = useState(fmt.backup === true);

  function save() {
    const fmtObj: Record<string, unknown> = { ...fmt };
    for (const [k] of NEWLINE_FLAGS) {
      if (expanded[k]) delete fmtObj[k];
      else fmtObj[k] = true;
    }
    if (backup) fmtObj.backup = true;
    else delete fmtObj.backup;
    onSave({ ...config, fmt: fmtObj });
  }

  return (
    <div className="modal-bg" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <h3>Settings</h3>
        <div className="row">
          Formatter (.gad.yaml → fmt)
          {NEWLINE_FLAGS.map(([k, label]) => (
            <label key={k} className="ck">
              <input
                type="checkbox"
                checked={expanded[k]}
                onChange={(e) => setExpanded((s) => ({ ...s, [k]: e.target.checked }))}
              />{" "}
              {label}
            </label>
          ))}
          <label className="ck">
            <input type="checkbox" checked={backup} onChange={(e) => setBackup(e.target.checked)} /> Keep .backup on
            format
          </label>
        </div>
        <div className="actions">
          <button onClick={onClose}>Cancel</button>
          <button onClick={save}>Save</button>
        </div>
      </div>
    </div>
  );
}

/** IdeStyles injects the IDE-only layout CSS (kept out of the playground styles). */
function IdeStyles() {
  return (
    <style>{`
.ide{position:fixed;inset:0;display:flex;flex-direction:column;background:var(--bg);color:var(--fg);font-size:14px}
.ide-header{display:flex;align-items:center;gap:.5rem;padding:.4rem .7rem;border-bottom:1px solid var(--border);background:var(--panel)}
.ide-header .brand{font-weight:700}
.ide-header .ws{color:var(--muted);font-size:.82rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:40vw}
.ide .spacer{flex:1}
.ide-main{flex:1;display:flex;min-height:0}
.ide-sidebar{width:240px;border-right:1px solid var(--border);background:var(--panel);overflow:auto;padding:.4rem}
.side-head{display:flex;justify-content:space-between;align-items:center;font-size:.72rem;text-transform:uppercase;color:var(--muted);letter-spacing:.05em;padding:.2rem .3rem}
.tree .node{padding:.12rem .3rem;border-radius:4px;cursor:pointer;white-space:nowrap}
.tree .node:hover{background:var(--code-bg,rgba(125,125,125,.12))}
.tree .node.active{background:var(--accent);color:#fff}
.tree .children{margin-left:.8rem}
.ide-center{flex:1;display:flex;flex-direction:column;min-width:0}
.tabbar{display:flex;overflow:auto;border-bottom:1px solid var(--border);background:var(--panel)}
.tabbar .tab{display:flex;gap:.4rem;align-items:center;padding:.3rem .6rem;border-right:1px solid var(--border);cursor:pointer;white-space:nowrap}
.tabbar .tab.active{background:var(--bg)}
.tabbar .tab .x{opacity:.6}.tabbar .tab .x:hover{opacity:1}
.toolbar{display:flex;gap:.4rem;align-items:center;padding:.35rem .6rem;border-bottom:1px solid var(--border)}
.toolbar .status{color:var(--muted);font-size:.85rem}
.dbgbar{display:flex;gap:.3rem}
.editor-host{flex:1;min-height:0;display:flex}
.editor-host>div{flex:1;min-width:0}
.editor-host .empty{margin:auto;color:var(--muted)}
.panes{height:200px;border-top:1px solid var(--border);background:var(--panel);display:flex;flex-direction:column;resize:vertical;overflow:hidden}
.pane-tabs{display:flex;gap:.3rem;align-items:center;padding:.25rem .6rem;border-bottom:1px solid var(--border)}
.pane-tabs button.on{background:var(--accent);color:#fff}
.panes .pane-body{flex:1;overflow:auto;margin:0;padding:.5rem .8rem;white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:.85rem}
.frame{padding:.1rem .3rem}.muted{color:var(--muted)}
table.locals td{padding:.1rem .5rem;border-bottom:1px solid var(--border);font-family:ui-monospace,monospace}
.bp-scope{display:flex;gap:.3rem;margin-bottom:.4rem}
.bp-scope button.on{background:var(--accent);color:#fff}
.bp-group{margin-bottom:.5rem}
.bp-file{font-weight:600;cursor:pointer;padding:.1rem 0}
.bp-file:hover{color:var(--accent)}
.bp-list{list-style:none;margin:0;padding:0}
.bp-list li{display:flex;align-items:center;gap:.5rem;padding:.1rem .2rem}
.bp-list li::before{content:"●";color:#e5484d}
.bp-list .x{margin-left:auto;cursor:pointer;border:0;background:transparent;color:var(--muted)}
.bp-list .x:hover{color:#e5484d}
.modal-bg{position:fixed;inset:0;background:rgba(0,0,0,.4);display:flex;align-items:center;justify-content:center;z-index:50}
.modal{background:var(--panel);border:1px solid var(--border);border-radius:10px;padding:1rem 1.2rem;min-width:380px;max-width:90vw;max-height:85vh;overflow:auto}
.modal h3{margin:.2rem 0 .8rem}
.modal .row{display:flex;flex-direction:column;gap:.25rem;margin:.5rem 0}
.modal .mods{display:grid;grid-template-columns:1fr 1fr;gap:.2rem .8rem;max-height:160px;overflow:auto}
.modal .ck{display:flex;gap:.35rem;align-items:center;margin:.25rem 0}
.modal .actions{display:flex;gap:.5rem;justify-content:flex-end;margin-top:1rem}
    `}</style>
  );
}
