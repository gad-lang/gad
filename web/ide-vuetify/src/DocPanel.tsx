// DocPanel — generates and displays the documentation for a source buffer,
// driven through the injected GadRunner.doc. A mode selector picks between the
// rendered doc (Markdown or HTML) and the generated source (Markdown, HTML,
// JSON, YAML); the three source views are highlighted with Prism. The Vuetify
// counterpart of the React DocPanel, shared by the Playground's toggleable Doc
// pane and the IDE's Doc panel.
import { defineComponent, ref, watch, type PropType } from "vue";
import Prism from "prismjs";
// Grammars for the source views (markup/HTML is in Prism core).
import "prismjs/components/prism-json";
import "prismjs/components/prism-yaml";
import "prismjs/components/prism-markdown";
import { VBtn, VSelect } from "./vuetify";
import { renderDocMarkdown } from "./docMarkdown";
import { resolveDocPaths, type DocRootFn } from "./docPaths";
import type { DocMode, DocResult } from "./types";

/** DocFn generates documentation for source in the given mode. */
export type DocFn = (source: string, sourceType: string, mode: DocMode) => Promise<DocResult>;

const MODES: { title: string; value: DocMode }[] = [
  { title: "Render Markdown", value: "render-md" },
  { title: "Markdown", value: "md" },
  { title: "Render HTML", value: "render-html" },
  { title: "HTML", value: "html" },
  { title: "Encode JSON", value: "json" },
  { title: "Encode YAML", value: "yaml" },
];

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

// stripPos removes `{data-src-pos="L,C"}` heading markers for the clean Markdown
// source view.
const stripPos = (md: string) => md.replace(/\s*\{data-src-pos="\d+,\d+"\}/g, "");

// highlight returns Prism-highlighted HTML for the given Prism language, falling
// back to escaped plain text when the grammar is unavailable.
function highlight(src: string, lang: string): string {
  const grammar = Prism.languages[lang];
  return grammar ? Prism.highlight(src, grammar, lang) : escapeHtml(src);
}

export default defineComponent({
  name: "DocPanel",
  props: {
    doc: { type: Function as PropType<DocFn>, required: true },
    source: { type: Function as PropType<() => string>, required: true },
    sourceType: { type: String, default: "" },
    // The open file's workspace path, used to resolve doc-comment asset/link
    // references against the generated `doc/` tree (see resolveDocPaths).
    docPath: { type: String, default: "" },
    // Optional override for the doc root used to resolve those references, chosen
    // per source path (e.g. to serve assets from a different URI). Defaults to `doc`.
    docRootFor: { type: Function as PropType<DocRootFn>, default: undefined },
    // Bumped by the host on every source edit so the panel re-generates in sync.
    revision: { type: Number, default: 0 },
    // Called with a documented symbol's source position when it is clicked.
    onNavigate: { type: Function as PropType<(line: number, column: number) => void>, default: undefined },
  },
  setup(props, { slots }) {
    const mode = ref<DocMode>("render-md");
    const res = ref<DocResult | null>(null);
    const busy = ref(false);
    let seq = 0;
    let debounce: ReturnType<typeof setTimeout> | undefined;

    async function generate() {
      const id = ++seq;
      busy.value = true;
      try {
        const r = await props.doc(props.source(), props.sourceType, mode.value);
        if (id === seq) res.value = r;
      } finally {
        if (id === seq) busy.value = false;
      }
    }
    // Regenerate on mode/dialect change immediately, and (debounced) whenever the
    // source revision changes, so the panel tracks edits automatically.
    watch([mode, () => props.sourceType], generate, { immediate: true });
    watch(() => props.revision, () => {
      clearTimeout(debounce);
      debounce = setTimeout(generate, 250);
    });

    function body() {
      const r = res.value;
      if (!r) return <p class="gp-muted">Generating documentation…</p>;
      if (!r.ok || r.error) return <div class="gp-error gad-ide__diag">{r.error || "doc failed"}</div>;
      switch (mode.value) {
        case "render-md":
          return <div class="gp-doc-rendered" innerHTML={resolveDocPaths(renderDocMarkdown(r.markdown ?? ""), props.docPath, props.docRootFor)} />;
        case "render-html":
          return <div class="gp-doc-rendered" innerHTML={resolveDocPaths(r.html ?? "", props.docPath, props.docRootFor)} />;
        case "md":
          return <pre class="gp-doc-src language-markdown" innerHTML={highlight(stripPos(r.markdown ?? ""), "markdown")} />;
        case "html":
          return <pre class="gp-doc-src language-markup" innerHTML={highlight(r.html ?? "", "markup")} />;
        case "json":
          return <pre class="gp-doc-src language-json" innerHTML={highlight(r.text ?? "", "json")} />;
        case "yaml":
          return <pre class="gp-doc-src language-yaml" innerHTML={highlight(r.text ?? "", "yaml")} />;
        default:
          return null;
      }
    }

    return () => (
      <div class="gp-doc">
        <div class="gp-pane-head">
          <span class="gp-doc-title">Doc</span>
          <span class="gp-actions gp-doc-actions">
            <VSelect
              class="gp-doc-ctl"
              modelValue={mode.value}
              {...{ "onUpdate:modelValue": (v: unknown) => (mode.value = (v as DocMode) ?? "render-md") }}
              items={MODES}
              density="compact"
              variant="outlined"
              hideDetails
              disabled={busy.value}
              style={{ maxWidth: "180px" }}
            />
            <VBtn class="gp-doc-ctl" size="small" height="32" variant="tonal" loading={busy.value} title="Reload documentation" onClick={generate}>↻</VBtn>
            {slots.default?.()}
          </span>
        </div>
        <div
          class="gp-doc-body"
          onClick={(e: MouseEvent) => {
            if (!props.onNavigate) return;
            const el = (e.target as HTMLElement).closest("[data-src-pos]");
            const pos = el?.getAttribute("data-src-pos");
            if (pos) {
              const [line, col] = pos.split(",").map((n) => parseInt(n, 10));
              if (line) props.onNavigate(line, col || 1);
            }
          }}
        >
          {body()}
        </div>
      </div>
    );
  },
});
