// DocPanel — generates and displays the documentation for a source buffer,
// driven through the injected GadRunner.doc. A mode selector picks between the
// rendered doc (Markdown or HTML) and the generated source (Markdown, HTML,
// JSON, YAML); the three source views are shown with syntax highlighting. Shared
// by the Playground's toggleable Doc pane and the IDE's Doc panel.
import { useCallback, useEffect, useRef, useState } from "react";
import { ReadonlyCode, type ReadonlyLanguage } from "./ReadonlyCode";
import { PlaygroundStyles } from "./playgroundStyles";
import { renderDocMarkdown } from "./docMarkdown";
import { resolveDocPaths, type DocRootFn } from "./docPaths";
import type { DocMode, DocResult } from "./types";

/** DocFn generates documentation for source in the given mode. */
export type DocFn = (source: string, sourceType: string, mode: DocMode) => Promise<DocResult>;

const MODES: { value: DocMode; label: string }[] = [
  { value: "render-md", label: "Render Markdown" },
  { value: "md", label: "Markdown" },
  { value: "render-html", label: "Render HTML" },
  { value: "html", label: "HTML" },
  { value: "json", label: "Encode JSON" },
  { value: "yaml", label: "Encode YAML" },
];

export interface DocPanelProps {
  /** Generates the documentation (markdown/html/encoded per mode). */
  doc: DocFn;
  /** Returns the current source to document. */
  source: () => string;
  sourceType: string;
  /** The open file's workspace path, used to resolve doc-comment asset/link
   * references against the generated `doc/` tree (see resolveDocPaths). */
  docPath?: string;
  /** Optional override for the doc root used to resolve those references, chosen
   * per source path (e.g. to serve assets from a different URI). Defaults to `doc`. */
  docRootFor?: DocRootFn;
  dark?: boolean;
  /** Bumped by the host whenever the source changes; the panel re-generates so it
   * stays in sync without a manual reload. */
  revision?: number;
  /** Called when a documented symbol is clicked in the rendered doc, to focus the
   * editor at its source position. */
  onNavigate?: (line: number, column: number) => void;
  /** Optional header slot (e.g. a close button in the Playground). */
  header?: React.ReactNode;
}

// stripPos removes the `{data-src-pos="L,C"}` heading markers so the plain
// Markdown source view stays clean.
const stripPos = (md: string) => md.replace(/\s*\{data-src-pos="\d+,\d+"\}/g, "");

/** DocPanel renders the doc generator + viewer. */
export function DocPanel({ doc, source, sourceType, docPath = "", docRootFor, dark = false, revision = 0, onNavigate, header }: DocPanelProps) {
  const [mode, setMode] = useState<DocMode>("render-md");
  const [res, setRes] = useState<DocResult | null>(null);
  const [busy, setBusy] = useState(false);
  const seq = useRef(0);
  // Keep the latest source getter/doc/type in a ref so the (debounced) effect
  // reads them fresh without re-subscribing on every host render.
  const live = useRef({ doc, source, sourceType });
  live.current = { doc, source, sourceType };

  const generate = useCallback(async () => {
    const id = ++seq.current;
    setBusy(true);
    try {
      const { doc, source, sourceType } = live.current;
      const r = await doc(source(), sourceType, mode);
      if (id === seq.current) setRes(r);
    } finally {
      if (id === seq.current) setBusy(false);
    }
  }, [mode]);

  // Regenerate on mount, on mode change, and (debounced) whenever the source
  // revision or dialect changes, so the panel tracks edits automatically.
  useEffect(() => {
    const t = setTimeout(() => void generate(), 250);
    return () => clearTimeout(t);
  }, [generate, revision, sourceType]);

  return (
    <div className="gp-doc">
      <PlaygroundStyles />
      <div className="gp-pane-head">
        <span className="gp-doc-title">Doc</span>
        <span className="gp-actions">
          <select
            className="gp-doc-mode gp-doc-ctl"
            value={mode}
            onChange={(e) => setMode(e.target.value as DocMode)}
            disabled={busy}
          >
            {MODES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
          <button
            type="button"
            className="gp-btn gp-doc-ctl"
            disabled={busy}
            title="Reload documentation"
            onClick={() => generate()}
          >
            {busy ? "…" : "↻"}
          </button>
          {header}
        </span>
      </div>
      <div
        className="gp-doc-body"
        onClick={(e) => {
          if (!onNavigate) return;
          const el = (e.target as HTMLElement).closest("[data-src-pos]");
          const pos = el?.getAttribute("data-src-pos");
          if (pos) {
            const [line, col] = pos.split(",").map((n) => parseInt(n, 10));
            if (line) onNavigate(line, col || 1);
          }
        }}
      >
        <DocView mode={mode} res={res} dark={dark} docPath={docPath} docRootFor={docRootFor} />
      </div>
    </div>
  );
}

function DocView({ mode, res, dark, docPath = "", docRootFor }: { mode: DocMode; res: DocResult | null; dark?: boolean; docPath?: string; docRootFor?: DocRootFn }) {
  if (!res) return <p className="gp-muted">Generating documentation…</p>;
  if (!res.ok || res.error) return <div className="gp-error gad-ide__diag">{res.error || "doc failed"}</div>;

  switch (mode) {
    case "render-md":
      return (
        <div
          className="gp-doc-rendered"
          // renderDocMarkdown returns sanitized HTML from the generated Markdown.
          dangerouslySetInnerHTML={{ __html: resolveDocPaths(renderDocMarkdown(res.markdown ?? ""), docPath, docRootFor) }}
        />
      );
    case "render-html":
      return <div className="gp-doc-rendered" dangerouslySetInnerHTML={{ __html: resolveDocPaths(res.html ?? "", docPath, docRootFor) }} />;
    case "md":
      return <ReadonlyCode value={stripPos(res.markdown ?? "")} language="markdown" dark={dark} />;
    case "html":
      return <ReadonlyCode value={res.html ?? ""} language="html" dark={dark} />;
    case "json":
      return <ReadonlyCode value={res.text ?? ""} language={"json" as ReadonlyLanguage} dark={dark} />;
    case "yaml":
      return <ReadonlyCode value={res.text ?? ""} language={"yaml" as ReadonlyLanguage} dark={dark} />;
    default:
      return null;
  }
}
