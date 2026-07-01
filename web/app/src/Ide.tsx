import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
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
  Tab,
  Tabs,
  TextField,
  ThemeProvider,
  Toolbar,
  Tooltip,
  Typography,
  createTheme,
} from "@mui/material";
import AddIcon from "@mui/icons-material/Add";
import AddLinkIcon from "@mui/icons-material/AddLink";
import VisibilityIcon from "@mui/icons-material/Visibility";
import VisibilityOffIcon from "@mui/icons-material/VisibilityOff";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import RefreshIcon from "@mui/icons-material/Refresh";
import UndoIcon from "@mui/icons-material/Undo";
import RedoIcon from "@mui/icons-material/Redo";
import EditIcon from "@mui/icons-material/Edit";
import DeleteIcon from "@mui/icons-material/Delete";
import OutputIcon from "@mui/icons-material/Notes";
import AccountTreeIcon from "@mui/icons-material/AccountTree";
import ViewQuiltIcon from "@mui/icons-material/ViewQuilt";
import TuneIcon from "@mui/icons-material/Tune";
import { DockviewReact, type DockviewApi, type DockviewReadyEvent, type IDockviewPanelProps } from "dockview-react";
import "dockview-react/dist/styles/dockview.css";

/** copyText writes text to the clipboard, ignoring failures (e.g. no permission). */
function copyText(text: string): void {
  void navigator.clipboard?.writeText(text).catch(() => {});
}
import { Editor, type EditorHandle, type EditorLanguage } from "./Editor";

/** Map a workspace file path to its CodeMirror language. */
function langForPath(path: string): EditorLanguage {
  const ext = (path.split(".").pop() ?? "").toLowerCase();
  switch (ext) {
    case "json": return "json";
    case "yaml": case "yml": return "yaml";
    case "html": case "htm": return "html";
    case "css": return "css";
    case "scss": return "scss";
    case "js": case "mjs": case "cjs": return "javascript";
    case "ts": case "mts": case "cts": return "typescript";
    case "jsx": return "jsx";
    case "tsx": return "tsx";
    case "gad": case "gadt": return "gad";
    case "md": case "mdx": return "markdown";
    default: return "text";
  }
}
import { InspectDialog, type InspectFn } from "./TreeNavigator";
import { renderDocMarkdown } from "./docMarkdown";
import { GadInput } from "./GadInput";
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

interface OpenTab {
  path: string;
  content: string;
  saved: boolean;
  runCfg: RunConfig;
}


interface EvalEntry {
  id: number;
  expr: string;
  repr: boolean;
  value: string;
  error: string;
}

const emptyRunCfg = (): RunConfig => ({ args: [], disabled: [], safe: false, saveOut: "", saveStdout: "", saveStderr: "", combine: false });

const DEFAULT_KEYS: Record<string, string> = {
  continue: "F9",
  stepOver: "F8",
  stepInto: "F7",
  stepOut: "Shift+F8",
};

const KEY_ACTIONS: { action: string; cmd: string; label: string }[] = [
  { action: "continue", cmd: "continue", label: "Resume (next breakpoint)" },
  { action: "stepOver", cmd: "next", label: "Step over" },
  { action: "stepInto", cmd: "stepIn", label: "Step into" },
  { action: "stepOut", cmd: "stepOut", label: "Step out" },
];

const MODIFIER_KEYS = ["Shift", "Control", "Alt", "Meta"];

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

// ---------------------------------------------------------------------------
// Shared IDE context — all panels consume this
// ---------------------------------------------------------------------------

type TreeAction = "open" | "rename" | "remove" | "run" | "format" | "transpile";

interface IdeShared {
  // theme
  dark: boolean;
  toggleTheme: () => void;
  // workspace tree
  tree: TreeNode | null;
  showHidden: boolean;
  setShowHidden: (v: boolean | ((p: boolean) => boolean)) => void;
  setFetchDialog: (v: boolean) => void;
  openFile: (path: string) => Promise<void>;
  treeAction: (action: TreeAction, node: TreeNode) => Promise<void>;
  refreshTree: () => Promise<void>;
  // tabs
  tabs: OpenTab[];
  active: number;
  setActive: (i: number) => void;
  activeTab: OpenTab | null;
  closeTab: (i: number) => void;
  onEdit: (v: string) => void;
  // editor actions
  save: () => Promise<void>;
  format: () => Promise<void>;
  reloadFile: () => Promise<void>;
  editorRef: React.RefObject<EditorHandle>;
  diagnose: import("@gad-lang/codemirror-gad").DiagnoseFn;
  fontSize: number;
  setFontSize: (px: number) => void;
  // debug
  debug: { session: string; path: string } | null;
  debugLoc: { line: number; column: number } | null;
  dbgCommand: (cmd: string) => Promise<void>;
  keys: Record<string, string>;
  startDebugFromDialog: (tab: OpenTab, stopOnEntry: boolean) => Promise<void>;
  // breakpoints
  bpFor: (path?: string) => number[];
  bpMetaFor: (path?: string) => BreakpointMeta;
  allBreakpoints: () => Record<string, number[]>;
  setBreakpoints: (path: string, lines: number[]) => void;
  setBpDialog: (v: { path: string; line: number } | null) => void;
  // output pane
  outChunks: { stream: "out" | "err"; text: string }[];
  outMode: "combined" | "split";
  setOutMode: (m: "combined" | "split") => void;
  clearOut: () => void;
  // call stack / locals
  stack: DebugResponse["frames"];
  locals: DebugResponse["locals"];
  selectedFrame: number;
  setSelectedFrame: (i: number) => void;
  onFrameClick: (i: number, f: { file: string; line: number; column: number }) => void;
  gotoFrame: (file: string, line: number, column: number) => Promise<void>;
  // evaluate panel
  evals: EvalEntry[];
  setEvals: React.Dispatch<React.SetStateAction<EvalEntry[]>>;
  evalOne: (entry: EvalEntry) => Promise<EvalEntry>;
  addEval: (expr: string, repr: boolean) => Promise<void>;
  // run / debug
  runActive: () => void;
  debugActive: () => void;
  // dialogs
  setDialog: (d: null | { kind: "run" | "debug"; tab: OpenTab }) => void;
  setInspectTarget: (t: { title: string; expr: string } | null) => void;
  setOutputDialog: (d: { title: string; text: string } | null) => void;
  // docs panel
  docs: DocComment[];
  reloadDocs: () => Promise<void>;
  docPanelOpen: boolean;
  toggleDocsPanel: () => void;
  // modules / config
  modules: ModuleInfo[];
  // status
  status: string;
}

const IdeCtx = createContext<IdeShared>({} as IdeShared);
const useIde = () => useContext(IdeCtx);

// ---------------------------------------------------------------------------
// Panel: Explorer (left sidebar)
// ---------------------------------------------------------------------------

function ExplorerPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  return (
    <aside className="ide-sidebar">
      <div className="side-head">
        <span>Explorer</span>
        <span style={{ flex: 1 }} />
        <IconButton
          size="small"
          title={ide.showHidden ? "Hide hidden files" : "Show hidden files"}
          onClick={() => ide.setShowHidden((v) => !v)}
        >
          {ide.showHidden ? <VisibilityIcon fontSize="small" /> : <VisibilityOffIcon fontSize="small" />}
        </IconButton>
        <IconButton size="small" title="Get file from web" onClick={() => ide.setFetchDialog(true)}>
          <AddLinkIcon fontSize="small" />
        </IconButton>
        <IconButton
          size="small"
          title="New file"
          onClick={async () => {
            const name = prompt("New file path (relative to workspace):", "untitled.gad");
            if (!name) return;
            await ideApi.mkfile(name);
            await ide.refreshTree();
            ide.openFile(name);
          }}
        >
          <AddIcon fontSize="small" />
        </IconButton>
      </div>
      <div className="tree">
        {ide.tree?.children?.map((n) => (
          <TreeView
            key={n.path}
            node={n}
            activePath={ide.activeTab?.path}
            onOpen={ide.openFile}
            onAction={ide.treeAction}
          />
        ))}
      </div>
    </aside>
  );
}

