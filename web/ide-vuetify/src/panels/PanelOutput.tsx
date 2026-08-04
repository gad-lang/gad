// Output dockview panel: STDOUT/STDERR from Run and the debugger, plus errors.
import { defineComponent, inject } from "vue";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelOutput",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    return () => {
      const run = ctx.runRes.value;
      return (
        <div class="pnl">
          <div class="pnl-body">
            {run && (
              <>
                {run.stdout && <pre class="pnl-out">{run.stdout}</pre>}
                {run.stderr && <pre class="pnl-out pnl-out--err">{run.stderr}</pre>}
                {run.ok && run.result && <div class="pnl-return">⇦ {run.result}</div>}
                {run.diagnostics.map((d, i) => (
                  <div key={i} class="pnl-diag">{d.line}:{d.column} {d.message}</div>
                ))}
              </>
            )}
            {ctx.dbgOutput.value && <pre class="pnl-out">{ctx.dbgOutput.value}</pre>}
            {(ctx.snap.value?.diagnostics ?? []).map((d, i) => (
              <div key={"s" + i} class="pnl-diag">{d.line}:{d.column} {d.message}</div>
            ))}
            {!run && !ctx.dbgOutput.value && <div class="text-medium-emphasis">Run or debug to see output.</div>}
          </div>
        </div>
      );
    };
  },
});
