// Call Stack dockview panel: the frames of the current paused debug session.
import { defineComponent, inject } from "vue";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelCallStack",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    return () => (
      <div class="pnl">
        <div class="pnl-body">
          <ul class="pnl-list">
            {(ctx.snap.value?.frames ?? []).map((f, i) => (
              <li key={i}>
                {f.name} <span class="text-medium-emphasis">@ {f.line}:{f.column}</span>
              </li>
            ))}
            {!ctx.snap.value?.frames?.length && <li class="text-medium-emphasis">(not paused)</li>}
          </ul>
        </div>
      </div>
    );
  },
});