// ---------------------------------------------------------------------------
// Panel: Editor (center — tabbar + toolbar + editor)
// ---------------------------------------------------------------------------

function EditorPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  const { dark, debugLoc, keys } = ide;
  const debug = ide.debug;

  return (
    <section className="ide-center">
      <div className="tabbar">
        {ide.tabs.map((t, i) => (
          <div
            key={t.path}
            className={"tab" + (i === ide.active ? " active" : "")}
            onClick={() => ide.setActive(i)}
          >
            <span>
              {t.path.split("/").pop()}
              {t.saved ? "" : " •"}
            </span>
            <span
              className="x"
              onClick={(e) => { e.stopPropagation(); ide.closeTab(i); }}
            >
              ✕
            </span>
          </div>
        ))}
      </div>

      <Toolbar variant="dense" className="toolbar" disableGutters sx={{ gap: 1, minHeight: 44 }}>
        <Button size="small" variant="outlined" onClick={ide.save} disabled={!ide.activeTab}>Save</Button>
        <Button size="small" variant="outlined" onClick={ide.format} disabled={!ide.activeTab}>Format</Button>
        <Tooltip title="Reload from disk">
          <span>
            <IconButton size="small" onClick={ide.reloadFile} disabled={!ide.activeTab}>
              <RefreshIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
        <Tooltip title="Undo">
          <span>
            <IconButton size="small" onClick={() => ide.editorRef.current?.undo()} disabled={!ide.activeTab}>
              <UndoIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
        <Tooltip title="Redo">
          <span>
            <IconButton size="small" onClick={() => ide.editorRef.current?.redo()} disabled={!ide.activeTab}>
              <RedoIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
        <Button
          size="small" variant="contained" color="success"
          onClick={() => ide.runActive()}
          disabled={!ide.activeTab}
        >
          Run ▶
        </Button>
        <Button
          size="small" variant="contained" color="warning"
          onClick={() => ide.debugActive()}
          disabled={!ide.activeTab}
        >
          Debug 🐞
        </Button>
        <Tooltip title="Run / Debug settings">
          <span>
            <IconButton
              size="small"
              onClick={() => ide.activeTab && ide.setDialog({ kind: "run", tab: ide.activeTab })}
              disabled={!ide.activeTab}
            >
              <TuneIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
        {debug && (
          <Box className="dbgbar">
            <Tooltip title={`Resume (${keys.continue})`}>
              <Button size="small" onClick={() => ide.dbgCommand("continue")}>Continue ({keys.continue})</Button>
            </Tooltip>
            <Tooltip title={`Step over (${keys.stepOver})`}>
              <Button size="small" onClick={() => ide.dbgCommand("next")}>Step Over ({keys.stepOver})</Button>
            </Tooltip>
            <Tooltip title={`Step into (${keys.stepInto})`}>
              <Button size="small" onClick={() => ide.dbgCommand("stepIn")}>Step In ({keys.stepInto})</Button>
            </Tooltip>
            <Tooltip title={`Step out (${keys.stepOut})`}>
              <Button size="small" onClick={() => ide.dbgCommand("stepOut")}>Step Out ({keys.stepOut})</Button>
            </Tooltip>
            <Button size="small" color="error" onClick={() => ide.dbgCommand("stop")}>Stop</Button>
          </Box>
        )}
        <Box sx={{ flex: 1 }} />
        <Tooltip title="Toggle doc-comments panel">
          <Button
            size="small"
            variant={ide.docPanelOpen ? "contained" : "outlined"}
            onClick={ide.toggleDocsPanel}
          >
            Docs
          </Button>
        </Tooltip>
        <Box className="font-control" title="Editor font size">
          <Button size="small" onClick={() => ide.setFontSize(ide.fontSize - 1)}>A−</Button>
          <span className="font-size">{ide.fontSize}px</span>
          <Button size="small" onClick={() => ide.setFontSize(ide.fontSize + 1)}>A+</Button>
        </Box>
        <Typography variant="caption" color="text.secondary">{ide.status}</Typography>
      </Toolbar>

      <div className="editor-host">
        {ide.activeTab ? (
          <Editor
            key={ide.activeTab.path}
            ref={ide.editorRef}
            initialDoc={ide.activeTab.content}
            language={langForPath(ide.activeTab.path)}
            diagnose={langForPath(ide.activeTab.path) === "gad" ? ide.diagnose : undefined}
            dark={dark}
            onChange={ide.onEdit}
            breakpoints={ide.bpFor(ide.activeTab.path)}
            onBreakpointsChange={(lines) => ide.setBreakpoints(ide.activeTab!.path, lines)}
            fontSize={ide.fontSize}
            debugLine={debug && debug.path === ide.activeTab.path ? debugLoc?.line : undefined}
            debugColumn={debug && debug.path === ide.activeTab.path ? debugLoc?.column : undefined}
            locals={debug && debug.path === ide.activeTab.path ? ide.locals : undefined}
            onInspectVar={(name) => ide.setInspectTarget({ title: name, expr: name })}
          />
        ) : (
          <div className="empty">Open a file from the explorer</div>
        )}
      </div>
    </section>
  );
}

// ---------------------------------------------------------------------------
// Bottom panels — each is its own dockview panel (tab)
// Tab order in default layout: [Call Stack | Locals | Breakpoints | Evaluate | Output]
// ---------------------------------------------------------------------------

function OutputTextPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  const { outChunks, outMode } = ide;
  return (
    <div className="panes-dockview">
      <div className="pane-toolbar">
        <button className={outMode === "combined" ? "on" : ""} title="Combined stdout+stderr" onClick={() => ide.setOutMode("combined")}>Combined</button>
        <button className={outMode === "split" ? "on" : ""} title="Split stdout / stderr" onClick={() => ide.setOutMode("split")}>Split</button>
        <button onClick={ide.clearOut}>Clear</button>
      </div>
      {outMode === "combined" && (
        <pre className="pane-body out-log">
          {outChunks.length === 0
            ? <span className="muted">(no output)</span>
            : outChunks.map((c, i) => (
              <span key={i} className={c.stream === "err" ? "out-err" : undefined}>{c.text}</span>
            ))}
        </pre>
      )}
      {outMode === "split" && (
        <div className="pane-body out-split">
          <div className="out-col">
            <div className="out-col-head">stdout</div>
            <pre>{outChunks.filter((c) => c.stream === "out").map((c) => c.text).join("")}</pre>
          </div>
          <div className="out-col">
            <div className="out-col-head out-err">stderr</div>
            <pre className="out-err">{outChunks.filter((c) => c.stream === "err").map((c) => c.text).join("")}</pre>
          </div>
        </div>
      )}
    </div>
  );
}

function CallStackPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  const { stack, selectedFrame } = ide;
  return (
    <div className="panes-dockview">
      <div className="pane-body">
        {(stack || []).map((f, i) => (
          <div
            key={i}
            className={"frame" + (i === selectedFrame ? " selected" : "")}
            title={`${f.file}:${f.line}:${f.column} — click to inspect, double-click to open`}
            onClick={() => ide.onFrameClick(i, f)}
            style={{ cursor: "pointer" }}
          >
            <span className="fn">{f.name || "main"}</span>{" "}
            <span className="muted">
              {f.file ? `${f.file.split("/").pop()}:` : ""}{f.line}:{f.column}
            </span>
          </div>
        ))}
        {!(stack || []).length && <span className="muted">(no frames)</span>}
      </div>
    </div>
  );
}

function LocalsPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  const { stack, locals, selectedFrame } = ide;
  const frame = stack && stack[selectedFrame];
  const frameLocals = frame ? frame.locals : locals;
  return (
    <div className="panes-dockview">
      <div className="pane-body">
        {frame && (
          <div className="locals-head muted">
            {frame.name || "main"} — {frame.file ? frame.file.split("/").pop() + ":" : ""}{frame.line}:{frame.column}
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
                  <IconButton size="small" title="Inspect (tree)" onClick={() => ide.setInspectTarget({ title: v.name, expr: v.name })}>
                    <AccountTreeIcon sx={{ fontSize: 14 }} />
                  </IconButton>
                  <IconButton size="small" title="Copy value" onClick={() => copyText(v.value)}>
                    <ContentCopyIcon sx={{ fontSize: 14 }} />
                  </IconButton>
                </td>
              </tr>
            ))}
            {!(frameLocals || []).length && <tr><td className="muted">(no locals)</td></tr>}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function BreakpointsPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  const [bpScope, setBpScope] = useState<"current" | "all">("current");
  return (
    <div className="panes-dockview">
      <div className="pane-body">
        <div className="bp-scope">
          <button className={bpScope === "current" ? "on" : ""} onClick={() => setBpScope("current")}>Current file</button>
          <button className={bpScope === "all" ? "on" : ""} onClick={() => setBpScope("all")}>All</button>
        </div>
        {bpScope === "current" ? (
          <BreakpointList
            path={ide.activeTab?.path}
            lines={ide.bpFor(ide.activeTab?.path)}
            meta={ide.bpMetaFor(ide.activeTab?.path)}
            onEdit={(line) => ide.activeTab && ide.setBpDialog({ path: ide.activeTab.path, line })}
            onRemove={(line) => ide.activeTab && ide.setBreakpoints(ide.activeTab.path, ide.bpFor(ide.activeTab.path).filter((l) => l !== line))}
            onNavigate={(line) => ide.editorRef.current?.gotoLocation(line, 1)}
          />
        ) : (
          <BreakpointGroups
            all={ide.allBreakpoints()}
            onOpen={ide.openFile}
            onNavigate={(file, line) => void ide.gotoFrame(file, line, 1)}
            onRemove={(file, line) => ide.setBreakpoints(file, (ide.allBreakpoints()[file] || []).filter((l) => l !== line))}
          />
        )}
      </div>
    </div>
  );
}

function EvaluateDockPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  return (
    <EvaluatePanel
      entries={ide.evals}
      dark={ide.dark}
      onAdd={ide.addEval}
      onUpdate={async (id, expr, repr) => {
        const updated = await ide.evalOne({ id, expr, repr, value: "", error: "" });
        ide.setEvals((cur) => cur.map((e) => (e.id === id ? updated : e)));
      }}
      onRemove={(id) => ide.setEvals((cur) => cur.filter((e) => e.id !== id))}
      onShowOutput={(e) => ide.setOutputDialog({ title: e.expr, text: e.error || e.value })}
      onCopy={copyText}
      onInspect={(e) => ide.setInspectTarget({ title: e.expr, expr: e.expr })}
    />
  );
}

// ---------------------------------------------------------------------------
// Panel: Docs (left group tab — doc comments for active file)
// ---------------------------------------------------------------------------

function DocsPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  return (
    <div className="doc-panel dock-panel-fill">
      <div className="doc-body">
        {ide.docs.length === 0 && <div className="muted" style={{ padding: ".4rem" }}>No doc comments in this file.</div>}
        {ide.docs.map((d, i) => (
          <div key={i} className="doc-entry">
            <div className="doc-entry-head" onClick={() => ide.editorRef.current?.gotoLocation(d.line, 1)} title={`Go to line ${d.line}`}>
              <span className={"doc-kind doc-kind-" + d.kind}>{d.kind}</span>
              <span className="doc-title">{d.title || `line ${d.line}`}</span>
            </div>
            <div className="doc-content language-gad" dangerouslySetInnerHTML={{ __html: renderDocMarkdown(d.content) }} />
          </div>
        ))}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Panel: Markdown preview (left group tab — rendered .md preview)
// ---------------------------------------------------------------------------

function MdPreviewPanel(_: IDockviewPanelProps) {
  const ide = useIde();
  const content = ide.activeTab?.content ?? "";
  const html = useMemo(() => renderDocMarkdown(content), [content]);
  return (
    <div className="doc-panel dock-panel-fill">
      <div className="doc-body">
        <div className="doc-content language-gad" dangerouslySetInnerHTML={{ __html: html }} />
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Dockview panel registry
// ---------------------------------------------------------------------------

const DOCKVIEW_COMPONENTS = {
  explorer: ExplorerPanel,
  editor: EditorPanel,
  output: OutputTextPanel,
  callstack: CallStackPanel,
  locals: LocalsPanel,
  breakpoints: BreakpointsPanel,
  evaluate: EvaluateDockPanel,
  docs: DocsPanel,
  markdown: MdPreviewPanel,
} as const;

function setupDefaultLayout(api: DockviewApi) {
  api.addPanel({ id: "editor", component: "editor", title: "Editor" });
  const explorerPanel = api.addPanel({ id: "explorer", component: "explorer", title: "Explorer", position: { direction: "left", referencePanel: "editor" } });
  explorerPanel.api.group.api.setSize({ width: 200 });
  // Bottom panels — callstack/locals/breakpoints/evaluate tab to the left of output
  api.addPanel({ id: "callstack", component: "callstack", title: "Call Stack", position: { direction: "below", referencePanel: "editor" } });
  api.addPanel({ id: "locals", component: "locals", title: "Locals", position: { direction: "within", referencePanel: "callstack" } });
  api.addPanel({ id: "breakpoints", component: "breakpoints", title: "Breakpoints", position: { direction: "within", referencePanel: "callstack" } });
  api.addPanel({ id: "evaluate", component: "evaluate", title: "Evaluate", position: { direction: "within", referencePanel: "callstack" } });
  api.addPanel({ id: "output", component: "output", title: "Output", position: { direction: "within", referencePanel: "callstack" } });
  api.getPanel("callstack")?.api.setActive();
}

// ---------------------------------------------------------------------------
// Main Ide component
// ---------------------------------------------------------------------------

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
  const [docPanelOpen, setDocPanelOpen] = useState(false);
  const [docs, setDocs] = useState<DocComment[]>([]);
  const [inspectTarget, setInspectTarget] = useState<{ title: string; expr: string } | null>(null);
  const [modules, setModules] = useState<ModuleInfo[]>([]);
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [tabs, setTabs] = useState<OpenTab[]>([]);
  const [active, setActive] = useState(-1);
  const [outChunks, setOutChunks] = useState<{ stream: "out" | "err"; text: string }[]>([]);
  const [outMode, setOutMode] = useState<"combined" | "split">("combined");
  const pushOut = useCallback((stream: "out" | "err", text: string) => {
    if (text) setOutChunks((c) => [...c, { stream, text }]);
  }, []);
  const clearOut = useCallback(() => setOutChunks([]), []);
  const setOutput = useCallback(
    (v: string | ((prev: string) => string)) => {
      if (typeof v === "function") {
        setOutChunks((c) => [{ stream: "out", text: v(c.map((x) => x.text).join("")) }]);
      } else {
        setOutChunks(v ? [{ stream: "out", text: v }] : []);
      }
    },
    [],
  );
  const [stack, setStack] = useState<DebugResponse["frames"]>([]);
  const [locals, setLocals] = useState<DebugResponse["locals"]>([]);
  const [status, setStatus] = useState("");

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
  const [fetchDialog, setFetchDialog] = useState(false);
  const [selectedFrame, setSelectedFrame] = useState(0);
  const frameClickTimer = useRef<number | null>(null);

  const editorRef = useRef<EditorHandle>(null);
  const activeTab = active >= 0 ? tabs[active] : null;

  // Dockview API ref — available after onReady fires.
  const dockviewApiRef = useRef<DockviewApi | null>(null);
  // Keep config accessible in the onReady closure without re-running it.
  const configRef = useRef(config);
  configRef.current = config;

  const activateBottomPanel = useCallback((id: string) => {
    dockviewApiRef.current?.getPanel(id)?.api.setActive();
  }, []);

  const refreshTree = useCallback(
    async () => setTree(await ideApi.tree(showHidden)),
    [showHidden],
  );

  useEffect(() => { void refreshTree(); }, [refreshTree]);

  useEffect(() => {
    (async () => {
      try {
        const cfg = await ideApi.config();
        configRef.current = cfg;
        setConfig(cfg);
        setModules(await ideApi.modules());
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

  const bpSpecsFor = (path: string): BreakpointSpec[] => {
    const meta = bpMetaFor(path);
    return bpFor(path).map((line) => ({ line, ...(meta[line] || {}) }));
  };

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
      activateBottomPanel("locals");
    }, 250);
  }

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
    if (existing >= 0) { setActive(existing); return; }
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

  const inspectExpr: InspectFn = useCallback(
    async (expr: string) => {
      try {
        const res = await ideApi.inspect(
          debug
            ? { expr, session: debug.session }
            : { expr, source: editorRef.current?.getValue() ?? activeTab?.content ?? "", path: activeTab?.path },
        );
        return res.ok && res.inspect ? res.inspect : null;
      } catch {
        return null;
      }
    },
    [debug, activeTab],
  );

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

  useEffect(() => {
    if (debug) void evalAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debugLoc]);

  const reloadDocs = useCallback(async () => {
    if (!docPanelOpen) return;
    const src = editorRef.current?.getValue() ?? activeTab?.content ?? "";
    try {
      setDocs(await ideApi.doc(src));
    } catch {
      /* leave previous docs on a transient failure */
    }
  }, [docPanelOpen, activeTab]);

  useEffect(() => { void reloadDocs(); }, [reloadDocs, active]);

  useEffect(() => {
    if (!docPanelOpen || !activeTab || activeTab.saved) return;
    const t = window.setTimeout(() => void reloadDocs(), 5000);
    return () => window.clearTimeout(t);
  }, [docPanelOpen, activeTab, reloadDocs]);

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
    activateBottomPanel("output");
  }

  async function formatFile(path: string) {
    try {
      const data = await ideApi.read(path);
      const res = await ideApi.format(data.content);
      if (!res.ok) { showDiagnostics(res.diagnostics); setStatus("format failed: " + path); return; }
      await ideApi.write(path, res.source);
      setTabs((ts) => ts.map((t) => (t.path === path ? { ...t, content: res.source, saved: true } : t)));
      if (activeTab?.path === path) editorRef.current?.setValue(res.source);
      setStatus("formatted " + path);
    } catch (e) {
      reportError("Format failed", e);
    }
  }

  async function transpileFile(path: string) {
    try {
      const data = await ideApi.read(path);
      const res = await ideApi.transpile(data.content, path);
      if (!res.ok) { showDiagnostics(res.diagnostics); setStatus("transpile failed: " + path); return; }
      const out = path.endsWith(".gadt") ? path.slice(0, -1) : path.replace(/\.gad$/, ".transpiled.gad");
      await ideApi.write(out, res.source);
      await refreshTree();
      await openFile(out);
      setStatus("transpiled to " + out);
    } catch (e) {
      reportError("Transpile failed", e);
    }
  }

  const treeAction = useCallback(
    async (action: TreeAction, node: TreeNode) => {
      switch (action) {
        case "open": void openFile(node.path); break;
        case "rename": {
          const to = prompt("Rename to (path relative to workspace):", node.path);
          if (!to || to === node.path) return;
          try {
            await ideApi.rename(node.path, to);
            setTabs((ts) => ts.map((t) => (t.path === node.path ? { ...t, path: to } : t)));
            await refreshTree();
            setStatus("renamed to " + to);
          } catch (e) { reportError("Rename failed", e); }
          break;
        }
        case "remove": setRemoveTarget(node); break;
        case "format": await formatFile(node.path); break;
        case "run": await openFile(node.path); setPendingRunPath(node.path); break;
        case "transpile": await transpileFile(node.path); break;
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [refreshTree],
  );

  useEffect(() => {
    if (!pendingRunPath) return;
    const tab = tabs.find((t) => t.path === pendingRunPath);
    if (tab) { void doRun(tab, tab.runCfg); setPendingRunPath(null); }
  }, [pendingRunPath, tabs]);

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
    activateBottomPanel("output");
    try {
      const res = await ideApi.run({
        path: tab.path, source: content, args: cfg.args, disabled: cfg.disabled,
        safe: cfg.safe, saveOut: cfg.saveOut || undefined,
        saveStdout: cfg.saveStdout || undefined, saveStderr: cfg.saveStderr || undefined,
        combine: cfg.combine || undefined,
      });
      clearOut();
      pushOut("out", res.stdout || "");
      pushOut("err", res.stderr || "");
      if (res.ok && res.result) pushOut("out", "\n⇦ " + res.result + "\n");
      const diag = (res.diagnostics || []).map((d) => `${d.line}:${d.column} ${d.message}`).join("\n");
      pushOut("err", diag);
      setStatus(res.ok ? "done" : "error");
    } catch (e) {
      pushOut("err", String(e));
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
        source: content, breakpoints: bpFor(tab.path),
        breakpointSpecs: bpSpecsFor(tab.path), stopOnEntry, path: tab.path,
        args: cfg.args, disabled: cfg.disabled, safe: cfg.safe,
      });
      applyDebug(res, tab.path);
    } catch (e) {
      setOutput(String(e));
      setStatus("error");
    }
  }

  async function dbgCommand(command: string) {
    if (!debug) return;
    if (command === "stop") { setDebug(null); setDebugLoc(null); setStatus("stopped"); return; }
    const res = await ideApi.dbgCmd(debug.session, command);
    applyDebug(res, debug.path);
  }

  function applyDebug(res: DebugResponse, path: string) {
    pushOut("out", res.stdout || "");
    pushOut("err", res.stderr || "");
    if (res.state === "stopped") {
      const stopFile = res.file && res.file !== "(main)" ? res.file : path;
      setDebug({ session: res.session!, path: stopFile });
      setStack(res.frames || []);
      setLocals(res.locals || []);
      setSelectedFrame(0);
      setDebugLoc({ line: res.line ?? 0, column: res.column ?? 1 });
      setStatus(`stopped (${res.reason}) at ${stopFile}:${res.line}`);
      activateBottomPanel("callstack");
      if (stopFile && stopFile !== activeTab?.path) void openFile(stopFile);
    } else if (res.state === "terminated") {
      if (res.result) pushOut("out", "\n⇦ " + res.result + "\n");
      setDebug(null); setDebugLoc(null); setStatus("program exited");
    } else {
      pushOut("err", res.error || "debug error");
      if (res.diagnostics) showDiagnostics(res.diagnostics);
      setDebug(null); setDebugLoc(null);
    }
  }

  function runActive() {
    if (!activeTab) return;
    void doRun(activeTab, activeTab.runCfg);
  }

  function debugActive() {
    if (!activeTab) return;
    void startDebug(activeTab, false);
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

  useEffect(() => {
    if (!debug) return;
    const keys = keysFromConfig(config);
    const handler = (e: KeyboardEvent) => {
      const pressed = eventToKey(e);
      const hit = KEY_ACTIONS.find((a) => keys[a.action] === pressed);
      if (hit) { e.preventDefault(); dbgCommand(hit.cmd); }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [debug, config]);

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

  // -------------------------------------------------------------------------
  // Dockview: docs panel toggle and markdown panel management
  // -------------------------------------------------------------------------

  const saveLayout = useCallback((layout: unknown) => {
    setConfig((c) => {
      const ide = { ...((c.ide as Record<string, unknown>) || {}), panels: layout };
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }, []);

  const toggleDocsPanel = useCallback(() => {
    const api = dockviewApiRef.current;
    if (!api) return;
    const existing = api.getPanel("docs");
    if (existing) {
      existing.api.close();
      setDocPanelOpen(false);
    } else {
      const ref = api.getPanel("explorer");
      api.addPanel({
        id: "docs",
        component: "docs",
        title: "Docs",
        position: ref
          ? { direction: "within", referencePanel: "explorer" }
          : undefined,
      });
      api.getPanel("docs")?.api.setActive();
      setDocPanelOpen(true);
    }
  }, []);

  // Show / hide markdown preview panel when active file type changes.
  const prevIsMdRef = useRef(false);
  useEffect(() => {
    const isMd = (activeTab?.path.endsWith(".md") || activeTab?.path.endsWith(".mdx")) ?? false;
    const api = dockviewApiRef.current;
    if (!api) return;
    if (isMd && !prevIsMdRef.current) {
      if (!api.getPanel("markdown")) {
        const ref = api.getPanel("explorer");
        api.addPanel({
          id: "markdown",
          component: "markdown",
          title: "MD Preview",
          position: ref ? { direction: "within", referencePanel: "explorer" } : undefined,
        });
      }
      api.getPanel("markdown")?.api.setActive();
    } else if (!isMd && prevIsMdRef.current) {
      api.getPanel("markdown")?.api.close();
    }
    prevIsMdRef.current = isMd;
  }, [activeTab?.path]);

  // Reset all panels to the default layout.
  function resetPanels() {
    const api = dockviewApiRef.current;
    if (!api) return;
    setDocPanelOpen(false);
    api.clear();
    setupDefaultLayout(api);
    setConfig((c) => {
      const ide = { ...((c.ide as Record<string, unknown>) || {}) };
      delete ide.panels;
      const next = { ...c, ide };
      ideApi.saveConfig(next).catch(() => {});
      return next;
    });
  }

  const onDockviewReady = useCallback((event: DockviewReadyEvent) => {
    const api = event.api;
    dockviewApiRef.current = api;

    // Restore saved layout or apply default.
    const saved = (configRef.current?.ide as Record<string, unknown>)?.panels;
    let restored = false;
    if (saved) {
      try {
        api.fromJSON(saved as Parameters<typeof api.fromJSON>[0]);
        restored = true;
      } catch {
        /* fall through to default */
      }
    }
    if (!restored) setupDefaultLayout(api);

    // Persist layout on every structural change.
    const disposable = api.onDidLayoutChange(() => saveLayout(api.toJSON()));
    return () => disposable.dispose();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const muiTheme = useMemo(
    () => createTheme({ palette: { mode: dark ? "dark" : "light", primary: { main: dark ? "#8aa6ff" : "#3b5bdb" } } }),
    [dark],
  );

  // Build the context value — recreated every render; panels re-render
  // accordingly (acceptable for a tool of this complexity).
  const ideShared: IdeShared = {
    dark, toggleTheme,
    tree, showHidden, setShowHidden, setFetchDialog, openFile, treeAction, refreshTree,
    tabs, active, setActive, activeTab, closeTab, onEdit,
    save, format, reloadFile, editorRef, diagnose, fontSize, setFontSize,
    debug, debugLoc, dbgCommand, keys,
    startDebugFromDialog: startDebug,
    bpFor, bpMetaFor, allBreakpoints, setBreakpoints, setBpDialog,
    outChunks, outMode, setOutMode, clearOut,
    stack, locals, selectedFrame, setSelectedFrame, onFrameClick, gotoFrame,
    evals, setEvals, evalOne, addEval,
    runActive, debugActive,
    setDialog, setInspectTarget, setOutputDialog,
    docs, reloadDocs, docPanelOpen, toggleDocsPanel,
    modules, status,
  };

  return (
    <IdeCtx.Provider value={ideShared}>
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
              <Tooltip title="Reset panels to default layout">
                <Button size="small" startIcon={<ViewQuiltIcon />} onClick={resetPanels}>
                  Reset Panels
                </Button>
              </Tooltip>
              <Button size="small" onClick={() => setKeybinds(true)}>⌨ Keys</Button>
              <Button size="small" onClick={() => setSettings(true)}>⚙ Settings</Button>
              <IconButton size="small" onClick={toggleTheme} title="Toggle theme">
                {dark ? "☀" : "☾"}
              </IconButton>
            </Toolbar>
          </AppBar>

          <DockviewReact
            className={`ide-dockview ${dark ? "dockview-theme-dark" : "dockview-theme-light"}`}
            components={DOCKVIEW_COMPONENTS as never}
            onReady={onDockviewReady}
          />

          {dialog && (
            <RunDebugSettingsDialog
              tab={dialog.tab}
              modules={modules}
              onCancel={() => setDialog(null)}
              onRun={(cfg) => { setDialog(null); persistRunCfg(dialog.tab.path, cfg); void doRun(dialog.tab, cfg); }}
              onDebug={(cfg, entry) => {
                setDialog(null);
                persistRunCfg(dialog.tab.path, cfg);
                void startDebug(dialog.tab, entry);
              }}
            />
          )}
          {settings && (
            <SettingsDialog
              config={config}
              onClose={() => setSettings(false)}
              onSave={async (next) => { setConfig(next); await ideApi.saveConfig(next); setSettings(false); setStatus("settings saved"); }}
            />
          )}
          {keybinds && (
            <KeybindingsDialog
              config={config}
              onClose={() => setKeybinds(false)}
              onSave={async (next) => { setConfig(next); await ideApi.saveConfig(next); setKeybinds(false); setStatus("keybindings saved"); }}
            />
          )}
          {removeTarget && (
            <RemoveDialog node={removeTarget} onClose={() => setRemoveTarget(null)} onConfirm={confirmRemove} />
          )}
          {bpDialog && (
            <BreakpointDialog
              line={bpDialog.line}
              dark={dark}
              initial={bpMetaFor(bpDialog.path)[bpDialog.line] || {}}
              onClose={() => setBpDialog(null)}
              onSave={(meta) => { setBpMeta(bpDialog.path, bpDialog.line, meta); setBpDialog(null); }}
            />
          )}
          {inspectTarget && (
            <InspectDialog
              title={inspectTarget.title}
              rootExpr={inspectTarget.expr}
              inspect={inspectExpr}
              onClose={() => setInspectTarget(null)}
              onGotoSource={(file) => { setInspectTarget(null); void openFile(file); }}
            />
          )}
          {outputDialog && (
            <Dialog open onClose={() => setOutputDialog(null)} maxWidth="md" fullWidth>
              <DialogTitle>{outputDialog.title}</DialogTitle>
              <DialogContent dividers>
                <TextField
                  multiline fullWidth minRows={6} maxRows={20}
                  value={outputDialog.text}
                  slotProps={{ input: { readOnly: true, sx: { fontFamily: "ui-monospace, monospace", fontSize: ".85rem" } } }}
                />
              </DialogContent>
              <DialogActions>
                <Button onClick={() => copyText(outputDialog.text)}>Copy</Button>
                <Box sx={{ flex: 1 }} />
                <Button variant="contained" onClick={() => setOutputDialog(null)}>Close</Button>
              </DialogActions>
            </Dialog>
          )}
          {errorDialog && (
            <Dialog open onClose={() => setErrorDialog(null)} maxWidth="sm" fullWidth>
              <DialogTitle>{errorDialog.title}</DialogTitle>
              <DialogContent dividers>
                <Typography component="pre" sx={{ whiteSpace: "pre-wrap", fontFamily: "ui-monospace, monospace", fontSize: ".85rem", m: 0 }}>
                  {errorDialog.detail}
                </Typography>
              </DialogContent>
              <DialogActions>
                <Button onClick={() => copyText(errorDialog.detail)}>Copy</Button>
                <Box sx={{ flex: 1 }} />
                <Button variant="contained" onClick={() => setErrorDialog(null)}>Close</Button>
              </DialogActions>
            </Dialog>
          )}
          {fetchDialog && (
            <FetchFromWebDialog
              defaultDir={activeTab ? activeTab.path.split("/").slice(0, -1).join("/") : ""}
              onClose={() => setFetchDialog(false)}
              onFetch={async (url, path) => {
                try {
                  await ideApi.fetchUrl(url, path);
                  setFetchDialog(false);
                  await refreshTree();
                  await openFile(path);
                } catch (e) { reportError("Fetch failed", e); }
              }}
            />
          )}
        </Box>
      </ThemeProvider>
    </IdeCtx.Provider>
  );
}

// ---------------------------------------------------------------------------
// Dialog & utility components (unchanged from original)
// ---------------------------------------------------------------------------

function FetchFromWebDialog({
  defaultDir, onClose, onFetch,
}: {
  defaultDir: string;
  onClose: () => void;
  onFetch: (url: string, path: string) => void;
}) {
  const [url, setUrl] = useState("");
  const [name, setName] = useState("");
  const [dir, setDir] = useState(defaultDir);
  const resolvedPath = () => {
    const filename = name.trim() || (url.split("/").pop()?.split("?")[0] ?? "file");
    const d = dir.trim();
    return d ? `${d}/${filename}` : filename;
  };
  return (
    <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Get file from web</DialogTitle>
      <DialogContent dividers>
        <TextField fullWidth autoFocus size="small" label="URL" placeholder="https://example.com/file.gad" value={url} onChange={(e) => setUrl(e.target.value)} sx={{ mb: 2 }} />
        <TextField fullWidth size="small" label="Output filename (leave blank to use URL filename)" placeholder="file.gad" value={name} onChange={(e) => setName(e.target.value)} sx={{ mb: 2 }} />
        <TextField fullWidth size="small" label="Target directory (relative to workspace)" placeholder="e.g. samples" value={dir} onChange={(e) => setDir(e.target.value)} helperText={`Saves to: ${resolvedPath()}`} />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={!url.trim()} onClick={() => onFetch(url.trim(), resolvedPath())}>Download</Button>
      </DialogActions>
    </Dialog>
  );
}

function KeybindingsDialog({
  config, onClose, onSave,
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
      e.preventDefault(); e.stopPropagation();
      if (e.key === "Escape") { setCapturing(null); return; }
      if (MODIFIER_KEYS.includes(e.key)) return;
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
              <Button size="small" variant={capturing === a.action ? "contained" : "outlined"} onClick={() => setCapturing(a.action)}>
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
        <Button variant="contained" onClick={save}>Save</Button>
      </DialogActions>
    </Dialog>
  );
}

function RemoveDialog({
  node, onClose, onConfirm,
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
        <Typography>Remove <code>{node.path}</code>?</Typography>
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
        <Button color="error" variant="contained" disabled={blocked} onClick={() => onConfirm(recursive)}>Remove</Button>
      </DialogActions>
    </Dialog>
  );
}

function TreeView({
  node, activePath, onOpen, onAction,
}: {
  node: TreeNode;
  activePath?: string;
  onOpen: (p: string) => void;
  onAction: (action: TreeAction, node: TreeNode) => void;
}) {
  const [open, setOpen] = useState(true);
  const [menu, setMenu] = useState<{ x: number; y: number } | null>(null);
  const onContextMenu = (e: React.MouseEvent) => { e.preventDefault(); e.stopPropagation(); setMenu({ x: e.clientX, y: e.clientY }); };
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "F2") { e.preventDefault(); onAction("rename", node); }
    else if (e.key === "Delete") { e.preventDefault(); onAction("remove", node); }
  };
  const close = () => setMenu(null);
  const act = (a: TreeAction) => { close(); onAction(a, node); };
  const isGad = /\.gadt?$/.test(node.name);
  const contextMenu = (
    <Menu open={!!menu} onClose={close} anchorReference="anchorPosition" anchorPosition={menu ? { top: menu.y, left: menu.x } : undefined}>
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
        <div className="node" tabIndex={0} onClick={() => setOpen((o) => !o)} onContextMenu={onContextMenu} onKeyDown={onKeyDown}>
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
    <div className={"node file" + (node.path === activePath ? " active" : "")} tabIndex={0} onClick={() => onOpen(node.path)} onContextMenu={onContextMenu} onKeyDown={onKeyDown}>
      📄 {node.name}
      {contextMenu}
    </div>
  );
}

function EvaluatePanel({
  entries, dark, onAdd, onUpdate, onRemove, onShowOutput, onCopy, onInspect,
}: {
  entries: EvalEntry[];
  dark?: boolean;
  onAdd: (expr: string, repr: boolean) => void;
  onUpdate: (id: number, expr: string, repr: boolean) => void;
  onRemove: (id: number) => void;
  onShowOutput: (e: EvalEntry) => void;
  onCopy: (text: string) => void;
  onInspect: (e: EvalEntry) => void;
}) {
  const [expr, setExpr] = useState("");
  const [repr, setRepr] = useState(false);
  const [editing, setEditing] = useState<number | null>(null);
  const submit = () => {
    const e = expr.trim();
    if (!e) return;
    if (editing !== null) { onUpdate(editing, e, repr); setEditing(null); }
    else onAdd(e, repr);
    setExpr(""); setRepr(false);
  };
  return (
    <div className="pane-body eval">
      <div className="eval-form">
        <GadInput value={expr} onChange={setExpr} onSubmit={submit} dark={dark} placeholder="expression" />
        <FormControlLabel control={<Checkbox size="small" checked={repr} onChange={(ev) => setRepr(ev.target.checked)} />} label="repr" />
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
                <IconButton size="small" title="Edit" onClick={() => { setEditing(e.id); setExpr(e.expr); setRepr(e.repr); }}>
                  <EditIcon sx={{ fontSize: 14 }} />
                </IconButton>
                <IconButton size="small" title="Inspect (tree)" onClick={() => onInspect(e)}>
                  <AccountTreeIcon sx={{ fontSize: 14 }} />
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
            <tr><td className="muted">(no expressions — add one above)</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function BreakpointList({
  path, lines, meta, onEdit, onRemove, onNavigate,
}: {
  path?: string;
  lines: number[];
  meta: BreakpointMeta;
  onEdit: (line: number) => void;
  onRemove: (line: number) => void;
  onNavigate: (line: number) => void;
}) {
  if (!path) return <div className="muted">No file open.</div>;
  if (!lines.length) return <div className="muted">No breakpoints in {path.split("/").pop()}.</div>;
  return (
    <ul className="bp-list">
      {lines.map((l) => {
        const m = meta[l] || {};
        return (
          <li key={l} className={m.disabled ? "bp-disabled" : ""}>
            <span className="bp-entry" title="Click to edit condition · Double-click to go to line" onClick={() => onEdit(l)} onDoubleClick={() => onNavigate(l)}>
              line {l}{m.disabled ? " (disabled)" : ""}
              {m.condition ? <em className="bp-cond"> if {m.condition}</em> : null}
            </span>
            <button className="x" title="Remove" onClick={() => onRemove(l)}>✕</button>
          </li>
        );
      })}
    </ul>
  );
}

function BreakpointDialog({
  line, dark, initial, onClose, onSave,
}: {
  line: number;
  dark?: boolean;
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
        <FormControlLabel control={<Checkbox checked={disabled} onChange={(e) => setDisabled(e.target.checked)} />} label="Disabled (ignore this breakpoint while debugging)" />
        <Typography variant="caption" sx={{ display: "block", mt: 1, mb: 0.5, color: "text.secondary" }}>
          Condition (Gad expression) — pauses only when truthy. Locals are in scope.
        </Typography>
        <GadInput value={condition} onChange={setCondition} onSubmit={() => onSave({ disabled, condition })} dark={dark} placeholder="e.g. i > 10" />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={() => onSave({ disabled, condition })}>Save</Button>
      </DialogActions>
    </Dialog>
  );
}

function BreakpointGroups({
  all, onOpen, onNavigate, onRemove,
}: {
  all: Record<string, number[]>;
  onOpen: (path: string) => void;
  onNavigate: (file: string, line: number) => void;
  onRemove: (file: string, line: number) => void;
}) {
  const files = Object.keys(all).filter((f) => (all[f] || []).length);
  if (!files.length) return <div className="muted">No breakpoints set.</div>;
  return (
    <div>
      {files.sort().map((file) => (
        <div key={file} className="bp-group">
          <div className="bp-file" onClick={() => onOpen(file)} title="Click to open file">{file}</div>
          <ul className="bp-list">
            {[...all[file]].sort((a, b) => a - b).map((l) => (
              <li key={l}>
                <span className="bp-entry" title="Double-click to go to line" onDoubleClick={() => onNavigate(file, l)}>line {l}</span>
                <button className="x" title="Remove" onClick={() => onRemove(file, l)}>✕</button>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}

function RunDebugSettingsDialog({
  tab, modules, onCancel, onRun, onDebug,
}: {
  tab: OpenTab;
  modules: ModuleInfo[];
  onCancel: () => void;
  onRun: (cfg: RunConfig) => void;
  onDebug: (cfg: RunConfig, stopOnEntry: boolean) => void;
}) {
  const [tabIdx, setTabIdx] = useState(0);
  const [args, setArgs] = useState(tab.runCfg.args.join("\n"));
  const [disabled, setDisabled] = useState<string[]>(tab.runCfg.disabled);
  const [safe, setSafe] = useState(tab.runCfg.safe);
  const [saveStdout, setSaveStdout] = useState(tab.runCfg.saveStdout ?? tab.runCfg.saveOut ?? "");
  const [saveStderr, setSaveStderr] = useState(tab.runCfg.saveStderr ?? "");
  const [combine, setCombine] = useState(tab.runCfg.combine ?? false);
  const [entry, setEntry] = useState(false);
  const toggle = (name: string) => setDisabled((d) => (d.includes(name) ? d.filter((n) => n !== name) : [...d, name]));
  const cfg = (): RunConfig => ({
    args: args.split("\n").map((s) => s.trim()).filter(Boolean),
    disabled, safe, saveOut: "", saveStdout: saveStdout.trim(), saveStderr: saveStderr.trim(), combine,
  });
  const sharedFields = (
    <>
      <TextField label="Arguments (one per line)" multiline minRows={3} fullWidth margin="dense" value={args} onChange={(e) => setArgs(e.target.value)} />
      <Typography variant="subtitle2" sx={{ mt: 1 }}>Builtin modules (checked = enabled)</Typography>
      <Box className="mods">
        {modules.map((m) => (
          <FormControlLabel key={m.name} control={<Checkbox size="small" checked={!disabled.includes(m.name)} onChange={() => toggle(m.name)} />} label={m.name + (m.unsafe ? " (unsafe)" : "")} />
        ))}
      </Box>
      <FormControlLabel control={<Checkbox checked={safe} onChange={(e) => setSafe(e.target.checked)} />} label="Safe mode (disable unsafe modules)" />
    </>
  );
  return (
    <Dialog open onClose={onCancel} maxWidth="sm" fullWidth>
      <DialogTitle>
        Run / Debug Settings
        <Typography variant="caption" sx={{ display: "block" }} color="text.secondary">{tab.path}</Typography>
      </DialogTitle>
      <Tabs value={tabIdx} onChange={(_, v: number) => setTabIdx(v)} sx={{ borderBottom: 1, borderColor: "divider", px: 2 }}>
        <Tab label="Run" /><Tab label="Debug" />
      </Tabs>
      <DialogContent dividers>
        {tabIdx === 0 && (
          <>
            {sharedFields}
            <TextField label="Save stdout to file (optional)" fullWidth margin="dense" value={saveStdout} onChange={(e) => setSaveStdout(e.target.value)} placeholder="stdout.log" />
            <TextField label="Save stderr to file (optional)" fullWidth margin="dense" value={saveStderr} onChange={(e) => setSaveStderr(e.target.value)} placeholder="stderr.log" disabled={combine} helperText={combine ? "Combined: both streams go to the stdout file" : ""} />
            <FormControlLabel control={<Checkbox checked={combine} onChange={(e) => setCombine(e.target.checked)} />} label="Combine stdout+stderr into the stdout file" />
          </>
        )}
        {tabIdx === 1 && (
          <>
            {sharedFields}
            <FormControlLabel sx={{ mt: 1 }} control={<Checkbox checked={entry} onChange={(e) => setEntry(e.target.checked)} />} label="Stop on entry (set breakpoints by clicking the gutter)" />
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel}>Cancel</Button>
        <Button variant="outlined" color="success" onClick={() => onRun(cfg())}>Run ▶</Button>
        <Button variant="contained" color="warning" onClick={() => onDebug(cfg(), entry)}>Debug 🐞</Button>
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
  config, onClose, onSave,
}: {
  config: Record<string, unknown>;
  onClose: () => void;
  onSave: (next: Record<string, unknown>) => void;
}) {
  const fmt = (config.fmt as Record<string, unknown>) || {};
  const transpile = (config.transpile as Record<string, unknown>) || {};
  const [expanded, setExpanded] = useState<Record<string, boolean>>(
    Object.fromEntries(NEWLINE_FLAGS.map(([k]) => [k, fmt[k] !== true])),
  );
  const [backup, setBackup] = useState(fmt.backup === true);
  const [writeFunc, setWriteFunc] = useState(String(transpile.writeFunc ?? ""));
  const [rawStart, setRawStart] = useState(String(transpile.rawStrFuncStart ?? ""));
  const [rawEnd, setRawEnd] = useState(String(transpile.rawStrFuncEnd ?? ""));
  function save() {
    const fmtObj: Record<string, unknown> = { ...fmt };
    for (const [k] of NEWLINE_FLAGS) {
      if (expanded[k]) delete fmtObj[k]; else fmtObj[k] = true;
    }
    if (backup) fmtObj.backup = true; else delete fmtObj.backup;
    const trObj: Record<string, unknown> = { ...transpile };
    const setOrDel = (k: string, v: string) => { if (v.trim() === "") delete trObj[k]; else trObj[k] = v; };
    setOrDel("writeFunc", writeFunc); setOrDel("rawStrFuncStart", rawStart); setOrDel("rawStrFuncEnd", rawEnd);
    const next: Record<string, unknown> = { ...config, fmt: fmtObj };
    if (Object.keys(trObj).length > 0) next.transpile = trObj; else delete next.transpile;
    onSave(next);
  }
  return (
    <Dialog open onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Settings</DialogTitle>
      <DialogContent dividers>
        <Typography variant="subtitle2">Formatter (.gad.yaml → fmt)</Typography>
        <Box sx={{ display: "flex", flexDirection: "column" }}>
          {NEWLINE_FLAGS.map(([k, label]) => (
            <FormControlLabel key={k} control={<Checkbox checked={expanded[k]} onChange={(e) => setExpanded((s) => ({ ...s, [k]: e.target.checked }))} />} label={label} />
          ))}
          <FormControlLabel control={<Checkbox checked={backup} onChange={(e) => setBackup(e.target.checked)} />} label="Keep .backup on format" />
        </Box>
        <Typography variant="subtitle2" sx={{ mt: 2 }}>Transpile (.gad.yaml → transpile)</Typography>
        <Typography variant="caption" color="text.secondary">Applied to <code>.gad</code>/<code>.gadt</code> transpile. Leave blank for defaults.</Typography>
        <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5, mt: 1 }}>
          <TextField size="small" label="Write function" placeholder="write" value={writeFunc} onChange={(e) => setWriteFunc(e.target.value)} />
          <TextField size="small" label="Raw-string func start" placeholder="rawstr(" value={rawStart} onChange={(e) => setRawStart(e.target.value)} />
          <TextField size="small" label="Raw-string func end" placeholder=";cast)" value={rawEnd} onChange={(e) => setRawEnd(e.target.value)} />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={save}>Save</Button>
      </DialogActions>
    </Dialog>
  );
}

/** IdeStyles injects the IDE-only layout CSS. */
function IdeStyles() {
  return (
    <style>{`
/* IDE shell */
.ide{position:fixed;inset:0;display:flex;flex-direction:column;background:var(--bg);color:var(--fg);font-size:14px}
.ide-dockview{flex:1;min-height:0}

/* Dockview theme integration */
.dockview-theme-light,.dockview-theme-dark{--dv-background-color:var(--bg);--dv-tabs-and-actions-container-background-color:var(--panel);--dv-activegroup-visiblepanel-tab-background-color:var(--bg);--dv-activegroup-hiddenpanel-tab-background-color:var(--panel);--dv-inactivegroup-visiblepanel-tab-background-color:var(--panel);--dv-inactivegroup-hiddenpanel-tab-background-color:var(--panel);--dv-tab-divider-color:var(--border);--dv-separator-border:var(--border);--dv-tabs-and-actions-container-font-size:12px;--dv-tab-color:var(--muted);--dv-activegroup-visiblepanel-tab-color:var(--fg);--dv-group-view-background-color:var(--bg)}
.dv-tab{padding:0 .7rem !important;font-size:.82rem !important}
.dv-void-container{background:var(--bg)}

/* Explorer panel */
.ide-sidebar{height:100%;overflow:auto;padding:.4rem;background:var(--panel)}
.side-head{display:flex;justify-content:space-between;align-items:center;font-size:.72rem;text-transform:uppercase;color:var(--muted);letter-spacing:.05em;padding:.2rem .3rem}
.tree .node{padding:.12rem .3rem;border-radius:4px;cursor:pointer;white-space:nowrap}
.tree .node:hover{background:var(--code-bg,rgba(125,125,125,.12))}
.tree .node.active{background:var(--accent);color:#fff}
.tree .children{margin-left:.8rem}

/* Editor panel */
.ide-center{height:100%;display:flex;flex-direction:column;min-width:0;background:var(--bg)}
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
.editor-host>div{flex:1;min-width:0}
.editor-host .empty{margin:auto;color:var(--muted)}

/* Output panel */
.panes-dockview{height:100%;display:flex;flex-direction:column;background:var(--panel)}
.pane-tabs{display:flex;gap:.3rem;align-items:center;padding:.25rem .6rem;border-bottom:1px solid var(--border)}
.pane-tabs button.on{background:var(--accent);color:#fff}
.out-log .out-err{color:#e5484d}
.out-split{display:flex;gap:0;padding:0}
.out-split .out-col{flex:1;min-width:0;display:flex;flex-direction:column;border-right:1px solid var(--border)}
.out-split .out-col:last-child{border-right:0}
.out-split .out-col-head{padding:.2rem .6rem;font-size:.72rem;text-transform:uppercase;color:var(--muted);border-bottom:1px solid var(--border)}
.out-split .out-col-head.out-err{color:#e5484d}
.out-split pre{flex:1;overflow:auto;margin:0;padding:.4rem .6rem;white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:.85rem}
.out-split pre.out-err{color:#e5484d}
.panes-dockview .pane-body{flex:1;overflow:auto;margin:0;padding:.5rem .8rem;white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:.85rem}
.frame{padding:.1rem .3rem;border-radius:4px}
.frame:hover{background:var(--code-bg,rgba(125,125,125,.12))}
.frame.selected{background:var(--accent);color:#fff}
.frame.selected .muted{color:rgba(255,255,255,.8)}
.frame .fn{font-weight:600}
.locals-head{margin-bottom:.3rem;font-size:.8rem}
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
table.eval-list td.eval-actions{width:9rem;text-align:right;white-space:nowrap;opacity:.3;transition:opacity .1s}
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

/* Docs / MD preview panel */
.dock-panel-fill{height:100%;display:flex;flex-direction:column;background:var(--panel);overflow:hidden}
.doc-body{flex:1;overflow:auto;padding:.4rem .6rem}
.doc-entry{margin-bottom:.7rem}
.doc-entry-head{display:flex;align-items:center;gap:.4rem;cursor:pointer}
.doc-entry-head:hover .doc-title{color:var(--accent)}
.doc-title{font-family:ui-monospace,monospace;font-size:.82rem;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.doc-kind{font-size:.62rem;text-transform:uppercase;padding:0 .3rem;border-radius:3px;background:var(--code-bg,rgba(125,125,125,.18));color:var(--muted)}
.doc-kind-root{background:var(--accent);color:#fff}
.doc-content{margin:.2rem 0 0;font-size:.82rem;color:var(--fg);line-height:1.45}
.doc-content p{margin:.3rem 0}
.doc-content h1,.doc-content h2,.doc-content h3,.doc-content h4{margin:.5rem 0 .25rem;font-size:.92rem}
.doc-content ul{margin:.3rem 0;padding-left:1.1rem}
.doc-content code{font-family:ui-monospace,monospace;font-size:.92em;background:var(--code-bg,rgba(125,125,125,.15));padding:0 .2rem;border-radius:3px}
.doc-content blockquote{margin:.3rem 0;padding-left:.6rem;border-left:3px solid var(--border);color:var(--muted)}
.doc-content pre.doc-code{margin:.4rem 0;padding:.4rem .6rem;overflow:auto;background:var(--code-bg,rgba(125,125,125,.12));border-radius:5px}
.doc-content pre.doc-code code{background:none;padding:0;white-space:pre}

/* Tree navigator */
.tree-nav{font-family:ui-monospace,monospace;font-size:.85rem}
.tn-row{display:flex;align-items:center;gap:.5rem;padding:.1rem .2rem;cursor:default;border-radius:4px}
.tn-row:hover{background:var(--code-bg,rgba(125,125,125,.12))}
.tn-twist{width:1rem;color:var(--muted);text-align:center}
.tn-key{color:var(--accent)}
.tn-type{color:var(--muted);font-size:.78rem}
.tn-val{white-space:nowrap;overflow:hidden;text-overflow:ellipsis;flex:1}
.tn-goto{background:none;border:none;cursor:pointer;padding:0 2px;font-size:.85em;opacity:.5;color:inherit;line-height:1}.tn-goto:hover{opacity:1}
.tn-loading{color:var(--muted)}

/* Misc */
.mods{display:grid;grid-template-columns:1fr 1fr;gap:0 .8rem;max-height:180px;overflow:auto}
.keybinds{display:flex;flex-direction:column;gap:.4rem;margin:.5rem 0}
.kb-row{display:flex;align-items:center;justify-content:space-between;gap:1rem}
.kb-row button{min-width:7rem;font-family:ui-monospace,monospace}
.muted{color:var(--muted)}
    `}</style>
  );
}
