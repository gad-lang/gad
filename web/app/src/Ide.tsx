import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  AppBar,
  Box,
  Button,
  Checkbox,
  CssBaseline,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Menu,
  MenuItem,
  TextField,
  ThemeProvider,
  Toolbar,
  Tooltip,
  Typography,
  createTheme,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import VisibilityIcon from "@mui/icons-material/Visibility";
import VisibilityOffIcon from "@mui/icons-material/VisibilityOff";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import RefreshIcon from "@mui/icons-material/Refresh";
import UndoIcon from "@mui/icons-material/Undo";
import RedoIcon from "@mui/icons-material/Redo";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import OutputIcon from "@mui/icons-material/Notes";
import CloseIcon from "@mui/icons-material/Close";

/** copyText writes text to the clipboard, ignoring failures (e.g. no permission). */
function copyText(text: string): void {
  void navigator.clipboard?.writeText(text).catch(() => {});
}
import { Editor, type EditorHandle } from "./Editor";
import { useTheme } from "./useTheme";
import {
  ideApi,
  type BreakpointMeta,
  type BreakpointSpec,
  type DebugResponse,
  type DocComment,
  type ModuleInfo,
  type RunConfig,
  type TreeNode,
  type Workspace,
} from "./backends/ide";
// EvalResult is referenced indirectly through ideApi.eval; no direct import needed.

interface OpenTab {
  path: string;
  content: string;
  saved: boolean;
  runCfg: RunConfig;
}

type Pane = "output" | "stack" | "locals" | "breakpoints" | "evaluate";

/** One entry in the Evaluate panel. */
interface EvalEntry {
  id: number;
  expr: string;
  repr: boolean;
  value: string;
  error: string;
}

const emptyRunCfg = (): RunConfig => ({ args: [], disabled: [], safe: false, saveOut: "" });

// Debugger keybindings: action -> default key chord. Stored under ide.keys.
const DEFAULT_KEYS: Record<string, string> = {
  continue: "F9",
  stepOver: "F8",
  stepInto: "F7",
  stepOut: "Shift+F8",
};

// KEY_ACTIONS maps each bindable action to its debug command + label.
const KEY_ACTIONS: { action: string; cmd: string; label: string }[] = [
  { action: "continue", cmd: "continue", label: "Resume (next breakpoint)" },
  { action: "stepOver", cmd: "next", label: "Step over" },
  { action: "stepInto", cmd: "stepIn", label: "Step into" },
  { action: "stepOut", cmd: "stepOut", label: "Step out" },
];

const MODIFIER_KEYS = ["Shift", "Control", "Alt", "Meta"];

/** eventToKey renders a keydown event as a chord string like "Shift+F8". */
function eventToKey(e: KeyboardEvent): string {
  const parts: string[] = [];
  if (e.ctrlKey) parts.push("Ctrl");
  if (e.shiftKey) parts.push("Shift");
  if (e.altKey) parts.push("Alt");
  if (e.metaKey) parts.push("Meta");
  if (!MODIFIER_KEYS.includes(e.key)) parts.push(e.key.length === 1 ? e.key.toUpperCase() : e.key);
  return parts.join("+");
}

function keysFromConfig(config: Record<string, unknown>): Record<string, string> {
  return { ...DEFAULT_KEYS, ...((config.ide as Record<string, unknown>)?.keys as Record<string, string>) };
}

