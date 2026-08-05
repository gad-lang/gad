// Editor dockview panel: the CodeMirror editor bound to the open file.
import { defineComponent, inject } from "vue";
import GadEditor from "../GadEditor";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelEditor",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    return () => (
      <div class="pnl">
        <div class="pnl-head">
          <span class="text-caption pnl-ellipsis">{ctx.openPath.value || "(no file)"}</span>
        </div>
        <div class="pnl-editor">
          {ctx.openPath.value ? (
            <GadEditor
              modelValue={ctx.source.value}
              {...{ "onUpdate:modelValue": (v: string) => (ctx.source.value = v) }}
              breakpoints={ctx.breakpoints.value}
              {...{ "onUpdate:breakpoints": (b: number[]) => (ctx.breakpoints.value = b) }}
              path={ctx.openPath.value}
              dark={ctx.dark.value}
              diagnose={ctx.diagnose}
              debugLine={ctx.debugLine.value}
              debugColumn={ctx.debugColumn.value}
              getLocals={ctx.getLocals}
              gotoLine={ctx.gotoTarget.value.line}
              gotoSeq={ctx.gotoTarget.value.seq}
              onBreakpointContext={(line: number) => ctx.openBpCondition(ctx.openPath.value, line)}
            />
          ) : (
            <div class="pa-4 text-medium-emphasis">Select or create a file to begin.</div>
          )}
        </div>
      </div>
    );
  },
});
