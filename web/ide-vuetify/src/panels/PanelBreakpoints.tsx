// Breakpoints dockview panel: two tabs — CURRENT FILE and ALL. Clicking an entry
// navigates to its source position; the delete button removes it; the edit
// (condition) button opens the condition dialog. A right-click on the editor
// gutter opens the same dialog (wired in GadIde).
import { defineComponent, inject, ref } from "vue";
import { VBtn, VIcon, VTab, VTabs } from "../vuetify";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelBreakpoints",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const tab = ref<"current" | "all">("current");

    const row = (path: string, line: number) => {
      const meta = ctx.bpMetaFor(path)[line] || {};
      return (
        <div key={path + ":" + line} class={["bp-row", { "bp-row--off": meta.disabled }]}>
          <span class="bp-entry" title="Go to line" onClick={() => ctx.gotoFileLine(path, line)}>
            <VIcon size="14" color="error" class="mr-1">mdi-circle</VIcon>
            {tab.value === "all" ? path + ":" + line : "line " + line}
            {meta.condition ? <em class="bp-cond"> if {meta.condition}</em> : null}
          </span>
          <span class="bp-actions">
            <VBtn size="x-small" variant="text" icon="mdi-pencil-outline" title="Edit condition"
              onClick={() => ctx.openBpCondition(path, line)} />
            <VBtn size="x-small" variant="text" icon="mdi-close" title="Remove"
              onClick={() => ctx.removeBreakpoint(path, line)} />
          </span>
        </div>
      );
    };

    return () => {
      const cur = ctx.openPath.value;
      const all = ctx.allBreakpoints();
      return (
        <div class="pnl">
          <VTabs modelValue={tab.value} {...{ "onUpdate:modelValue": (v: unknown) => (tab.value = v as "current" | "all") }} density="compact">
            <VTab value="current">Current file</VTab>
            <VTab value="all">All</VTab>
          </VTabs>
          <div class="pnl-body">
            {tab.value === "current"
              ? cur
                ? ctx.bpFor(cur).length
                  ? ctx.bpFor(cur).map((l) => row(cur, l))
                  : <div class="text-medium-emphasis">No breakpoints in {cur.split("/").pop()}.</div>
                : <div class="text-medium-emphasis">No file open.</div>
              : Object.keys(all).length
                ? Object.entries(all).flatMap(([p, ls]) => ls.map((l) => row(p, l)))
                : <div class="text-medium-emphasis">No breakpoints.</div>}
          </div>
        </div>
      );
    };
  },
});
