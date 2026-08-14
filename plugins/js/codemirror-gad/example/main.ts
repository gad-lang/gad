// Demo: a CodeMirror 6 editor with Gad language support. The sidebar is a tree
// of the repository `samples/` directory (.gad / .gadt / .gadx); clicking a file
// opens it in the editor with the matching sourceType. The tree and file
// contents are read from the filesystem by the dev server (see serve.ts). Serve
// with `bun run demo`.
import {
  EditorView,
  lineNumbers,
  highlightActiveLine,
  drawSelection,
} from "@codemirror/view";
import { EditorState, type Extension } from "@codemirror/state";
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  bracketMatching,
  indentOnInput,
} from "@codemirror/language";
import { closeBrackets, autocompletion } from "@codemirror/autocomplete";
import { gad, type GadOptions } from "../src/index";
import { iconFor } from "./fileIcons";

// Sample files are read from the filesystem by the dev server (see serve.ts):
// the manifest lists them, and each file's contents are fetched on demand.
async function fetchManifest(): Promise<string[]> {
  const res = await fetch("./samples/manifest.json");
  return res.ok ? ((await res.json()) as string[]) : [];
}

async function fetchSample(path: string): Promise<string> {
  const res = await fetch("./samples/" + path);
  return res.ok ? await res.text() : `// failed to load ${path}`;
}

// sourceTypeFor picks the highlighter dialect from a sample's extension.
function sourceTypeFor(path: string): GadOptions {
  if (path.endsWith(".gadx")) return { sourceType: "gadx" };
  if (path.endsWith(".gadt")) return { sourceType: "template" };
  return { sourceType: "gad" };
}

// --- editor ----------------------------------------------------------------
const base: Extension = [
  lineNumbers(),
  highlightActiveLine(),
  drawSelection(),
  indentOnInput(),
  bracketMatching(),
  closeBrackets(),
  autocompletion(),
  syntaxHighlighting(defaultHighlightStyle),
  EditorView.editable.of(true),
];

const parent = document.getElementById("editor")!;
let view: EditorView | undefined;

async function open(path: string): Promise<void> {
  const doc = await fetchSample(path);
  view?.destroy();
  view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [base, gad(sourceTypeFor(path))],
    }),
    parent,
  });
}

// --- sample tree -----------------------------------------------------------
// Build a nested tree from the flat "dir/sub/file" sample paths.
interface TreeNode {
  dirs: Map<string, TreeNode>;
  files: string[]; // full sample paths
}

function buildTree(paths: string[]): TreeNode {
  const root: TreeNode = { dirs: new Map(), files: [] };
  for (const path of paths) {
    const parts = path.split("/");
    let node = root;
    for (let i = 0; i < parts.length - 1; i++) {
      const dir = parts[i];
      if (!node.dirs.has(dir)) node.dirs.set(dir, { dirs: new Map(), files: [] });
      node = node.dirs.get(dir)!;
    }
    node.files.push(path);
  }
  return root;
}

let activeButton: HTMLButtonElement | undefined;

function renderTree(node: TreeNode, container: HTMLElement): void {
  for (const [dir, child] of [...node.dirs.entries()].sort()) {
    const details = document.createElement("details");
    details.open = true;
    const summary = document.createElement("summary");
    summary.textContent = dir + "/";
    details.appendChild(summary);
    const sub = document.createElement("div");
    sub.className = "tree-children";
    renderTree(child, sub);
    details.appendChild(sub);
    container.appendChild(details);
  }
  for (const path of node.files.sort()) {
    const btn = document.createElement("button");
    btn.className = "file";
    const icon = document.createElement("img");
    icon.className = "file-icon";
    icon.src = iconFor(path);
    icon.alt = "";
    btn.append(icon, document.createTextNode(path.split("/").pop()!));
    btn.onclick = () => {
      activeButton?.classList.remove("active");
      btn.classList.add("active");
      activeButton = btn;
      open(path);
    };
    container.appendChild(btn);
  }
}

const treeEl = document.getElementById("tree")!;

// Build the tree from the filesystem manifest at startup, then open the first
// file by default.
fetchManifest().then((paths) => {
  paths.sort();
  renderTree(buildTree(paths), treeEl);
  treeEl.querySelector<HTMLButtonElement>("button.file")?.click();
});
