// DocPanel — generates and displays the documentation for a source buffer,
// driven through the injected GadRunner.doc. A mode selector picks between the
// rendered doc (Markdown or HTML) and the generated source (Markdown, HTML,
// JSON, YAML); the three source views are shown with syntax highlighting. Shared
// by the Playground's toggleable Doc pane and the IDE's Doc panel.
import { useCallback, useEffect, useRef, useState } from "react";
import { ReadonlyCode, type ReadonlyLanguage } from "./ReadonlyCode";
import { PlaygroundStyles } from "./playgroundStyles";
import { renderDocMarkdown } from "./docMarkdown";
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
  dark?: boolean;
  /** Optional header slot (e.g. a close button in the Playground). */
  header?: React.ReactNode;
}

/** DocPanel renders the doc generator + viewer. */
export function DocPanel({ doc, source, sourceType, dark = false, header }: DocPanelProps) {
  const [mode, setMode] = useState<DocMode>("render-md");
  const [res, setRes] = useState<DocResult | null>(null);
  const [busy, setBusy] = useState(false);
  const seq = useRef(0);

  const generate = useCallback(
    async (m: DocMode) => {
      const id = ++seq.current;
      setBusy(true);
      try {
        const r = await doc(source(), sourceType, m);
        if (id === seq.current) setRes(r);
      } finally {
        if (id === seq.current) setBusy(false);
      }
    },
    [doc, source, sourceType],
  );

  // Generate on mount and whenever the mode changes.
  useEffect(() => {
    void generate(mode);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mode]);

  return (
    <div className="gp-doc">
      <PlaygroundStyles />
      <div className="gp-pane-head">
        <span className="gp-doc-title">Doc</span>
        <span className="gp-actions">
          <select
            className="gp-doc-mode"
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
          <button type="button" className="gp-btn" disabled={busy} onClick={() => generate(mode)}>
            {busy ? "…" : "↻"}
          </button>
          {header}
        </span>
      </div>
      <div className="gp-doc-body">
        <DocView mode={mode} res={res} dark={dark} />
      </div>
    </div>
  );
}

function DocView({ mode, res, dark }: { mode: DocMode; res: DocResult | null; dark?: boolean }) {
  if (!res) return <p className="gp-muted">Generating documentation…</p>;
  if (!res.ok || res.error) return <div className="gp-error gad-ide__diag">{res.error || "doc failed"}</div>;

  switch (mode) {
    case "render-md":
      return (
        <div
          className="gp-doc-rendered"
          // renderDocMarkdown returns sanitized HTML from the generated Markdown.
          dangerouslySetInnerHTML={{ __html: renderDocMarkdown(res.markdown ?? "") }}
        />
      );
    case "render-html":
      return <div className="gp-doc-rendered" dangerouslySetInnerHTML={{ __html: res.html ?? "" }} />;
    case "md":
      return <ReadonlyCode value={res.markdown ?? ""} language="markdown" dark={dark} />;
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
