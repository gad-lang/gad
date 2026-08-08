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
import type { DocMode, DocResult, GadRunner } from "./types";

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

// highlight returns Prism-highlighted HTML for the given Prism language, falling
// back to escaped plain text when the grammar is unavailable.
function highlight(src: string, lang: string): string {
  const grammar = Prism.languages[lang];
  return grammar ? Prism.highlight(src, grammar, lang) : escapeHtml(src);
}

export default defineComponent({
  name: "DocPanel",
  props: {
    runner: { type: Object as PropType<GadRunner>, required: true },
    source: { type: Function as PropType<() => string>, required: true },
    sourceType: { type: String, default: "" },
  },
  setup(props, { slots }) {
    const mode = ref<DocMode>("render-md");
    const res = ref<DocResult | null>(null);
    const busy = ref(false);
    let seq = 0;

    async function generate() {
      if (!props.runner.doc) return;
      const id = ++seq;
      busy.value = true;
      try {
        const r = await props.runner.doc(props.source(), props.sourceType, mode.value);
        if (id === seq) res.value = r;
      } finally {
        if (id === seq) busy.value = false;
      }
    }
    watch(mode, generate, { immediate: true });

    function body() {
      const r = res.value;
      if (!r) return <p class="gp-muted">Generating documentation…</p>;
      if (!r.ok || r.error) return <div class="gp-error gad-ide__diag">{r.error || "doc failed"}</div>;
      switch (mode.value) {
        case "render-md":
          return <div class="gp-doc-rendered" innerHTML={renderDocMarkdown(r.markdown ?? "")} />;
        case "render-html":
          return <div class="gp-doc-rendered" innerHTML={r.html ?? ""} />;
        case "md":
          return <pre class="gp-doc-src language-markdown" innerHTML={highlight(r.markdown ?? "", "markdown")} />;
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
          <span class="gp-actions">
            <VSelect
              modelValue={mode.value}
              {...{ "onUpdate:modelValue": (v: unknown) => (mode.value = (v as DocMode) ?? "render-md") }}
              items={MODES}
              density="compact"
              variant="outlined"
              hideDetails
              disabled={busy.value}
              style={{ maxWidth: "180px" }}
            />
            <VBtn size="small" variant="tonal" loading={busy.value} onClick={generate}>↻</VBtn>
            {slots.default?.()}
          </span>
        </div>
        <div class="gp-doc-body">{body()}</div>
      </div>
    );
  },
});
