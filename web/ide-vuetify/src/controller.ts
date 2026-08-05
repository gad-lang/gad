// The IDE controller: all reactive state and actions for <GadIde>, extracted so
// the dockview panels (Explorer, Editor, Call Stack, Locals, Output) can share
// one instance via provide/inject. dockview-vue teleports panels into the host's
// Vue tree, so inject() resolves the provided controller natively.
import { computed, reactive, ref, shallowRef, watch, type InjectionKey, type Ref } from "vue";
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import { langOf } from "./codemirror";
import type { LocalVar } from "./codemirror";
import { renderDocMarkdown } from "./docMarkdown";
import type { BreakpointSpec, DebugResponse, IdeApi, InspectResult, RunMode, RunProfile, TreeNode, Workspace } from "./api";
import type { RunResult } from "./types";
import type { InspectFn } from "./InspectorNode";
import type { GadEditorView } from "./codemirror";

export interface TreeRow {
  node: TreeNode;
  depth: number;
}

/** RunTarget is what run()/debugStart() execute: a file, its content and args. */
export interface RunTarget {
  source: string;
  path: string;
  args: string[];
}

/** BpMeta is a breakpoint's per-line metadata: disabled flag and condition. */
export interface BpMeta {
  disabled?: boolean;
  condition?: string;
}

export type IdeController = ReturnType<typeof createController>;

/** Injection key for the shared controller. */
export const IdeControllerKey: InjectionKey<IdeController> = Symbol("gad-ide-controller");

/** ControllerHooks feed the reactive props/emits GadIde owns into the controller
 * so the editor-panel toolbar (run profiles, run/debug gating) can live here. */
export interface ControllerHooks {
  onReset?: () => Promise<void> | void;
  getRunProfiles?: () => RunProfile[];
  getRunMode?: () => RunMode;
  emitRunProfiles?: (profiles: RunProfile[]) => void;
}

