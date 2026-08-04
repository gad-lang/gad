// The IDE controller: all reactive state and actions for <GadIde>, extracted so
// the dockview panels (Explorer, Editor, Call Stack, Locals, Output) can share
// one instance via provide/inject. dockview-vue teleports panels into the host's
// Vue tree, so inject() resolves the provided controller natively.
import { computed, reactive, ref, shallowRef, watch, type InjectionKey, type Ref } from "vue";
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import { langOf } from "./codemirror";
import type { LocalVar } from "./codemirror";
import { renderDocMarkdown } from "./docMarkdown";
import type { DebugResponse, IdeApi, InspectResult, TreeNode, Workspace } from "./api";
import type { RunResult } from "./types";
import type { InspectFn } from "./InspectorNode";

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

export type IdeController = ReturnType<typeof createController>;

/** Injection key for the shared controller. */
export const IdeControllerKey: InjectionKey<IdeController> = Symbol("gad-ide-controller");

export function createController(
  api: IdeApi,
  workspace: Workspace,
  dark: Ref<boolean>,
  onReset?: () => Promise<void> | void,
) {
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

  async function loadTree() {
    tree.value = await api.tree();
    const parts = openPath.value.split("/");
    for (let i = 1; i < parts.length; i++) expanded.add(parts.slice(0, i).join("/"));
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
  const breakpoints = ref<number[]>([]);
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
          breakpoints: [...breakpoints.value].sort((a, b) => a - b),
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
    newFile, newDir, removeOpen, reset, canReset,
    diagnose,
    // run/format/doc
    busy, runRes, run, format, docHtml, refreshDoc,
    // debug
    session, snap, dbgOutput, breakpoints, debugLine, debugColumn, stopped,
    getLocals, debugStart, debugCmd, evalExpr, evalOut, doEval,
    inspectFn, dialect,
    init,
  };
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c] as string);
}
