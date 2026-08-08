// GadPlayground — a two-pane playground (source | output): format, format &
// apply, or run the source, executed through the injected GadRunner. A dialect
// selector switches the example between plain Gad, a Gad template (.gadt) and
// Gadx (.gadx); each keeps its own buffer. The React counterpart of the Vuetify
// playground, with a similar layout (editor left, output right).
import { useMemo, useRef, useState } from "react";
import type { DiagnoseFn } from "@gad-lang/codemirror-gad";
import { Editor, type EditorHandle, type EditorLanguage } from "./Editor";
import { PlaygroundStyles } from "./playgroundStyles";
import { DocPanel } from "./DocPanel";
import type { FormatResult, GadRunner, RunResult } from "./types";

type Dialect = "gad" | "gadt" | "gadx";

const SAMPLES: Record<Dialect, string> = {
  gad: `// edit me — errors are underlined as you type
param *args

greet := func(name; greeting="Hello") {
  return greeting + ", " + name
}

squares := [n*n for n in [1,2,3,4] if n>1]
println(greet("Gad"), squares)
return squares
`,
  gadt: `{% var (name = "Gad", items = [1, 2, 3]) %}
<h1>Hello, {%= name %}!</h1>
<ul>
{% for i in items %}  <li>item {%= i %}</li>
{% end %}</ul>
`,
  gadx: `@main
    h1 Hello Gadx
    ul
        @for i in [1, 2, 3]
            li item {= i }
`,
};

// Editor language + backend sourceType per dialect.
const LANG: Record<Dialect, EditorLanguage> = { gad: "gad", gadt: "gadt", gadx: "gadx" };
const SOURCE_TYPE: Record<Dialect, string> = { gad: "", gadt: "gadTemplate", gadx: "gadx" };

export interface GadPlaygroundProps {
  runner: GadRunner;
  dark?: boolean;
}

type Output = { kind: "format"; fmt: FormatResult } | { kind: "run"; run: RunResult } | null;

/** GadPlayground renders the two-pane playground. */
type TagEncode = "" | "json" | "yaml";

export function GadPlayground({ runner, dark = false }: GadPlaygroundProps) {
  const [dialect, setDialect] = useState<Dialect>("gad");
  // Each dialect keeps its own buffer so switching examples preserves edits.
  const buffers = useRef<Record<Dialect, string>>({ ...SAMPLES });
  const editorRef = useRef<EditorHandle>(null);
  const [busy, setBusy] = useState(false);
  const [out, setOut] = useState<Output>(null);
  // GADX only: encode the returned tag as JSON/YAML instead of rendering HTML.
  const [tagEncode, setTagEncode] = useState<TagEncode>("");
  // The Doc panel on the right is hidden by default; toggled when the runner
  // supports doc generation.
  const [showDoc, setShowDoc] = useState(false);

  const sourceType = SOURCE_TYPE[dialect];
  const diagnose: DiagnoseFn | undefined = useMemo(
    () => (runner.diagnose ? (src: string) => runner.diagnose!(src, sourceType) : undefined),
    [runner, sourceType],
  );

  const source = () => editorRef.current?.getValue() ?? buffers.current[dialect];

  function switchDialect(d: Dialect) {
    // Persist the current buffer before switching.
    buffers.current[dialect] = source();
    setDialect(d);
    setOut(null);
  }

  async function doFormat(apply: boolean) {
    setBusy(true);
    try {
      const fmt = await runner.format(source(), sourceType);
      setOut({ kind: "format", fmt });
      if (apply && fmt.ok) {
        buffers.current[dialect] = fmt.source;
        editorRef.current?.setValue(fmt.source);
      }
    } finally {
      setBusy(false);
    }
  }
  async function doRun() {
    setBusy(true);
    try {
      const te = dialect === "gadx" ? tagEncode : "";
      setOut({ kind: "run", run: await runner.run(source(), sourceType, te) });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="gp-split">
      <PlaygroundStyles />
      <section className="gp-pane">
        <div className="gp-pane-head">
          <span className="gp-dialect">
            {(["gad", "gadt", "gadx"] as Dialect[]).map((d) => (
              <button
                key={d}
                type="button"
                className={"gp-tab" + (dialect === d ? " gp-tab--active" : "")}
                onClick={() => switchDialect(d)}
              >
                {d === "gad" ? "GAD" : d === "gadt" ? "GAD Template" : "GADx"}
              </button>
            ))}
          </span>
          <span className="gp-actions">
            {dialect === "gadx" && (
              <label className="gp-tagenc" title="GADX output: render HTML, or encode the tag as JSON/YAML">
                Tag:
                <select value={tagEncode} onChange={(e) => setTagEncode(e.target.value as TagEncode)}>
                  <option value="">Render</option>
                  <option value="json">JSON</option>
                  <option value="yaml">YAML</option>
                </select>
              </label>
            )}
            <button type="button" className="gp-btn" disabled={busy} onClick={() => doFormat(false)}>Format</button>
            <button type="button" className="gp-btn" disabled={busy} onClick={() => doFormat(true)}>Format &amp; apply</button>
            <button type="button" className="gp-btn gp-btn--primary" disabled={busy} onClick={doRun}>▶ Run</button>
            {runner.doc && (
              <button
                type="button"
                className={"gp-btn" + (showDoc ? " gp-btn--active" : "")}
                aria-pressed={showDoc}
                onClick={() => setShowDoc((v) => !v)}
                title="Toggle the documentation panel"
              >
                Doc
              </button>
            )}
          </span>
        </div>
        <div className="gp-editor">
          <Editor
            key={dialect}
            ref={editorRef}
            initialDoc={buffers.current[dialect]}
            language={LANG[dialect]}
            dark={dark}
            diagnose={diagnose}
            onChange={(v) => (buffers.current[dialect] = v)}
          />
        </div>
      </section>
      <section className="gp-pane">
        <div className="gp-pane-head">Output</div>
        <div className="gp-pane-body">
          {!out && <p className="gp-muted">Format or run the source on the left.</p>}
          {out?.kind === "format" && <FormatView fmt={out.fmt} />}
          {out?.kind === "run" && <RunView run={out.run} />}
        </div>
      </section>
      {runner.doc && showDoc && (
        <section className="gp-pane gp-pane--doc">
          <DocPanel
            doc={runner.doc}
            source={source}
            sourceType={sourceType}
            dark={dark}
            header={
              <button type="button" className="gp-btn" title="Hide the doc panel" onClick={() => setShowDoc(false)}>
                ✕
              </button>
            }
          />
        </section>
      )}
    </div>
  );
}

function FormatView({ fmt }: { fmt: FormatResult }) {
  return fmt.ok ? (
    <pre className="gp-out">{fmt.source}</pre>
  ) : (
    <div>
      {(fmt.diagnostics ?? []).map((d, i) => (
        <div className="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
      ))}
    </div>
  );
}

function RunView({ run }: { run: RunResult }) {
  return (
    <div className={run.ok ? "" : "gp-error"}>
      {run.stdout && <pre className="gp-out">{run.stdout}</pre>}
      {run.stderr && <pre className="gp-out gad-ide__diag">{run.stderr}</pre>}
      {run.ok && run.result && <div className="gp-return">↩ {run.result}</div>}
      {(run.diagnostics ?? []).map((d, i) => (
        <div className="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
      ))}
    </div>
  );
}