export function createController(
  api: IdeApi,
  workspace: Workspace,
  dark: Ref<boolean>,
  hooks: ControllerHooks = {},
) {
  const onReset = hooks.onReset;
  // --- file tree ----------------------------------------------------------
  const tree = shallowRef<TreeNode | null>(null);
  const expanded = reactive(new Set<string>());
  const openPath = ref<string>(workspace.openFile || "");
  const source = ref<string>("");

  const rows = computed<TreeRow[]>(() => {
    const out: TreeRow[] = [];
    const walk = (node: TreeNode, depth: number) => {
      for (const c of node.children ?? []) {
        out.push({ node: c, depth });
        if (c.dir && expanded.has(c.path)) walk(c, depth + 1);
      }
    };
    if (tree.value) walk(tree.value, 0);
    return out;
  });

  const showHidden = ref(false);
  async function loadTree() {
    tree.value = await api.tree(showHidden.value);
    const parts = openPath.value.split("/");
    for (let i = 1; i < parts.length; i++) expanded.add(parts.slice(0, i).join("/"));
  }
  async function toggleHidden() {
    showHidden.value = !showHidden.value;
    await loadTree();
  }
  async function openFile(path: string) {
    openPath.value = path;
    source.value = (await api.read(path)).content;
  }
  function toggleDir(path: string) {
    if (expanded.has(path)) expanded.delete(path);
    else expanded.add(path);
  }
  function isExpanded(path: string) {
    return expanded.has(path);
  }

  watch(source, (v) => {
    if (openPath.value) void api.write(openPath.value, v);
  });

  function currentDir(): string {
    const slash = openPath.value.lastIndexOf("/");
    return slash === -1 ? "" : openPath.value.slice(0, slash + 1);
  }
  function firstFile(node: TreeNode | null): string | null {
    for (const c of node?.children ?? []) {
      if (!c.dir) return c.path;
      const f = firstFile(c);
      if (f) return f;
    }
    return null;
  }
  async function newFile() {
    const name = prompt("New file name (e.g. hello.gad):", "untitled.gad");
    if (!name) return;
    const path = currentDir() + name;
    await api.mkfile(path);
    await loadTree();
    await openFile(path);
  }
  async function newDir() {
    const name = prompt("New folder name:", "folder");
    if (!name) return;
    await api.mkdir(currentDir() + name);
    await loadTree();
  }
  async function removeOpen() {
    if (!openPath.value || !confirm("Delete " + openPath.value + "?")) return;
    await api.del(openPath.value);
    openPath.value = "";
    source.value = "";
    await loadTree();
  }
  async function reset() {
    if (!onReset || !confirm("Discard all your changes and restore the samples?")) return;
    await onReset();
    await loadTree();
    const first = firstFile(tree.value);
    if (first) await openFile(first);
  }
  const canReset = computed(() => !!onReset);

  // --- diagnose -----------------------------------------------------------
  const diagnose = (src: string): Promise<GadDiagnostic[]> => api.diagnose(src);

  // --- run / format / doc -------------------------------------------------
  const busy = ref(false);
  const runRes = ref<RunResult | null>(null);
  // The current file as a run target (used when no profile is active).
  const currentTarget = (): RunTarget => ({ source: source.value, path: openPath.value, args: [] });
  async function run(target?: RunTarget) {
    const t = target ?? currentTarget();
    busy.value = true;
    try {
      runRes.value = await api.run({ source: t.source, path: t.path, args: t.args });
    } finally {
      busy.value = false;
    }
  }
  async function format() {
    busy.value = true;
    try {
      const r = await api.format(source.value);
      if (r.ok) source.value = r.source;
      else runRes.value = { ok: false, stdout: "", stderr: "", result: "", diagnostics: r.diagnostics };
    } finally {
      busy.value = false;
    }
  }

  // --- editor actions (save / reload / undo / redo) -----------------------
  // The open editor registers itself so undo/redo can reach the CodeMirror view.
  const editor = shallowRef<GadEditorView | null>(null);
  function registerEditor(v: GadEditorView | null) {
    editor.value = v;
  }
  async function save() {
    if (openPath.value) await api.write(openPath.value, source.value);
  }
  async function reload() {
    if (openPath.value) source.value = (await api.read(openPath.value)).content;
  }
  function undo() {
    editor.value?.undo();
  }
  function redo() {
    editor.value?.redo();
  }
  // Rendered documentation of the open file, shown by the Docs panel.
  const docHtml = ref("");
  async function refreshDoc() {
    const docs = await api.doc(source.value);
    docHtml.value = docs.length
      ? docs.map((d) => `<h4>${escapeHtml(d.title || d.kind)}</h4>` + renderDocMarkdown(d.content)).join("\n")
      : `<p class="text-medium-emphasis">No documentation comments in this file.</p>`;
  }

  // --- debugger -----------------------------------------------------------
  const session = ref<string | null>(null);
  const snap = ref<DebugResponse | null>(null);
  const dbgOutput = ref("");

  // Per-file breakpoints and their metadata (disabled flag + condition), keyed
  // by workspace path — surfaced by the Breakpoints panel and the gutter.
  const bpLinesByFile = reactive<Record<string, number[]>>({});
  const bpMetaByFile = reactive<Record<string, Record<number, BpMeta>>>({});
  function bpFor(path?: string): number[] {
    return (path && bpLinesByFile[path]) || [];
  }
  function bpMetaFor(path?: string): Record<number, BpMeta> {
    return (path && bpMetaByFile[path]) || {};
  }
  function allBreakpoints(): Record<string, number[]> {
    const out: Record<string, number[]> = {};
    for (const [p, ls] of Object.entries(bpLinesByFile)) if (ls.length) out[p] = ls;
    return out;
  }
  function setFileBreakpoints(path: string, lines: number[]) {
    const sorted = [...new Set(lines)].sort((a, b) => a - b);
    bpLinesByFile[path] = sorted;
    const m = bpMetaByFile[path];
    if (m) for (const k of Object.keys(m)) if (!sorted.includes(Number(k))) delete m[Number(k)];
  }
  function setBpMeta(path: string, line: number, meta: BpMeta) {
    if (!bpMetaByFile[path]) bpMetaByFile[path] = {};
    bpMetaByFile[path][line] = { ...(bpMetaByFile[path][line] || {}), ...meta };
    if (!bpFor(path).includes(line)) setFileBreakpoints(path, [...bpFor(path), line]);
  }
  function removeBreakpoint(path: string, line: number) {
    setFileBreakpoints(path, bpFor(path).filter((l) => l !== line));
  }
  function bpSpecsFor(path: string): BreakpointSpec[] {
    const m = bpMetaFor(path);
    return bpFor(path).map((line) => ({ line, disabled: m[line]?.disabled, condition: m[line]?.condition }));
  }
  // v-model binding for the editor gutter (the current file's breakpoints).
  const breakpoints = computed<number[]>({
    get: () => bpFor(openPath.value),
    set: (v) => setFileBreakpoints(openPath.value, v),
  });
  // A navigation target (seq bumped on each request) the editor scrolls to.
  const gotoTarget = ref<{ line: number; seq: number }>({ line: 0, seq: 0 });
  function goto(line: number) {
    gotoTarget.value = { line, seq: gotoTarget.value.seq + 1 };
  }
  async function gotoFileLine(path: string, line: number) {
    if (path && path !== openPath.value) await openFile(path);
    goto(line);
  }
  // Breakpoint-condition dialog target ({ path, line } | null), shared so the
  // editor gutter's right-click and the Breakpoints panel open the same dialog.
  const bpDialog = ref<{ path: string; line: number } | null>(null);
  function openBpCondition(path: string, line: number) {
    bpDialog.value = { path, line };
  }
  const debugLine = computed(() => (snap.value?.state === "stopped" ? snap.value.line ?? 0 : 0));
  const debugColumn = computed(() => snap.value?.column ?? 1);
  const stopped = computed(() => snap.value?.state === "stopped");
  const localsMap = computed<Map<string, LocalVar>>(() => {
    const m = new Map<string, LocalVar>();
    for (const v of snap.value?.locals ?? []) m.set(v.name, v);
    return m;
  });
  const getLocals = () => localsMap.value;

  function applySnap(r: DebugResponse) {
    snap.value = r;
    if (r.output) dbgOutput.value += r.output;
    if (r.state === "terminated" || r.state === "error") session.value = null;
    else if (r.session) session.value = r.session;
  }
  async function debugStart(target?: RunTarget) {
    const t = target ?? currentTarget();
    busy.value = true;
    dbgOutput.value = "";
    snap.value = null;
    runRes.value = null;
    try {
      applySnap(
        await api.dbgStart({
          source: t.source,
          path: t.path,
          args: t.args,
          breakpoints: bpFor(t.path),
          breakpointSpecs: bpSpecsFor(t.path),
          stopOnEntry: false,
        }),
      );
    } finally {
      busy.value = false;
    }
  }
  async function debugCmd(command: "continue" | "next" | "stepIn" | "stepOut") {
    if (!session.value) return;
    busy.value = true;
    try {
      applySnap(await api.dbgCmd(session.value, command));
    } finally {
      busy.value = false;
    }
  }

  const evalExpr = ref("");
  const evalOut = ref("");
  async function doEval() {
    if (!session.value || !evalExpr.value.trim()) return;
    const r = await api.dbgEval(session.value, evalExpr.value, true);
    evalOut.value = r.ok ? r.value ?? "" : "error: " + (r.error ?? "");
  }

  // --- value inspector ----------------------------------------------------
  const inspectFn: InspectFn = async (expr: string): Promise<InspectResult | null> => {
    const r = await api.inspect({
      expr,
      session: session.value ?? undefined,
      source: source.value,
      path: openPath.value,
    });
    return r.ok ? r.inspect ?? null : null;
  };

  const dialect = computed(() => {
    const l = langOf(openPath.value);
    return l === "gadx" ? "gadx" : l === "gadt" ? "gadTemplate" : "gad";
  });

  // --- run/debug profiles (JetBrains-style) -------------------------------
  const runProfiles = computed(() => hooks.getRunProfiles?.() ?? []);
  const runMode = computed<RunMode>(() => hooks.getRunMode?.() ?? "debug");
  const activeProfile = ref<string | null>(null);
  const profileDialog = ref(false);
  const activeProfileObj = computed(() => runProfiles.value.find((p) => p.name === activeProfile.value) ?? null);
  const runLabel = computed(() => activeProfileObj.value?.name ?? "Current file");
  const canRun = computed(() => runMode.value === "run" || runMode.value === "debug");
  const canDebug = computed(() => runMode.value === "debug");

  // effectiveTarget resolves what Run/Debug execute: the active profile's file +
  // args (read fresh unless it is the open file), else the open file.
  async function effectiveTarget(): Promise<RunTarget> {
    const p = activeProfileObj.value;
    if (!p) return { source: source.value, path: openPath.value, args: [] };
    const src = p.path === openPath.value ? source.value : (await api.read(p.path)).content;
    return { source: src, path: p.path, args: p.args };
  }
  async function runActive() {
    await run(await effectiveTarget());
  }
  async function debugActive() {
    await debugStart(await effectiveTarget());
  }
  function addProfile(p: RunProfile) {
    hooks.emitRunProfiles?.([...runProfiles.value.filter((x) => x.name !== p.name), p]);
    activeProfile.value = p.name;
  }
  function deleteProfile(name: string) {
    hooks.emitRunProfiles?.(runProfiles.value.filter((p) => p.name !== name));
    if (activeProfile.value === name) activeProfile.value = null;
  }

  // Docs-reveal request signal: the toolbar Doc button bumps it; GadIde watches
  // it to reveal/focus the Docs panel (which owns the dockview api).
  const docRequest = ref(0);
  function requestDocs() {
    void refreshDoc();
    docRequest.value++;
  }

  // Settings dialog open state — set by the editor toolbar's Settings button,
  // consumed by the dialog GadIde renders (its panel tab needs the dockview api).
  const settingsOpen = ref(false);

  async function init() {
    await loadTree();
    if (openPath.value) await openFile(openPath.value);
    else {
      const first = firstFile(tree.value);
      if (first) await openFile(first);
    }
  }

  return {
    api,
    dark,
    // tree
    tree, rows, openPath, source, isExpanded, toggleDir, openFile,
    showHidden, toggleHidden,
    newFile, newDir, removeOpen, reset, canReset,
    diagnose,
    // run/format/doc
    busy, runRes, run, format, docHtml, refreshDoc,
    // editor actions
    registerEditor, save, reload, undo, redo,
    // run/debug profiles + gating
    runProfiles, runMode, activeProfile, profileDialog, activeProfileObj, runLabel,
    canRun, canDebug, runActive, debugActive, addProfile, deleteProfile,
    docRequest, requestDocs, settingsOpen,
    // debug
    session, snap, dbgOutput, breakpoints, debugLine, debugColumn, stopped,
    getLocals, debugStart, debugCmd, evalExpr, evalOut, doEval,
    inspectFn, dialect,
    // breakpoints
    bpFor, bpMetaFor, allBreakpoints, setFileBreakpoints, setBpMeta, removeBreakpoint,
    gotoTarget, goto, gotoFileLine, bpDialog, openBpCondition,
    init,
  };
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c] as string);
}
