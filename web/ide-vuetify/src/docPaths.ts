// DOC-PATH resolution for rendered documentation. Generated docs mirror the
// source tree under a `doc/` root (a `.gad` at `a/b.gad` is documented at
// `doc/a/b.md`), and asset/link references in doc comments resolve against that
// tree: a root-absolute `/x` points at the doc root, and a relative `x` is
// relative to the doc file's directory (`doc/<dir>/x`). This rewrites those
// references in the rendered doc HTML to the matching path. External URLs
// (scheme, protocol-relative), in-page anchors and data URIs are left as is.
//
// The doc root defaults to `doc`, but a host can supply a DocRootFn to point the
// references at a different location/URI (a CDN, an HTTP asset route, …), chosen
// per source path.

/** DocRootFn returns the doc root (a path or absolute URI) for a source path. */
export type DocRootFn = (sourcePath: string) => string;

/** dirOf returns a path's directory (no leading/trailing slash). */
function dirOf(path: string): string {
  const clean = path.replace(/^\/+/, "");
  const slash = clean.lastIndexOf("/");
  return slash === -1 ? "" : clean.slice(0, slash);
}

/** joinUri joins base and rel, collapsing `.`/`..` while preserving a leading
 * scheme (`https://`), protocol-relative (`//`) or root (`/`) prefix. */
function joinUri(base: string, rel: string): string {
  const combined = base.replace(/\/+$/, "") + "/" + rel.replace(/^\/+/, "");
  const m = /^([a-z][a-z0-9+.-]*:\/\/|\/\/|\/)/i.exec(combined);
  const prefix = m ? m[1] : "";
  const out: string[] = [];
  for (const seg of combined.slice(prefix.length).split("/")) {
    if (seg === "" || seg === ".") continue;
    if (seg === "..") out.pop();
    else out.push(seg);
  }
  return prefix + out.join("/");
}

/** resolveDocRef maps one doc-comment URL to a doc-tree path, or null to leave it
 * unchanged (external URL, anchor, data URI, empty). */
export function resolveDocRef(url: string, docRoot: string, fileDir: string): string | null {
  if (!url || url.startsWith("#") || url.startsWith("//") || /^[a-z][a-z0-9+.-]*:/i.test(url)) {
    return null;
  }
  return url.startsWith("/") ? joinUri(docRoot, url) : joinUri(fileDir, url);
}

/** resolveDocPaths rewrites `src`/`href` references in doc HTML to their doc-tree
 * paths for the source at sourcePath. docRootFor overrides the default `doc` root
 * (e.g. to serve assets from a different URI). */
export function resolveDocPaths(html: string, sourcePath: string, docRootFor?: DocRootFn): string {
  if (!html || typeof DOMParser === "undefined") return html;
  const docRoot = docRootFor ? docRootFor(sourcePath) : "doc";
  const dir = dirOf(sourcePath);
  const fileDir = dir ? joinUri(docRoot, dir) : docRoot;
  const doc = new DOMParser().parseFromString(html, "text/html");
  for (const attr of ["src", "href"]) {
    doc.querySelectorAll(`[${attr}]`).forEach((el) => {
      const resolved = resolveDocRef(el.getAttribute(attr) ?? "", docRoot, fileDir);
      if (resolved !== null) el.setAttribute(attr, resolved);
    });
  }
  return doc.body.innerHTML;
}
