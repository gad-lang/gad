// Explorer dockview panel: the workspace file tree with create/upload/delete/
// reset. Files (and directories) can be uploaded via the button or dragged and
// dropped anywhere onto the panel — both go through the controller's upload().
import { defineComponent, inject, ref } from "vue";
import { VBtn, VIcon } from "../vuetify";
import { IdeControllerKey } from "../controller";
import { filesFromDataTransfer, filesFromInput } from "../upload";

export default defineComponent({
  name: "PanelExplorer",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const fileInput = ref<HTMLInputElement>();
    const dragOver = ref(false);

    async function onDrop(e: DragEvent) {
      dragOver.value = false;
      if (!e.dataTransfer) return;
      await ctx.upload(await filesFromDataTransfer(e.dataTransfer));
    }
    async function onPick(e: Event) {
      const input = e.target as HTMLInputElement;
      await ctx.upload(await filesFromInput(input.files));
      input.value = "";
    }

    return () => (
      <div class="pnl">
        <div class="pnl-head">
          <span class="text-caption font-weight-medium">EXPLORER</span>
          <div>
            <VBtn size="x-small" variant="text" icon="mdi-file-plus-outline" title="New file" onClick={() => ctx.newFile()} />
            <VBtn size="x-small" variant="text" icon="mdi-folder-plus-outline" title="New folder" onClick={() => ctx.newDir()} />
            <VBtn size="x-small" variant="text" icon="mdi-upload-outline" title="Upload files" onClick={() => fileInput.value?.click()} />
            <VBtn size="x-small" variant="text" icon="mdi-link-variant" title="Import from URL" onClick={() => (ctx.urlDialog.value = true)} />
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
        <div
          class={["pnl-body", { "pnl-body--drop": dragOver.value }]}
          onDragover={(e: DragEvent) => { e.preventDefault(); dragOver.value = true; }}
          onDragleave={() => (dragOver.value = false)}
          onDrop={(e: DragEvent) => { e.preventDefault(); void onDrop(e); }}
        >
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
          {dragOver.value && <div class="pnl-drop-hint">Drop files or folders to upload</div>}
        </div>
        <input ref={fileInput} type="file" multiple style={{ display: "none" }} onChange={onPick} />
      </div>
    );
  },
});
