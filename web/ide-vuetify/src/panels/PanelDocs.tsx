// Docs dockview panel: the rendered documentation of the open file (auto-updates
// when the file changes; a refresh button re-extracts on demand).
import { defineComponent, inject, onMounted, watch } from "vue";
import { VBtn } from "../vuetify";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelDocs",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    onMounted(() => void ctx.refreshDoc());
    watch(() => ctx.openPath.value, () => void ctx.refreshDoc());
    return () => (
      <div class="pnl">
        <div class="pnl-head">
          <span class="text-caption font-weight-medium">DOCS</span>
          <VBtn size="x-small" variant="text" icon="mdi-refresh" title="Refresh" onClick={() => ctx.refreshDoc()} />
        </div>
        <div class="pnl-body gad-ide__doc" innerHTML={ctx.docHtml.value} />
      </div>
    );
  },
});