/** The multi-file React IDE served by `gad ide`. */
export function Ide({ workspace }: { workspace: Workspace }) {
  const [theme, toggleTheme] = useTheme();
  const dark = theme === "dark";

  const [tree, setTree] = useState<TreeNode | null>(null);
  const [showHidden, setShowHidden] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<TreeNode | null>(null);
  const [pendingRunPath, setPendingRunPath] = useState<string | null>(null);
  const [errorDialog, setErrorDialog] = useState<{ title: string; detail: string } | null>(null);
  const [evals, setEvals] = useState<EvalEntry[]>([]);
  const [outputDialog, setOutputDialog] = useState<{ title: string; text: string } | null>(null);
  const [bpDialog, setBpDialog] = useState<{ path: string; line: number } | null>(null);
  const [docPanel, setDocPanel] = useState(false);
  const [docs, setDocs] = useState<DocComment[]>([]);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [tabs, setTabs] = useState<OpenTab[]>([]);
  const [active, setActive] = useState(-1);
  const [pane, setPane] = useState<Pane>("output");
  const [output, setOutput] = useState("");
  const [stack, setStack] = useState<DebugResponse["frames"]>([]);
  const [locals, setLocals] = useState<DebugResponse["locals"]>([]);
  const [status, setStatus] = useState("");

  // reportError surfaces a backend/operation failure in both the status line and
  // a modal dialog, and returns the short message for convenience.
  const reportError = useCallback((title: string, e: unknown): string => {
    const detail = e instanceof Error ? e.message : String(e);
    setStatus(title + ": " + detail);
    setErrorDialog({ title, detail });
    return detail;
  }, []);
  const [debug, setDebug] = useState<{ session: string; path: string } | null>(null);
  const [debugLoc, setDebugLoc] = useState<{ line: number; column: number } | null>(null);
  const [pendingGoto, setPendingGoto] = useState<{ path: string; line: number; column: number } | null>(null);
  const [dialog, setDialog] = useState<null | { kind: "run" | "debug"; tab: OpenTab }>(null);
  const [settings, setSettings] = useState(false);
  const [keybinds, setKeybinds] = useState(false);
  const [bpScope, setBpScope] = useState<"current" | "all">("current");
  const [selectedFrame, setSelectedFrame] = useState(0);
  const frameClickTimer = useRef<number | null>(null);

  const editorRef = useRef<EditorHandle>(null);
  const activeTab = active >= 0 ? tabs[active] : null;

  const refreshTree = useCallback(
    async () => setTree(await ideApi.tree(showHidden)),
    [showHidden],
  );

  // Reload the tree whenever the hidden-files toggle changes.
  useEffect(() => {
    void refreshTree();
  }, [refreshTree]);

  useEffect(() => {
    (async () => {
      try {
        setConfig(await ideApi.config());
        setModules(await ideApi.modules());
        // The tree is loaded by the showHidden effect above.
        if (workspace.openFile) openFile(workspace.openFile);
      } catch (e) {
        reportError("Failed to start", e);
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
      // Drop metadata for lines that no longer have a breakpoint.
      const allMeta = { ...((ide.breakpointMeta as Record<string, BreakpointMeta>) || {}) };
      if (allMeta[path]) {
        const kept: BreakpointMeta = {};
        for (const l of lines) if (allMeta[path][l]) kept[l] = allMeta[path][l];
        if (Object.keys(kept).length) allMeta[path] = kept;
        else delete allMeta[path];
        ide.breakpointMeta = allMeta;
      }
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }

  // Per-line breakpoint metadata (disabled / condition), keyed by path then line.
  const bpMetaFor = (path?: string): BreakpointMeta =>
    (path
      ? ((config.ide as Record<string, unknown>)?.breakpointMeta as Record<string, BreakpointMeta>)?.[path]
      : undefined) || {};

  function setBpMeta(path: string, line: number, meta: { disabled?: boolean; condition?: string }) {
    setConfig((c) => {
      const ide = { ...((c.ide as Record<string, unknown>) || {}) };
      const allMeta = { ...((ide.breakpointMeta as Record<string, BreakpointMeta>) || {}) };
      const forPath = { ...(allMeta[path] || {}) };
      const clean = {
        ...(meta.disabled ? { disabled: true } : {}),
        ...(meta.condition && meta.condition.trim() ? { condition: meta.condition.trim() } : {}),
      };
      if (Object.keys(clean).length) forPath[line] = clean;
      else delete forPath[line];
      if (Object.keys(forPath).length) allMeta[path] = forPath;
      else delete allMeta[path];
      ide.breakpointMeta = allMeta;
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }

  // Build the breakpointSpecs payload for a debug start from lines + metadata.
  const bpSpecsFor = (path: string): BreakpointSpec[] => {
    const meta = bpMetaFor(path);
    return bpFor(path).map((line) => ({ line, ...(meta[line] || {}) }));
  };

  // onFrameClick distinguishes a single click (select the frame and show its
  // locals) from a double click (navigate to the frame's source position).
  function onFrameClick(i: number, f: { file: string; line: number; column: number }) {
    if (frameClickTimer.current !== null) {
      window.clearTimeout(frameClickTimer.current);
      frameClickTimer.current = null;
      gotoFrame(f.file, f.line, f.column);
      return;
    }
    frameClickTimer.current = window.setTimeout(() => {
      frameClickTimer.current = null;
      setSelectedFrame(i);
      setPane("locals");
    }, 250);
  }

  // gotoFrame opens the frame's file (if needed) and queues a cursor move.
  async function gotoFrame(file: string, line: number, column: number) {
    if (!file) return;
    try {
      await openFile(file);
      setPendingGoto({ path: file, line, column });
    } catch {
      /* synthetic or missing source file */
    }
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

  // --- Evaluate panel -------------------------------------------------------

  // evalOne evaluates a single expression entry and returns the updated entry.
  // While a debug session is paused it evaluates in the live frame's scope;
  // otherwise it runs standalone against the active file as context.
  const evalOne = useCallback(
    async (entry: EvalEntry): Promise<EvalEntry> => {
      try {
        const res = debug
          ? await ideApi.dbgEval(debug.session, entry.expr, entry.repr)
          : await ideApi.eval({
              expr: entry.expr,
              repr: entry.repr,
              source: editorRef.current?.getValue() ?? activeTab?.content ?? "",
              path: activeTab?.path,
            });
        return res.ok
          ? { ...entry, value: res.value ?? "", error: "" }
          : { ...entry, value: "", error: res.error || "error" };
      } catch (e) {
        return { ...entry, value: "", error: e instanceof Error ? e.message : String(e) };
      }
    },
    [activeTab, debug],
  );

  // Re-evaluate every entry (used on add and whenever the debugger steps).
  const evalAll = useCallback(async () => {
    setEvals((cur) => {
      void Promise.all(cur.map(evalOne)).then(setEvals);
      return cur;
    });
  }, [evalOne]);

  async function addEval(expr: string, repr: boolean) {
    const entry: EvalEntry = { id: Date.now(), expr, repr, value: "", error: "" };
    const evaluated = await evalOne(entry);
    setEvals((cur) => [...cur, evaluated]);
  }

  // Refresh the Evaluate panel whenever the debugger stops at a new location
  // (debugLoc changes per step). Runs after state settles, so evalOne sees the
  // current debug session and evaluates in the live frame.
  useEffect(() => {
    if (debug) void evalAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debugLoc]);

  // --- Doc-comment panel ----------------------------------------------------

  const reloadDocs = useCallback(async () => {
    if (!docPanel) return;
    const src = editorRef.current?.getValue() ?? activeTab?.content ?? "";
    try {
      setDocs(await ideApi.doc(src));
    } catch {
      /* leave previous docs on a transient failure */
    }
  }, [docPanel, activeTab]);

  // Refresh docs when the panel opens or the active file changes.
  useEffect(() => {
    void reloadDocs();
  }, [reloadDocs, active]);

  // Auto-reload docs 5s after the last edit while the panel is open.
  useEffect(() => {
    if (!docPanel || !activeTab || activeTab.saved) return;
    const t = window.setTimeout(() => void reloadDocs(), 5000);
    return () => window.clearTimeout(t);
  }, [docPanel, activeTab, reloadDocs]);

  // Reload the active file from disk, discarding unsaved edits (after a confirm
  // when the buffer is dirty).
  async function reloadFile() {
    if (!activeTab) return;
    if (!activeTab.saved && !confirm(`Discard unsaved changes to ${activeTab.path}?`)) return;
    try {
      const data = await ideApi.read(activeTab.path);
      editorRef.current?.setValue(data.content);
      setTabs((ts) => ts.map((t, i) => (i === active ? { ...t, content: data.content, saved: true } : t)));
      setStatus("reloaded " + activeTab.path);
    } catch (e) {
      reportError("Reload failed", e);
    }
  }

  async function save() {
    if (!activeTab) return;
    const content = editorRef.current?.getValue() ?? activeTab.content;
    try {
      await ideApi.write(activeTab.path, content);
      setTabs((ts) => ts.map((t, i) => (i === active ? { ...t, content, saved: true } : t)));
      setStatus("saved " + activeTab.path);
    } catch (e) {
      reportError("Save failed", e);
    }
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

  // Format a file on disk (used by the explorer context menu) without requiring
  // it to be the active editor tab.
  async function formatFile(path: string) {
    try {
      const data = await ideApi.read(path);
      const res = await ideApi.format(data.content);
      if (!res.ok) {
        showDiagnostics(res.diagnostics);
        setStatus("format failed: " + path);
        return;
      }
      await ideApi.write(path, res.source);
      setTabs((ts) => ts.map((t) => (t.path === path ? { ...t, content: res.source, saved: true } : t)));
      if (activeTab?.path === path) editorRef.current?.setValue(res.source);
      setStatus("formatted " + path);
    } catch (e) {
      reportError("Format failed", e);
    }
  }

  // Transpile a file to a sibling .gad (a .gadt becomes name.gad; another .gad
  // becomes name.transpiled.gad to avoid clobbering the source) and open it.
  async function transpileFile(path: string) {
    try {
      const data = await ideApi.read(path);
      const res = await ideApi.transpile(data.content, path);
      if (!res.ok) {
        showDiagnostics(res.diagnostics);
        setStatus("transpile failed: " + path);
        return;
      }
      const out = path.endsWith(".gadt")
        ? path.slice(0, -1) // .gadt -> .gad
        : path.replace(/\.gad$/, ".transpiled.gad");
      await ideApi.write(out, res.source);
      await refreshTree();
      await openFile(out);
      setStatus("transpiled to " + out);
    } catch (e) {
      reportError("Transpile failed", e);
    }
  }

  // Explorer context-menu / keyboard actions on a tree node.
  const treeAction = useCallback(
    async (action: TreeAction, node: TreeNode) => {
      switch (action) {
        case "open":
          void openFile(node.path);
          break;
        case "rename": {
          const to = prompt("Rename to (path relative to workspace):", node.path);
          if (!to || to === node.path) return;
          try {
            await ideApi.rename(node.path, to);
            setTabs((ts) => ts.map((t) => (t.path === node.path ? { ...t, path: to } : t)));
            await refreshTree();
            setStatus("renamed to " + to);
          } catch (e) {
            reportError("Rename failed", e);
          }
          break;
        }
        case "remove":
          setRemoveTarget(node);
          break;
        case "format":
          await formatFile(node.path);
          break;
        case "run":
          await openFile(node.path);
          setPendingRunPath(node.path);
          break;
        case "transpile":
          await transpileFile(node.path);
          break;
      }
    },
    // openFile/formatFile/refreshTree are stable enough for this menu handler.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [refreshTree],
  );

  // Open the run dialog once a context-menu "run" target has finished opening.
  useEffect(() => {
    if (!pendingRunPath) return;
    const tab = tabs.find((t) => t.path === pendingRunPath);
    if (tab) {
      setDialog({ kind: "run", tab });
      setPendingRunPath(null);
    }
  }, [pendingRunPath, tabs]);

  // Confirm and execute a tree-node removal.
  async function confirmRemove(recursive: boolean) {
    const node = removeTarget;
    setRemoveTarget(null);
    if (!node) return;
    if (node.dir && (node.children?.length ?? 0) > 0 && !recursive) return;
    try {
      await ideApi.del(node.path);
      setTabs((ts) => ts.filter((t) => t.path !== node.path && !t.path.startsWith(node.path + "/")));
      await refreshTree();
      setStatus("removed " + node.path);
    } catch (e) {
      reportError("Remove failed", e);
    }
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
      const cfg = tab.runCfg;
      const res = await ideApi.dbgStart({
        source: content,
        breakpoints: bpFor(tab.path),
        breakpointSpecs: bpSpecsFor(tab.path),
        stopOnEntry,
        path: tab.path,
        args: cfg.args,
        disabled: cfg.disabled,
        safe: cfg.safe,
      });
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
      setDebugLoc(null);
      setStatus("stopped");
      return;
    }
    const res = await ideApi.dbgCmd(debug.session, command);
    applyDebug(res, debug.path);
  }

  function applyDebug(res: DebugResponse, path: string) {
    if (res.output) setOutput((o) => o + res.output);
    if (res.state === "stopped") {
      // Follow the stop into its file: stepping into an imported module reports
      // that module's path, so open it and highlight the line there. The
      // "(main)" sentinel (inline source, no path) maps back to the debugged file.
      const stopFile = res.file && res.file !== "(main)" ? res.file : path;
      setDebug({ session: res.session!, path: stopFile });
      setStack(res.frames || []);
      setLocals(res.locals || []);
      setSelectedFrame(0);
      setDebugLoc({ line: res.line ?? 0, column: res.column ?? 1 });
      setStatus(`stopped (${res.reason}) at ${stopFile}:${res.line}`);
      setPane("stack");
      if (stopFile && stopFile !== activeTab?.path) void openFile(stopFile);
      // The Evaluate panel refreshes via an effect on debugLoc (so it sees the
      // just-updated debug session).
    } else if (res.state === "terminated") {
      if (res.result) setOutput((o) => o + "\n⇦ " + res.result + "\n");
      if (res.error) setOutput((o) => o + "\n" + res.error + "\n");
      setDebug(null);
      setDebugLoc(null);
      setStatus("program exited");
    } else {
      setOutput(res.error || "debug error");
      if (res.diagnostics) showDiagnostics(res.diagnostics);
      setDebug(null);
      setDebugLoc(null);
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

  // Debugger keyboard shortcuts (active only while a debug session is paused).
  useEffect(() => {
    if (!debug) return;
    const keys = keysFromConfig(config);
    const handler = (e: KeyboardEvent) => {
      const pressed = eventToKey(e);
      const hit = KEY_ACTIONS.find((a) => keys[a.action] === pressed);
      if (hit) {
        e.preventDefault();
        dbgCommand(hit.cmd);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debug, config]);

  // Apply a queued cursor move once its file's tab is active and mounted.
  useEffect(() => {
    if (pendingGoto && activeTab?.path === pendingGoto.path) {
      editorRef.current?.gotoLocation(pendingGoto.line, pendingGoto.column);
      setPendingGoto(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pendingGoto, active]);

  const diagnose = useMemo(() => ideApi.diagnose, []);
  const keys = keysFromConfig(config);
  const fontSize = ((config.ide as Record<string, unknown>)?.fontSize as number) || 14;

  function setFontSize(px: number) {
    const clamped = Math.min(28, Math.max(9, px));
    setConfig((c) => {
      const ide = { ...((c.ide as Record<string, unknown>) || {}), fontSize: clamped };
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }

  const muiTheme = useMemo(
    () => createTheme({ palette: { mode: dark ? "dark" : "light", primary: { main: dark ? "#8aa6ff" : "#3b5bdb" } } }),
    [dark],
  );

  return (
    <ThemeProvider theme={muiTheme}>
      <CssBaseline />
      <Box className="ide" data-theme={theme}>
        <IdeStyles />
        <AppBar position="static" color="default" elevation={1}>
          <Toolbar variant="dense" sx={{ gap: 1 }}>
            <Typography variant="h6" sx={{ fontSize: "1.05rem", fontWeight: 700 }}>
              Gad IDE
            </Typography>
            <Typography variant="body2" color="text.secondary" noWrap sx={{ maxWidth: "40vw" }} title={workspace.root}>
              {workspace.root}
            </Typography>
            <Box sx={{ flex: 1 }} />
            <Button size="small" onClick={() => setKeybinds(true)}>
              ⌨ Keys
            </Button>
            <Button size="small" onClick={() => setSettings(true)}>
              ⚙ Settings
            </Button>
            <IconButton size="small" onClick={toggleTheme} title="Toggle theme">
              {dark ? "☀" : "☾"}
            </IconButton>
          </Toolbar>
        </AppBar>

      <div className="ide-main">
        <aside className="ide-sidebar">
          <div className="side-head">
            <span>Explorer</span>
            <span style={{ flex: 1 }} />
            <IconButton
              size="small"
              title={showHidden ? "Hide hidden files" : "Show hidden files"}
              onClick={() => setShowHidden((v) => !v)}
            >
              {showHidden ? (
                <VisibilityIcon fontSize="small" />
              ) : (
                <VisibilityOffIcon fontSize="small" />
              )}
            </IconButton>
            <IconButton
              size="small"
              title="New file"
              onClick={async () => {
                const name = prompt("New file path (relative to workspace):", "untitled.gad");
                if (!name) return;
                await ideApi.mkfile(name);
                await refreshTree();
                openFile(name);
              }}
            >
              <AddIcon fontSize="small" />
            </IconButton>
          </div>
          <div className="tree">
            {tree?.children?.map((n) => (
              <TreeView
                key={n.path}
                node={n}
                activePath={activeTab?.path}
                onOpen={openFile}
                onAction={treeAction}
              />
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

          <Toolbar variant="dense" className="toolbar" disableGutters sx={{ gap: 1, minHeight: 44 }}>
            <Button size="small" variant="outlined" onClick={save} disabled={!activeTab}>
              Save
            </Button>
            <Button size="small" variant="outlined" onClick={format} disabled={!activeTab}>
              Format
            </Button>
            <Tooltip title="Reload from disk">
              <span>
                <IconButton size="small" onClick={reloadFile} disabled={!activeTab}>
                  <RefreshIcon fontSize="small" />
                </IconButton>
              </span>
            </Tooltip>
            <Tooltip title="Undo">
              <span>
                <IconButton
                  size="small"
                  onClick={() => editorRef.current?.undo()}
                  disabled={!activeTab}
                >
                  <UndoIcon fontSize="small" />
                </IconButton>
              </span>
            </Tooltip>
            <Tooltip title="Redo">
              <span>
                <IconButton
                  size="small"
                  onClick={() => editorRef.current?.redo()}
                  disabled={!activeTab}
                >
                  <RedoIcon fontSize="small" />
                </IconButton>
              </span>
            </Tooltip>
            <Button
              size="small"
              variant="contained"
              color="success"
              onClick={() => activeTab && setDialog({ kind: "run", tab: activeTab })}
              disabled={!activeTab}
            >
              Run ▶
            </Button>
            <Button
              size="small"
              variant="contained"
              color="warning"
              onClick={() => activeTab && setDialog({ kind: "debug", tab: activeTab })}
              disabled={!activeTab}
            >
              Debug 🐞
            </Button>
            {debug && (
              <Box className="dbgbar">
                <Tooltip title={`Resume (${keys.continue})`}>
                  <Button size="small" onClick={() => dbgCommand("continue")}>
                    Continue ({keys.continue})
                  </Button>
                </Tooltip>
                <Tooltip title={`Step over (${keys.stepOver})`}>
                  <Button size="small" onClick={() => dbgCommand("next")}>
                    Step Over ({keys.stepOver})
                  </Button>
                </Tooltip>
                <Tooltip title={`Step into (${keys.stepInto})`}>
                  <Button size="small" onClick={() => dbgCommand("stepIn")}>
                    Step In ({keys.stepInto})
                  </Button>
                </Tooltip>
                <Tooltip title={`Step out (${keys.stepOut})`}>
                  <Button size="small" onClick={() => dbgCommand("stepOut")}>
                    Step Out ({keys.stepOut})
                  </Button>
                </Tooltip>
                <Button size="small" color="error" onClick={() => dbgCommand("stop")}>
                  Stop
                </Button>
              </Box>
            )}
            <Box sx={{ flex: 1 }} />
            <Tooltip title="Toggle doc-comments panel">
              <Button
                size="small"
                variant={docPanel ? "contained" : "outlined"}
                onClick={() => setDocPanel((v) => !v)}
              >
                Docs
              </Button>
            </Tooltip>
            <Box className="font-control" title="Editor font size">
              <Button size="small" onClick={() => setFontSize(fontSize - 1)}>
                A−
              </Button>
              <span className="font-size">{fontSize}px</span>
              <Button size="small" onClick={() => setFontSize(fontSize + 1)}>
                A+
              </Button>
            </Box>
            <Typography variant="caption" color="text.secondary">
              {status}
            </Typography>
          </Toolbar>

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
                fontSize={fontSize}
                debugLine={debug && debug.path === activeTab.path ? debugLoc?.line : undefined}
                debugColumn={debug && debug.path === activeTab.path ? debugLoc?.column : undefined}
                locals={debug && debug.path === activeTab.path ? locals : undefined}
              />
            ) : (
              <div className="empty">Open a file from the explorer</div>
            )}
            {docPanel && (
              <DocPanel
                docs={docs}
                onReload={reloadDocs}
                onClose={() => setDocPanel(false)}
                onGoto={(line) => editorRef.current?.gotoLocation(line, 1)}
              />
            )}
          </div>

          <div className="panes">
            <div className="pane-tabs">
              {(["output", "stack", "locals", "breakpoints", "evaluate"] as Pane[]).map((p) => (
                <button key={p} className={pane === p ? "on" : ""} onClick={() => setPane(p)}>
                  {p === "output"
                    ? "Output"
                    : p === "stack"
                      ? "Call stack"
                      : p === "locals"
                        ? "Locals"
                        : p === "breakpoints"
                          ? "Breakpoints"
                          : "Evaluate"}
                </button>
              ))}
              <span className="spacer" />
              {pane === "output" && <button onClick={() => setOutput("")}>Clear</button>}
            </div>
            {pane === "output" && <pre className="pane-body">{output}</pre>}
            {pane === "stack" && (
              <div className="pane-body">
                {(stack || []).map((f, i) => (
                  <div
                    key={i}
                    className={"frame" + (i === selectedFrame ? " selected" : "")}
                    title={`${f.file}:${f.line}:${f.column} — click to inspect, double-click to open`}
                    onClick={() => onFrameClick(i, f)}
                    style={{ cursor: "pointer" }}
                  >
                    <span className="fn">{f.name || "main"}</span>{" "}
                    <span className="muted">
                      {f.file ? `${f.file.split("/").pop()}:` : ""}
                      {f.line}:{f.column}
                    </span>
                  </div>
                ))}
              </div>
            )}
            {pane === "locals" &&
              (() => {
                const frame = stack && stack[selectedFrame];
                const frameLocals = frame ? frame.locals : locals;
                return (
                  <div className="pane-body">
                    {frame && (
                      <div className="locals-head muted">
                        {frame.name || "main"} — {frame.file ? frame.file.split("/").pop() + ":" : ""}
                        {frame.line}:{frame.column}
                      </div>
                    )}
                    <table className="locals">
                      <tbody>
                        {(frameLocals || []).map((v, i) => (
                          <tr key={i}>
                            <td>{v.name}</td>
                            <td className="muted">{v.type}</td>
                            <td>{v.value}</td>
                            <td className="locals-copy">
                              <IconButton
                                size="small"
                                title="Copy value"
                                onClick={() => copyText(v.value)}
                              >
                                <ContentCopyIcon sx={{ fontSize: 14 }} />
                              </IconButton>
                            </td>
                          </tr>
                        ))}
                        {(frameLocals || []).length === 0 && (
                          <tr>
                            <td className="muted">(no locals)</td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                );
              })()}
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
                    meta={bpMetaFor(activeTab?.path)}
                    onEdit={(line) => activeTab && setBpDialog({ path: activeTab.path, line })}
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
            {pane === "evaluate" && (
              <EvaluatePanel
                entries={evals}
                onAdd={addEval}
                onUpdate={async (id, expr, repr) => {
                  const updated = await evalOne({ id, expr, repr, value: "", error: "" });
                  setEvals((cur) => cur.map((e) => (e.id === id ? updated : e)));
                }}
                onRemove={(id) => setEvals((cur) => cur.filter((e) => e.id !== id))}
                onShowOutput={(e) =>
                  setOutputDialog({ title: e.expr, text: e.error || e.value })
                }
                onCopy={copyText}
              />
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

      {keybinds && (
        <KeybindingsDialog
          config={config}
          onClose={() => setKeybinds(false)}
          onSave={async (next) => {
            setConfig(next);
            await ideApi.saveConfig(next);
            setKeybinds(false);
            setStatus("keybindings saved");
          }}
        />
      )}
      {removeTarget && (
        <RemoveDialog
          node={removeTarget}
          onClose={() => setRemoveTarget(null)}
          onConfirm={confirmRemove}
        />
      )}
      {bpDialog && (
        <BreakpointDialog
          line={bpDialog.line}
          initial={bpMetaFor(bpDialog.path)[bpDialog.line] || {}}
          onClose={() => setBpDialog(null)}
          onSave={(meta) => {
            setBpMeta(bpDialog.path, bpDialog.line, meta);
            setBpDialog(null);
          }}
        />
      )}
      {outputDialog && (
        <Dialog open onClose={() => setOutputDialog(null)} maxWidth="md" fullWidth>
          <DialogTitle>{outputDialog.title}</DialogTitle>
          <DialogContent dividers>
            <TextField
              multiline
              fullWidth
              minRows={6}
              maxRows={20}
              value={outputDialog.text}
              slotProps={{
                input: { readOnly: true, sx: { fontFamily: "ui-monospace, monospace", fontSize: ".85rem" } },
              }}
            />
          </DialogContent>
          <DialogActions>
            <Button onClick={() => copyText(outputDialog.text)}>Copy</Button>
            <Box sx={{ flex: 1 }} />
            <Button variant="contained" onClick={() => setOutputDialog(null)}>
              Close
            </Button>
          </DialogActions>
        </Dialog>
      )}
      {errorDialog && (
        <Dialog open onClose={() => setErrorDialog(null)} maxWidth="sm" fullWidth>
          <DialogTitle>{errorDialog.title}</DialogTitle>
          <DialogContent dividers>
            <Typography
              component="pre"
              sx={{ whiteSpace: "pre-wrap", fontFamily: "ui-monospace, monospace", fontSize: ".85rem", m: 0 }}
            >
              {errorDialog.detail}
            </Typography>
          </DialogContent>
          <DialogActions>
            <Button onClick={() => copyText(errorDialog.detail)}>Copy</Button>
            <Box sx={{ flex: 1 }} />
            <Button variant="contained" onClick={() => setErrorDialog(null)}>
              Close
            </Button>
          </DialogActions>
        </Dialog>
      )}
      </Box>
    </ThemeProvider>
  );
}

function KeybindingsDialog({
  config,
  onClose,
  onSave,
}: {
  config: Record<string, unknown>;
  onClose: () => void;
  onSave: (next: Record<string, unknown>) => void;
}) {
  const [bindings, setBindings] = useState<Record<string, string>>(keysFromConfig(config));
  const [capturing, setCapturing] = useState<string | null>(null);

  useEffect(() => {
    if (!capturing) return;
    const h = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.key === "Escape") {
        setCapturing(null);
        return;
      }
      if (MODIFIER_KEYS.includes(e.key)) return; // wait for the non-modifier key
      setBindings((b) => ({ ...b, [capturing]: eventToKey(e) }));
      setCapturing(null);
    };
    window.addEventListener("keydown", h, true);
    return () => window.removeEventListener("keydown", h, true);
  }, [capturing]);

  function save() {
    const ide = { ...((config.ide as Record<string, unknown>) || {}) };
    ide.keys = bindings;
    onSave({ ...config, ide });
  }

  return (
    <Dialog open onClose={onClose} maxWidth="xs" fullWidth>
      <DialogTitle>Debugger keybindings</DialogTitle>
      <DialogContent dividers>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          Click a key, then press the shortcut. Esc cancels capture.
        </Typography>
        <Box className="keybinds">
          {KEY_ACTIONS.map((a) => (
            <div key={a.action} className="kb-row">
              <span>{a.label}</span>
              <Button
                size="small"
                variant={capturing === a.action ? "contained" : "outlined"}
                onClick={() => setCapturing(a.action)}
              >
                {capturing === a.action ? "press a key…" : bindings[a.action] || "—"}
              </Button>
            </div>
          ))}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => setBindings({ ...DEFAULT_KEYS })}>Reset to defaults</Button>
        <Box sx={{ flex: 1 }} />
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={save}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}

type TreeAction = "open" | "rename" | "remove" | "run" | "format" | "transpile";

function RemoveDialog({
  node,
  onClose,
  onConfirm,
}: {
  node: TreeNode;
  onClose: () => void;
  onConfirm: (recursive: boolean) => void;
}) {
  const nonEmptyDir = node.dir && (node.children?.length ?? 0) > 0;
  const [recursive, setRecursive] = useState(false);
  const blocked = nonEmptyDir && !recursive;
  return (
    <Dialog open onClose={onClose}>
      <DialogTitle>Remove {node.dir ? "directory" : "file"}</DialogTitle>
      <DialogContent>
        <Typography>
          Remove <code>{node.path}</code>?
        </Typography>
        {nonEmptyDir && (
          <FormControlLabel
            sx={{ mt: 1 }}
            control={<Checkbox checked={recursive} onChange={(e) => setRecursive(e.target.checked)} />}
            label="This directory is not empty — remove recursively"
          />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button color="error" variant="contained" disabled={blocked} onClick={() => onConfirm(recursive)}>
          Remove
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function TreeView({
  node,
  activePath,
  onOpen,
  onAction,
}: {
  node: TreeNode;
  activePath?: string;
  onOpen: (p: string) => void;
  onAction: (action: TreeAction, node: TreeNode) => void;
}) {
  const [open, setOpen] = useState(true);
  const [menu, setMenu] = useState<{ x: number; y: number } | null>(null);

  const onContextMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setMenu({ x: e.clientX, y: e.clientY });
  };
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "F2") {
      e.preventDefault();
      onAction("rename", node);
    } else if (e.key === "Delete") {
      e.preventDefault();
      onAction("remove", node);
    }
  };
  const close = () => setMenu(null);
  const act = (a: TreeAction) => {
    close();
    onAction(a, node);
  };

  const isGad = /\.gadt?$/.test(node.name);
  const contextMenu = (
    <Menu
      open={!!menu}
      onClose={close}
      anchorReference="anchorPosition"
      anchorPosition={menu ? { top: menu.y, left: menu.x } : undefined}
    >
      {!node.dir && <MenuItem onClick={() => act("open")}>Open</MenuItem>}
      {!node.dir && isGad && <MenuItem onClick={() => act("run")}>Run</MenuItem>}
      {!node.dir && isGad && <MenuItem onClick={() => act("format")}>Format</MenuItem>}
      {!node.dir && isGad && <MenuItem onClick={() => act("transpile")}>Transpile</MenuItem>}
      <MenuItem onClick={() => act("rename")}>Rename… (F2)</MenuItem>
      <MenuItem onClick={() => act("remove")}>Remove…</MenuItem>
    </Menu>
  );

  if (node.dir) {
    return (
      <div>
        <div
          className="node"
          tabIndex={0}
          onClick={() => setOpen((o) => !o)}
          onContextMenu={onContextMenu}
          onKeyDown={onKeyDown}
        >
          {open ? "📂" : "📁"} {node.name}
        </div>
        {contextMenu}
        {open && (
          <div className="children">
            {node.children?.map((c) => (
              <TreeView key={c.path} node={c} activePath={activePath} onOpen={onOpen} onAction={onAction} />
            ))}
          </div>
        )}
      </div>
    );
  }
  return (
    <div
      className={"node file" + (node.path === activePath ? " active" : "")}
      tabIndex={0}
      onClick={() => onOpen(node.path)}
      onContextMenu={onContextMenu}
      onKeyDown={onKeyDown}
    >
      📄 {node.name}
      {contextMenu}
    </div>
  );
}

function DocPanel({
  docs,
  onReload,
  onClose,
  onGoto,
}: {
  docs: DocComment[];
  onReload: () => void;
  onClose: () => void;
  onGoto: (line: number) => void;
}) {
  return (
    <div className="doc-panel">
      <div className="doc-head">
        <span>Doc comments</span>
        <span style={{ flex: 1 }} />
        <IconButton size="small" title="Reload" onClick={onReload}>
          <RefreshIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" title="Close" onClick={onClose}>
          <CloseIcon fontSize="small" />
        </IconButton>
      </div>
      <div className="doc-body">
        {docs.length === 0 && <div className="muted">No doc comments in this file.</div>}
        {docs.map((d, i) => (
          <div key={i} className="doc-entry">
            <div className="doc-entry-head" onClick={() => onGoto(d.line)} title={`Go to line ${d.line}`}>
              <span className={"doc-kind doc-kind-" + d.kind}>{d.kind}</span>
              <span className="doc-title">{d.title || `line ${d.line}`}</span>
            </div>
            <pre className="doc-content">{d.content}</pre>
          </div>
        ))}
      </div>
    </div>
  );
}

function EvaluatePanel({
  entries,
  onAdd,
  onUpdate,
  onRemove,
  onShowOutput,
  onCopy,
}: {
  entries: EvalEntry[];
  onAdd: (expr: string, repr: boolean) => void;
  onUpdate: (id: number, expr: string, repr: boolean) => void;
  onRemove: (id: number) => void;
  onShowOutput: (e: EvalEntry) => void;
  onCopy: (text: string) => void;
}) {
  const [expr, setExpr] = useState("");
  const [repr, setRepr] = useState(false);
  const [editing, setEditing] = useState<number | null>(null);

  const submit = () => {
    const e = expr.trim();
    if (!e) return;
    if (editing !== null) {
      onUpdate(editing, e, repr);
      setEditing(null);
    } else {
      onAdd(e, repr);
    }
    setExpr("");
    setRepr(false);
  };

  return (
    <div className="pane-body eval">
      <div className="eval-form">
        <TextField
          size="small"
          fullWidth
          placeholder="expression"
          value={expr}
          onChange={(ev) => setExpr(ev.target.value)}
          onKeyDown={(ev) => {
            if (ev.key === "Enter") submit();
          }}
        />
        <FormControlLabel
          control={<Checkbox size="small" checked={repr} onChange={(ev) => setRepr(ev.target.checked)} />}
          label="repr"
        />
        <IconButton size="small" title={editing !== null ? "Save" : "Add"} onClick={submit}>
          {editing !== null ? <RefreshIcon fontSize="small" /> : <AddIcon fontSize="small" />}
        </IconButton>
      </div>
      <table className="eval-list">
        <tbody>
          {entries.map((e) => (
            <tr key={e.id} className={e.error ? "err" : ""}>
              <td className="eval-expr">{e.repr ? "repr " : ""}{e.expr}</td>
              <td className="eval-val">{e.error || e.value}</td>
              <td className="eval-actions">
                <IconButton
                  size="small"
                  title="Edit"
                  onClick={() => {
                    setEditing(e.id);
                    setExpr(e.expr);
                    setRepr(e.repr);
                  }}
                >
                  <EditIcon sx={{ fontSize: 14 }} />
                </IconButton>
                <IconButton size="small" title="Output" onClick={() => onShowOutput(e)}>
                  <OutputIcon sx={{ fontSize: 14 }} />
                </IconButton>
                <IconButton size="small" title="Copy" onClick={() => onCopy(e.error || e.value)}>
                  <ContentCopyIcon sx={{ fontSize: 14 }} />
                </IconButton>
                <IconButton size="small" title="Remove" onClick={() => onRemove(e.id)}>
                  <DeleteIcon sx={{ fontSize: 14 }} />
                </IconButton>
              </td>
            </tr>
          ))}
          {entries.length === 0 && (
            <tr>
              <td className="muted">(no expressions — add one above)</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function BreakpointList({
  path,
  lines,
  meta,
  onEdit,
  onRemove,
}: {
  path?: string;
  lines: number[];
  meta: BreakpointMeta;
  onEdit: (line: number) => void;
  onRemove: (line: number) => void;
}) {
  if (!path) return <div className="muted">No file open.</div>;
  if (!lines.length) return <div className="muted">No breakpoints in {path.split("/").pop()}.</div>;
  return (
    <ul className="bp-list">
      {lines.map((l) => {
        const m = meta[l] || {};
        return (
          <li key={l} className={m.disabled ? "bp-disabled" : ""}>
            <span className="bp-entry" title="Click to edit condition" onClick={() => onEdit(l)}>
              line {l}
              {m.disabled ? " (disabled)" : ""}
              {m.condition ? <em className="bp-cond"> if {m.condition}</em> : null}
            </span>
            <button className="x" title="Remove" onClick={() => onRemove(l)}>
              ✕
            </button>
          </li>
        );
      })}
    </ul>
  );
}

function BreakpointDialog({
  line,
  initial,
  onClose,
  onSave,
}: {
  line: number;
  initial: { disabled?: boolean; condition?: string };
  onClose: () => void;
  onSave: (meta: { disabled?: boolean; condition?: string }) => void;
}) {
  const [disabled, setDisabled] = useState(!!initial.disabled);
  const [condition, setCondition] = useState(initial.condition ?? "");
  return (
    <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Breakpoint — line {line}</DialogTitle>
      <DialogContent dividers>
        <FormControlLabel
          control={<Checkbox checked={disabled} onChange={(e) => setDisabled(e.target.checked)} />}
          label="Disabled (ignore this breakpoint while debugging)"
        />
        <TextField
          fullWidth
          size="small"
          sx={{ mt: 1 }}
          label="Condition (Gad expression)"
          placeholder="e.g. i > 10"
          helperText="Pauses only when the expression is truthy (!value.IsFalsy()). Locals are in scope."
          value={condition}
          onChange={(e) => setCondition(e.target.value)}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={() => onSave({ disabled, condition })}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
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
    <Dialog open onClose={onCancel} maxWidth="sm" fullWidth>
      <DialogTitle>
        {kind === "debug" ? "Debug" : "Run"} {tab.path}
      </DialogTitle>
      <DialogContent dividers>
        <TextField
          label="Arguments (one per line)"
          multiline
          minRows={3}
          fullWidth
          margin="dense"
          value={args}
          onChange={(e) => setArgs(e.target.value)}
        />
        {kind === "debug" && (
          <FormControlLabel
            control={<Checkbox checked={entry} onChange={(e) => setEntry(e.target.checked)} />}
            label="Stop on entry (set breakpoints by clicking the gutter)"
          />
        )}
        <Typography variant="subtitle2" sx={{ mt: 1 }}>
          Builtin modules (checked = enabled)
        </Typography>
        <Box className="mods">
          {modules.map((m) => (
            <FormControlLabel
              key={m.name}
              control={<Checkbox size="small" checked={!disabled.includes(m.name)} onChange={() => toggle(m.name)} />}
              label={m.name + (m.unsafe ? " (unsafe)" : "")}
            />
          ))}
        </Box>
        <FormControlLabel
          control={<Checkbox checked={safe} onChange={(e) => setSafe(e.target.checked)} />}
          label="Safe mode (disable unsafe modules)"
        />
        <TextField
          label="Save stdout+stderr to file (optional)"
          fullWidth
          margin="dense"
          value={saveOut}
          onChange={(e) => setSaveOut(e.target.value)}
          placeholder="output.log"
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        {kind === "debug" ? (
          <Button variant="contained" onClick={() => onDebug(cfg(), entry)}>
            Start Debug
          </Button>
        ) : (
          <Button variant="contained" onClick={() => onRun(cfg())}>
            Run
          </Button>
        )}
      </DialogActions>
    </Dialog>
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
  const transpile = (config.transpile as Record<string, unknown>) || {};
  // Checked = expanded layout (default, no key); unchecked writes no-…: true.
  const [expanded, setExpanded] = useState<Record<string, boolean>>(
    Object.fromEntries(NEWLINE_FLAGS.map(([k]) => [k, fmt[k] !== true])),
  );
  const [backup, setBackup] = useState(fmt.backup === true);
  // Transpile options (.gad.yaml → transpile). Empty fields fall back to the
  // built-in defaults on the backend, so we keep them as plain strings here.
  const [writeFunc, setWriteFunc] = useState(String(transpile.writeFunc ?? ""));
  const [rawStart, setRawStart] = useState(String(transpile.rawStrFuncStart ?? ""));
  const [rawEnd, setRawEnd] = useState(String(transpile.rawStrFuncEnd ?? ""));

  function save() {
    const fmtObj: Record<string, unknown> = { ...fmt };
    for (const [k] of NEWLINE_FLAGS) {
      if (expanded[k]) delete fmtObj[k];
      else fmtObj[k] = true;
    }
    if (backup) fmtObj.backup = true;
    else delete fmtObj.backup;

    const trObj: Record<string, unknown> = { ...transpile };
    const setOrDel = (k: string, v: string) => {
      if (v.trim() === "") delete trObj[k];
      else trObj[k] = v;
    };
    setOrDel("writeFunc", writeFunc);
    setOrDel("rawStrFuncStart", rawStart);
    setOrDel("rawStrFuncEnd", rawEnd);

    const next: Record<string, unknown> = { ...config, fmt: fmtObj };
    if (Object.keys(trObj).length > 0) next.transpile = trObj;
    else delete next.transpile;
    onSave(next);
  }

  return (
    <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Settings</DialogTitle>
      <DialogContent dividers>
        <Typography variant="subtitle2">Formatter (.gad.yaml → fmt)</Typography>
        <Box sx={{ display: "flex", flexDirection: "column" }}>
          {NEWLINE_FLAGS.map(([k, label]) => (
            <FormControlLabel
              key={k}
              control={
                <Checkbox
                  checked={expanded[k]}
                  onChange={(e) => setExpanded((s) => ({ ...s, [k]: e.target.checked }))}
                />
              }
              label={label}
            />
          ))}
          <FormControlLabel
            control={<Checkbox checked={backup} onChange={(e) => setBackup(e.target.checked)} />}
            label="Keep .backup on format"
          />
        </Box>

        <Typography variant="subtitle2" sx={{ mt: 2 }}>
          Transpile (.gad.yaml → transpile)
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Applied to <code>.gad</code>/<code>.gadt</code> transpile. Leave blank for defaults.
        </Typography>
        <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5, mt: 1 }}>
          <TextField
            size="small"
            label="Write function"
            placeholder="write"
            value={writeFunc}
            onChange={(e) => setWriteFunc(e.target.value)}
          />
          <TextField
            size="small"
            label="Raw-string func start"
            placeholder="rawstr("
            value={rawStart}
            onChange={(e) => setRawStart(e.target.value)}
          />
          <TextField
            size="small"
            label="Raw-string func end"
            placeholder=";cast)"
            value={rawEnd}
            onChange={(e) => setRawEnd(e.target.value)}
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={save}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
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
.font-control{display:flex;align-items:center;gap:.25rem}
.font-control .font-size{color:var(--muted);font-size:.8rem;min-width:2.6rem;text-align:center}
.dbgbar{display:flex;gap:.3rem}
.editor-host{flex:1;min-height:0;display:flex}
.doc-panel{width:320px;min-width:200px;border-left:1px solid var(--border);background:var(--panel);display:flex;flex-direction:column;overflow:hidden}
.doc-head{display:flex;align-items:center;gap:.3rem;padding:.3rem .6rem;border-bottom:1px solid var(--border);font-size:.72rem;text-transform:uppercase;color:var(--muted);letter-spacing:.05em}
.doc-body{flex:1;overflow:auto;padding:.4rem .6rem}
.doc-entry{margin-bottom:.7rem}
.doc-entry-head{display:flex;align-items:center;gap:.4rem;cursor:pointer}
.doc-entry-head:hover .doc-title{color:var(--accent)}
.doc-title{font-family:ui-monospace,monospace;font-size:.82rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.doc-kind{font-size:.62rem;text-transform:uppercase;padding:0 .3rem;border-radius:3px;background:var(--code-bg,rgba(125,125,125,.18));color:var(--muted)}
.doc-kind-root{background:var(--accent);color:#fff}
.doc-content{margin:.2rem 0 0;white-space:pre-wrap;word-break:break-word;font-size:.82rem;color:var(--fg)}
.editor-host>div{flex:1;min-width:0}
.editor-host .empty{margin:auto;color:var(--muted)}
.panes{height:200px;border-top:1px solid var(--border);background:var(--panel);display:flex;flex-direction:column;resize:vertical;overflow:hidden}
.pane-tabs{display:flex;gap:.3rem;align-items:center;padding:.25rem .6rem;border-bottom:1px solid var(--border)}
.pane-tabs button.on{background:var(--accent);color:#fff}
.panes .pane-body{flex:1;overflow:auto;margin:0;padding:.5rem .8rem;white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:.85rem}
.frame{padding:.1rem .3rem;border-radius:4px}
.frame:hover{background:var(--code-bg,rgba(125,125,125,.12))}
.frame.selected{background:var(--accent);color:#fff}
.frame.selected .muted{color:rgba(255,255,255,.8)}
.frame .fn{font-weight:600}
.locals-head{margin-bottom:.3rem;font-size:.8rem}
.muted{color:var(--muted)}
table.locals td{padding:.1rem .5rem;border-bottom:1px solid var(--border);font-family:ui-monospace,monospace}
table.locals td.locals-copy{padding:0;width:1.6rem;text-align:right;opacity:0;transition:opacity .1s}
table.locals tr:hover td.locals-copy{opacity:.8}
.eval{display:flex;flex-direction:column;gap:.4rem}
.eval-form{display:flex;align-items:center;gap:.4rem;position:sticky;top:0;background:var(--panel);padding-bottom:.3rem}
table.eval-list{width:100%;border-collapse:collapse}
table.eval-list td{padding:.15rem .4rem;border-bottom:1px solid var(--border);font-family:ui-monospace,monospace;vertical-align:top}
table.eval-list td.eval-expr{white-space:nowrap;color:var(--muted)}
table.eval-list td.eval-val{white-space:pre-wrap;word-break:break-word}
table.eval-list tr.err td.eval-val{color:#e5484d}
table.eval-list td.eval-actions{width:7rem;text-align:right;white-space:nowrap;opacity:.3;transition:opacity .1s}
table.eval-list tr:hover td.eval-actions{opacity:.9}
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
.bp-list .bp-entry{cursor:pointer}
.bp-list .bp-entry:hover{color:var(--accent)}
.bp-list li.bp-disabled{opacity:.5}
.bp-list .bp-cond{color:var(--muted);font-style:italic}
.modal-bg{position:fixed;inset:0;background:rgba(0,0,0,.4);display:flex;align-items:center;justify-content:center;z-index:50}
.modal{background:var(--panel);border:1px solid var(--border);border-radius:10px;padding:1rem 1.2rem;min-width:380px;max-width:90vw;max-height:85vh;overflow:auto}
.modal h3{margin:.2rem 0 .8rem}
.modal .row{display:flex;flex-direction:column;gap:.25rem;margin:.5rem 0}
.mods{display:grid;grid-template-columns:1fr 1fr;gap:0 .8rem;max-height:180px;overflow:auto}
.keybinds{display:flex;flex-direction:column;gap:.4rem;margin:.5rem 0}
.kb-row{display:flex;align-items:center;justify-content:space-between;gap:1rem}
.kb-row button{min-width:7rem;font-family:ui-monospace,monospace}
.kb-row button.capturing{outline:2px solid var(--accent)}
    `}</style>
  );
}
