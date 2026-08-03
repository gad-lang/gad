// Pure helpers to turn the WebFS flat path lists into a nested tree for the
// embeddable IDE navigator. Kept free of React/DOM so it is unit-testable.

export interface FileNode {
  name: string;
  path: string; // full slash path; dirs end with "/"
  dir: boolean;
  children: FileNode[];
}

/**
 * buildTree assembles a sorted nested tree from the visible file paths and the
 * explicitly-created (possibly empty) directory paths. Directories are inferred
 * from the "/" in file paths and unioned with the explicit dir list. Folders
 * sort before files, each alphabetically.
 */
export function buildTree(files: string[], dirs: string[] = []): FileNode {
  const root: FileNode = { name: "", path: "", dir: true, children: [] };
  const dirIndex = new Map<string, FileNode>([["", root]]);

  const ensureDir = (dirPath: string): FileNode => {
    // dirPath has no trailing slash here ("" is the root).
    const existing = dirIndex.get(dirPath);
    if (existing) return existing;
    const slash = dirPath.lastIndexOf("/");
    const parentPath = slash === -1 ? "" : dirPath.slice(0, slash);
    const name = slash === -1 ? dirPath : dirPath.slice(slash + 1);
    const parent = ensureDir(parentPath);
    const node: FileNode = { name, path: dirPath + "/", dir: true, children: [] };
    parent.children.push(node);
    dirIndex.set(dirPath, node);
    return node;
  };

  for (const d of dirs) {
    const clean = d.replace(/\/+$/, "");
    if (clean) ensureDir(clean);
  }

  for (const f of files) {
    const slash = f.lastIndexOf("/");
    const parent = slash === -1 ? root : ensureDir(f.slice(0, slash));
    parent.children.push({
      name: slash === -1 ? f : f.slice(slash + 1),
      path: f,
      dir: false,
      children: [],
    });
  }

  sortTree(root);
  return root;
}

function sortTree(node: FileNode): void {
  node.children.sort((a, b) => {
    if (a.dir !== b.dir) return a.dir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
  for (const c of node.children) if (c.dir) sortTree(c);
}

/** langFor picks a source dialect from a file extension, for doc/highlight. */
export function langFor(path: string): "gad" | "gadTemplate" | "giom" {
  if (path.endsWith(".giom")) return "giom";
  if (path.endsWith(".gadt")) return "gadTemplate";
  return "gad";
}
