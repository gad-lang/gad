// Locals dockview panel: EVALUATE (always on top) and LOCALS, each row with an
// inspect icon that opens the value tree navigator in a dialog.
import { defineComponent, inject, ref } from "vue";
import { VBtn, VCard, VCardActions, VCardText, VCardTitle, VDialog, VSpacer, VTextField } from "../vuetify";
import InspectorNode from "../InspectorNode";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelLocals",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const open = ref(false);
    const expr = ref("");
    const label = ref("");
    const openInspect = (e: string, l: string) => {
      if (!e.trim()) return;
      expr.value = e;
      label.value = l;
      open.value = true;
    };
    return () => (
      <div class="pnl">
        <div class="pnl-body">
          {/* EVALUATE — always above LOCALS */}
          <div class="d-flex align-center" style={{ gap: "4px" }}>
            <VTextField
              modelValue={ctx.evalExpr.value}
              {...{ "onUpdate:modelValue": (v: string) => (ctx.evalExpr.value = v) }}
              label="Evaluate"
              density="compact"
              variant="outlined"
              hideDetails
              onKeyup={(e: KeyboardEvent) => e.key === "Enter" && ctx.doEval()}
            />
            <VBtn size="small" variant="tonal" disabled={!ctx.stopped.value} onClick={() => ctx.doEval()}>
              Eval
            </VBtn>
            <VBtn
              size="x-small"
              variant="text"
              icon="mdi-file-tree-outline"
              title="Inspect value"
              disabled={!ctx.evalExpr.value.trim()}
              onClick={() => openInspect(ctx.evalExpr.value, ctx.evalExpr.value)}
            />
          </div>
          {ctx.evalOut.value && <pre class="pnl-out">{ctx.evalOut.value}</pre>}

          {/* LOCALS */}
          <div class="text-caption text-medium-emphasis mt-2 mb-1">LOCALS</div>
          <ul class="pnl-list">
            {(ctx.snap.value?.locals ?? []).map((v, i) => (
              <li key={i} class="pnl-local">
                <span class="pnl-ellipsis">
                  {v.name} = {v.value} <span class="text-medium-emphasis">({v.type})</span>
                </span>
                <VBtn
                  size="x-small"
                  variant="text"
                  icon="mdi-file-tree-outline"
                  title="Inspect value"
                  onClick={() => openInspect(v.name, v.name)}
                />
              </li>
            ))}
            {!ctx.snap.value?.locals?.length && <li class="text-medium-emphasis">(none)</li>}
          </ul>
        </div>

        <VDialog
          modelValue={open.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (open.value = v) }}
          maxWidth="720"
          scrollable
        >
          <VCard>
            <VCardTitle class="text-subtitle-1">Inspect — {label.value}</VCardTitle>
            <VCardText style={{ maxHeight: "60vh", overflow: "auto" }}>
              {open.value && (
                <InspectorNode key={expr.value} inspect={ctx.inspectFn} label={label.value} expr={expr.value} root />
              )}
            </VCardText>
            <VCardActions>
              <VSpacer />
              <VBtn onClick={() => (open.value = false)}>Close</VBtn>
            </VCardActions>
          </VCard>
        </VDialog>
      </div>
    );
  },
});
