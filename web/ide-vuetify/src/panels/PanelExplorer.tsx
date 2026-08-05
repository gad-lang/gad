// Explorer dockview panel: the workspace file tree with create/delete/reset.
import { defineComponent, inject } from "vue";
import { VBtn, VIcon } from "../vuetify";
import { IdeControllerKey } from "../controller";

export default defineComponent({
  name: "PanelExplorer",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    return () => (
      <div class="pnl">
        <div class="pnl-head">
          <span class="text-caption font-weight-medium">EXPLORER</span>
          <div>
            <VBtn size="x-small" variant="text" icon="mdi-file-plus-outline" title="New file" onClick={() => ctx.newFile()} />
            <VBtn size="x-small" variant="text" icon="mdi-folder-plus-outline" title="New folder" onClick={() => ctx.newDir()} />
            <VBtn size="x-small" variant="text" icon="mdi-delete-outline" title="Delete open file"
              disabled={!ctx.openPath.value} onClick={() => ctx.removeOpen()} />
            <VBtn
              size="x-small"
              variant="text"
              icon={ctx.showHidden.value ? "mdi-eye-off-outline" : "mdi-eye-outline"}
              title={ctx.showHidden.value ? "Hide hidden files" : "Show hidden files"}
              onClick={() => ctx.toggleHidden()}
            />
            {ctx.canReset.value && (
              <VBtn size="x-small" variant="text" icon="mdi-backup-restore" title="Reset changes" onClick={() => ctx.reset()} />
            )}
          </div>
        </div>
        <div class="pnl-body">
          {ctx.rows.value.map((row) => (
            <div
              key={row.node.path}
              class={["pnl-row", { "pnl-row--active": row.node.path === ctx.openPath.value }]}
              style={{ paddingLeft: 6 + row.depth * 14 + "px" }}
              onClick={() => (row.node.dir ? ctx.toggleDir(row.node.path) : ctx.openFile(row.node.path))}
            >
              <VIcon size="16" class="mr-1">
                {row.node.dir
                  ? ctx.isExpanded(row.node.path)
                    ? "mdi-chevron-down"
                    : "mdi-chevron-right"
                  : "mdi-file-outline"}
              </VIcon>
              <span class="pnl-ellipsis">{row.node.name}</span>
            </div>
          ))}
        </div>
      </div>
    );
  },
});
