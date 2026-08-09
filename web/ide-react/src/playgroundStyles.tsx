// Shared styles for the standalone GadPlayground / GadNotebook components. They
// are injected once into <head> (deduped by id) so the components work on their
// own, without the full IDE stylesheet. Colors use the same CSS variables the IDE
// defines (--bg/--fg/--panel/--border/--accent/--muted/--code-bg), with fallbacks
// so the components also render acceptably outside the IDE shell.
import { useEffect } from "react";

const STYLE_ID = "gad-playground-styles";

const CSS = `
.gp-split{display:grid;grid-template-columns:1fr 1fr;height:100%;min-height:0;
  background:var(--bg,#fff);color:var(--fg,#1d1d28);font-size:14px}
/* When the Doc pane is shown, the playground becomes three columns. */
.gp-split:has(.gp-pane--doc){grid-template-columns:1fr 1fr 1fr}
.gp-pane{display:flex;flex-direction:column;min-width:0;min-height:0;border-right:1px solid var(--border,#e2e2ea)}
.gp-pane:last-child{border-right:none}
.gp-pane-head{display:flex;align-items:center;justify-content:space-between;gap:8px;
  padding:4px 8px;border-bottom:1px solid var(--border,#e2e2ea);font-size:13px;min-height:36px;
  background:var(--panel,#fafafa)}
.gp-dialect{display:flex;gap:0;border:1px solid var(--border,#e2e2ea);border-radius:6px;overflow:hidden}
.gp-tab{background:transparent;border:none;border-right:1px solid var(--border,#e2e2ea);
  color:var(--fg,#1d1d28);cursor:pointer;padding:.25rem .6rem;font-size:.82rem}
.gp-tab:last-child{border-right:none}
.gp-tab--active{background:var(--accent,#3b5bdb);color:#fff}
.gp-actions{display:flex;gap:6px;align-items:center}
.gp-tagenc{display:flex;align-items:center;gap:4px;font-size:.8rem;color:var(--muted,#6b6b80)}
.gp-tagenc select{font-size:.8rem;padding:.15rem .3rem;border:1px solid var(--border,#e2e2ea);border-radius:5px;background:var(--panel,#fff);color:var(--fg,#1d1d28)}
.gp-btn{background:var(--panel,#fff);border:1px solid var(--border,#e2e2ea);color:var(--fg,#1d1d28);
  border-radius:6px;padding:.3rem .7rem;cursor:pointer;font-size:.85rem}
.gp-btn:hover:not(:disabled){background:var(--code-bg,rgba(125,125,125,.12))}
.gp-btn:disabled{opacity:.5;cursor:default}
.gp-btn--primary{background:var(--accent,#3b5bdb);border-color:var(--accent,#3b5bdb);color:#fff}
.gp-pane-body{flex:1;overflow:auto;padding:8px}
.gp-editor{flex:1;min-height:0;overflow:hidden}
.gp-editor .cm-editor{height:100%}
.gp-out{white-space:pre-wrap;font-family:ui-monospace,monospace;font-size:12px;margin:4px 0}
.gp-muted{color:var(--muted,#6b6b80)}
.gp-error{color:var(--error,#d64545)}
.gp-return{font-family:ui-monospace,monospace;font-size:12px;color:var(--muted,#6b6b80)}
.gp-btn--active{background:var(--accent,#3b5bdb);border-color:var(--accent,#3b5bdb);color:#fff}
/* Doc panel */
.gp-doc{display:flex;flex-direction:column;min-height:0;height:100%}
.gp-doc-title{font-weight:600}
.gp-doc-mode{font-size:.8rem;padding:.2rem .3rem;border:1px solid var(--border,#e2e2ea);border-radius:5px;
  background:var(--panel,#fff);color:var(--fg,#1d1d28)}
/* Give the mode selector and the reload/close buttons the same height so the
   Doc panel toolbar controls line up. */
.gp-doc .gp-doc-ctl{height:28px;box-sizing:border-box;display:inline-flex;align-items:center;
  padding-top:0;padding-bottom:0;line-height:1}
.gp-doc-body{flex:1;min-height:0;overflow:auto}
.gp-doc-body .readonly-code{height:100%}
.gp-doc-rendered{padding:10px 14px;font-size:13px;line-height:1.55}
.gp-doc-rendered h1{font-size:1.5rem}.gp-doc-rendered h2{font-size:1.2rem}.gp-doc-rendered h3{font-size:1.05rem}
.gp-doc-rendered pre{background:var(--code-bg,rgba(125,125,125,.12));padding:.7rem .9rem;border-radius:8px;overflow:auto}
.gp-doc-rendered code{font-family:ui-monospace,monospace;font-size:.88em}
.gp-doc-rendered table{border-collapse:collapse}.gp-doc-rendered th,.gp-doc-rendered td{border:1px solid var(--border,#e2e2ea);padding:.3rem .5rem}
.gad-ide__diag{color:var(--error,#d64545);font-family:ui-monospace,monospace;font-size:12px}

.gnb{padding:12px;overflow:auto;height:100%;background:var(--bg,#fff);color:var(--fg,#1d1d28)}
.gnb-cell{border:1px solid var(--border,#e2e2ea);border-radius:8px;margin-bottom:12px;overflow:hidden}
/* Editor + (optional) Doc panel side by side within a cell. */
.gnb-cell-main{display:grid;grid-template-columns:1fr;min-height:0}
.gnb-cell-main--doc{grid-template-columns:1fr 1fr}
.gnb-doc{border-left:1px solid var(--border,#e2e2ea);min-width:0;display:flex;flex-direction:column;overflow:hidden}
.gnb-doc .gp-doc{height:100%}
.gnb-editor{height:220px}
.gnb-editor .cm-editor{height:100%}
.gnb-bar{display:flex;align-items:center;gap:8px;flex-wrap:wrap;padding:4px 8px;
  border-top:1px solid var(--border,#e2e2ea)}
.gnb-out{padding:6px 10px;border-top:1px solid var(--border,#e2e2ea)}
`;

/** PlaygroundStyles injects the shared Playground/Notebook CSS into <head> once
 * (persists across mounts; SSR-safe no-op when document is unavailable). */
export function PlaygroundStyles() {
  useEffect(() => {
    if (typeof document === "undefined" || document.getElementById(STYLE_ID)) return;
    const el = document.createElement("style");
    el.id = STYLE_ID;
    el.textContent = CSS;
    document.head.appendChild(el);
  }, []);
  return null;
}
