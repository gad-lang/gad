// Docs dockview panel: two views of the open file's documentation — its doc
// comments (rendered, auto-updating), or generated documentation via the shared
// DocPanel (Render Markdown/HTML, and Markdown/HTML/JSON/YAML source).
import { defineComponent, inject, onMounted, ref, watch } from "vue";
import { VBtn, VBtnToggle } from "../vuetify";
import { IdeControllerKey } from "../controller";
import DocPanel from "../DocPanel";

// docSourceType maps the open file path to the doc dialect.
function docSourceType(path: string): string {
  if (path.endsWith(".gadx")) return "gadx";
  if (path.endsWith(".gadt")) return "gadTemplate";
  return "gad";
}

export default defineComponent({
  name: "PanelDocs",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const view = ref<"comments" | "generate">("comments");
    onMounted(() => void ctx.refreshDoc());
    watch(() => ctx.openPath.value, () => void ctx.refreshDoc());
    return () => (
      <div class="pnl">
        <div class="pnl-head">
          <span class="text-caption font-weight-medium">DOCS</span>
          <span style={{ display: "flex", gap: "4px", alignItems: "center" }}>
            <VBtnToggle
              modelValue={view.value}
              {...{ "onUpdate:modelValue": (v: unknown) => { if (v) view.value = v as "comments" | "generate"; } }}
              density="compact"
              variant="outlined"
              mandatory
            >
              <VBtn size="x-small" value="comments">Comments</VBtn>
              <VBtn size="x-small" value="generate">Generate</VBtn>
            </VBtnToggle>
            {view.value === "comments" && (
              <VBtn size="x-small" variant="text" icon="mdi-refresh" title="Refresh" onClick={() => ctx.refreshDoc()} />
            )}
          </span>
        </div>
        {view.value === "generate" ? (
          <div class="pnl-body" style={{ padding: 0 }}>
            <DocPanel
              doc={ctx.api.docGen}
              source={() => ctx.source.value}
              sourceType={docSourceType(ctx.openPath.value)}
            />
          </div>
        ) : (
          <div class="pnl-body gad-ide__doc" innerHTML={ctx.docHtml.value} />
        )}
      </div>
    );
  },
});
