// localIdeApi — an in-browser IdeApi implementation for the reusable <Ide>
// component, so the full IDE runs with no Go server. Files come from WebFS (a
// read-only bundled sample tree + a LocalStorage overlay); run/format/diagnose/
// doc/eval/transpile and the stepping debugger all go through the Gad WASM
// module hosted in a Web Worker (see wasm/client.ts). Config (layout,
// breakpoints, …) persists to LocalStorage.
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import type {
  IdeApi,
  TreeNode,
  Workspace,
  ModuleInfo,
  DocComment,
  EvalResult,
  InspectResult,
  BreakpointSpec,
  DebugResponse,
} from "@gad-lang/ide-react";
import { WebFS } from "../webfs";
import { sharedClient } from "./wasmWorker";

const fs = new WebFS();
const CONFIG_KEY = "gad-webide-config-v1";

/** dialectOf maps a file extension to the WASM run/diagnose sourceType. */
function dialectOf(path = ""): string {
  if (path.endsWith(".gadx")) return "gadx";
  if (path.endsWith(".gadt")) return "gadTemplate";
  return "gad";
}

/** buildTree assembles a nested TreeNode from the flat WebFS listings. */
/** buildTree assembles the nested TreeNode from the flat WebFS listings. When
 * `hidden` is false, paths with a dot-prefixed segment (e.g. `.gad/…`) are
 * omitted — the show/hidden toggle. */
function buildTree(hidden: boolean): TreeNode {
  const visible = (p: string) => hidden || !p.split("/").some((s) => s.startsWith("."));
  const root: TreeNode = { name: "", path: "", dir: true, children: [] };
  const dirs = new Map<string, TreeNode>([["", root]]);
  const ensureDir = (dirPath: string): TreeNode => {
    const hit = dirs.get(dirPath);
    if (hit) return hit;
    const slash = dirPath.lastIndexOf("/");
    const parent = ensureDir(slash === -1 ? "" : dirPath.slice(0, slash));
    const node: TreeNode = {
      name: slash === -1 ? dirPath : dirPath.slice(slash + 1),
      path: dirPath,
      dir: true,
      children: [],
    };
    parent.children!.push(node);
    dirs.set(dirPath, node);
    return node;
  };
  for (const d of fs.listDirs()) {
    const clean = d.replace(/\/+$/, "");
    if (clean && visible(clean)) ensureDir(clean);
  }
  for (const f of fs.listFiles()) {
    if (!visible(f)) continue;
    const slash = f.lastIndexOf("/");
    const parent = slash === -1 ? root : ensureDir(f.slice(0, slash));
    parent.children!.push({ name: slash === -1 ? f : f.slice(slash + 1), path: f, dir: false });
  }
  const sort = (n: TreeNode) => {
    n.children?.sort((a, b) => (a.dir !== b.dir ? (a.dir ? -1 : 1) : a.name.localeCompare(b.name)));
    n.children?.forEach((c) => c.dir && sort(c));
  };
  sort(root);
  return root;
}

function loadConfig(): Record<string, unknown> {
  try {
    const raw = localStorage.getItem(CONFIG_KEY);
    return raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
  } catch {
    return {};
  }
}

// A small, static list of importable builtin namespaces for the modules panel.
const MODULES: ModuleInfo[] = [
  { name: "strings", unsafe: false },
  { name: "fmt", unsafe: false },
  { name: "json", unsafe: false },
  { name: "time", unsafe: false },
  { name: "math", unsafe: false },
];

export const localIdeApi: IdeApi = {
  workspace: async (): Promise<Workspace> => {
    const files = fs.listFiles();
    // Open the "01_hello.gad" sample by default when present.
    const openFile = files.find((p) => p === "01_hello.gad" || p.endsWith("/01_hello.gad")) ?? files[0] ?? "";
    return { root: "gad (in-browser)", name: "gad", openFile };
  },
  tree: async (hidden = false) => buildTree(hidden),
  read: async (path: string) => ({ path, content: fs.read(path) ?? "" }),
  write: async (path: string, content: string) => {
    fs.write(path, content);
    return { path };
  },
  mkfile: async (path: string) => {
    fs.createFile(path);
    return { path };
  },
  del: async (path: string) => {
    fs.remove(path);
    return { path };
  },
  rename: async (path: string, to: string) => {
    const content = fs.read(path) ?? "";
    fs.write(to, content);
    fs.remove(path);
    return { path: to };
  },
  mkdir: async (path: string) => {
    fs.createDir(path);
    return { path };
  },
  fetchUrl: async (url: string, path: string) => {
    const text = await (await fetch(url)).text();
    fs.write(path, text);
    return { path, size: text.length };
  },
  config: async () => loadConfig(),
  saveConfig: async (doc: Record<string, unknown>) => {
    try {
      localStorage.setItem(CONFIG_KEY, JSON.stringify(doc));
    } catch {
      /* storage full/unavailable */
    }
    return doc;
  },
  modules: async () => MODULES,
  format: (source: string) => sharedClient().format(source),
  transpile: (source: string, path?: string) =>
    sharedClient().transpile(source, (path ?? "").endsWith(".gadt")),
  doc: async (source: string): Promise<DocComment[]> => (await sharedClient().docComments(source)).docs || [],
  eval: (req: { expr: string; repr?: boolean; source?: string; path?: string }): Promise<EvalResult> =>
    sharedClient().evalExpr(req.source ?? "", req.expr, req.repr ?? false),
  inspect: (req: { expr: string; session?: string; source?: string; path?: string }): Promise<{
    ok: boolean;
    inspect?: InspectResult;
    error?: string;
  }> => sharedClient().inspect(req.session ?? "", req.expr, req.source ?? ""),
  diagnose: async (source: string): Promise<GadDiagnostic[]> =>
    (await sharedClient().diagnose(source)).diagnostics,
  run: (req: {
    path?: string;
    source?: string;
    args?: string[];
    disabled?: string[];
    safe?: boolean;
    saveOut?: string;
    saveStdout?: string;
    saveStderr?: string;
    combine?: boolean;
    tagEncode?: string;
  }) => sharedClient().run(req.source ?? fs.read(req.path ?? "") ?? "", dialectOf(req.path), req.args ?? [], req.tagEncode ?? ""),
  dbgStart: (req: {
    source: string;
    breakpoints: number[];
    breakpointSpecs?: BreakpointSpec[];
    stopOnEntry: boolean;
    path?: string;
    args?: string[];
  }): Promise<DebugResponse> =>
    sharedClient().debugStart(
      req.source, req.path ?? "", req.breakpoints, req.stopOnEntry, req.args ?? [], req.breakpointSpecs ?? [],
    ),
  dbgCmd: (session: string, command: string): Promise<DebugResponse> =>
    sharedClient().debugCommand(session, command),
  dbgEval: (session: string, expr: string, repr: boolean) =>
    sharedClient().debugEval(session, expr, repr),
};
