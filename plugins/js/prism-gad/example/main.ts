// Demo: static highlighting with PrismJS and @gad-lang/prism-gad. The sidebar is
// a tree of the repository `samples/` directory (.gad / .gadt / .gadx); clicking
// a file highlights it with the grammar chosen by `gadGrammarFor(sourceType)`.
// The tree and file contents are read from the filesystem by the dev server
// (see serve.ts). Serve with `bun run demo`.
import Prism from "prismjs";
import { registerGad, registerGadx, gadGrammarFor, type GadSourceType } from "../src/index";
import { iconFor } from "./fileIcons";

registerGad(Prism); // the embedded Gad grammar (required by template & gadx)
registerGadx(Prism); // Prism.languages.gadx

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

// sourceTypeFor picks the grammar dialect from a sample's extension.
function sourceTypeFor(path: string): GadSourceType {
  if (path.endsWith(".gadx")) return "gadx";
  if (path.endsWith(".gadt")) return "template";
  return "gad";
}

// --- output ----------------------------------------------------------------
const out = document.getElementById("out")!;

async function render(path: string): Promise<void> {
  const source = await fetchSample(path);
  const sourceType = sourceTypeFor(path);
  const grammar = gadGrammarFor(sourceType);
  const lang = sourceType === "template" ? "gadt" : sourceType; // language- class for the theme
  const code = Prism.highlight(source, grammar, sourceType);
  out.innerHTML = `<pre class="language-${lang}"><code class="language-${lang}">${code}</code></pre>`;
}

// --- sample tree -----------------------------------------------------------
interface TreeNode {
  dirs: Map<string, TreeNode>;
  files: string[];
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
      render(path);
    };
    container.appendChild(btn);
  }
}

const treeEl = document.getElementById("tree")!;

// Build the tree from the filesystem manifest at startup, then highlight the
// first file by default.
fetchManifest().then((paths) => {
  paths.sort();
  renderTree(buildTree(paths), treeEl);
  treeEl.querySelector<HTMLButtonElement>("button.file")?.click();
});
