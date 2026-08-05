// Explorer dockview panel: the workspace file tree with create/upload/delete/
// reset. When the host provides an onUpload handler, files (and directories) can
// be uploaded via a dialog (with a target-folder picker) or dragged and dropped
// onto a folder row (or the panel background = root). Without onUpload, the
// upload button, URL import and drag-drop are all omitted.
import { defineComponent, inject, ref } from "vue";
import { VBtn, VCard, VCardActions, VCardText, VCardTitle, VDialog, VIcon, VSpacer } from "../vuetify";
import { IdeControllerKey } from "../controller";
import DirTree from "../DirTree";
import { filesFromDataTransfer, filesFromInput } from "../upload";
import type { UploadedFile } from "../api";

export default defineComponent({
  name: "PanelExplorer",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const fileInput = ref<HTMLInputElement>();
    // The path of the directory row currently under a drag (null = none).
    const dropDir = ref<string | null>(null);
    // Upload dialog: the picked files awaiting a chosen target folder.
    const uploadDlg = ref(false);
    const pending = ref<UploadedFile[]>([]);
    const uploadDir = ref("");

    const parentDir = (path: string) => {
      const s = path.lastIndexOf("/");
      return s === -1 ? "" : path.slice(0, s);
    };

    async function onDrop(e: DragEvent, targetDir: string) {
      dropDir.value = null;
      if (!ctx.canUpload.value || !e.dataTransfer) return;
      await ctx.upload(await filesFromDataTransfer(e.dataTransfer), targetDir);
    }
    async function onPick(e: Event) {
      const input = e.target as HTMLInputElement;
      pending.value = await filesFromInput(input.files);
      input.value = "";
      if (pending.value.length) {
        uploadDir.value = "";
        uploadDlg.value = true;
      }
    }
    async function confirmUpload() {
      uploadDlg.value = false;
      await ctx.upload(pending.value, uploadDir.value);
      pending.value = [];
    }

    return () => (
      <div class="pnl">
        <div class="pnl-head">
          <span class="text-caption font-weight-medium">EXPLORER</span>
          <div>
            {ctx.canEdit.value && (
              <>
                <VBtn size="x-small" variant="text" icon="mdi-file-plus-outline" title="New file" onClick={() => ctx.newFile()} />
                <VBtn size="x-small" variant="text" icon="mdi-folder-plus-outline" title="New folder" onClick={() => ctx.newDir()} />
              </>
            )}
            {ctx.canUpload.value && (
              <>
                <VBtn size="x-small" variant="text" icon="mdi-upload-outline" title="Upload files" onClick={() => fileInput.value?.click()} />
                <VBtn size="x-small" variant="text" icon="mdi-link-variant" title="Import from URL" onClick={() => (ctx.urlDialog.value = true)} />
              </>
            )}
            {ctx.canEdit.value && (
              <VBtn size="x-small" variant="text" icon="mdi-delete-outline" title="Delete open file"
                disabled={!ctx.openPath.value} onClick={() => ctx.removeOpen()} />
            )}
            <VBtn
              size="x-small"
              variant="text"
              icon={ctx.showHidden.value ? "mdi-eye-off-outline" : "mdi-eye-outline"}
              title={ctx.showHidden.value ? "Hide hidden files" : "Show hidden files"}
              onClick={() => ctx.toggleHidden()}
            />
            {ctx.canReset.value && ctx.canEdit.value && (
              <VBtn size="x-small" variant="text" icon="mdi-backup-restore" title="Reset changes" onClick={() => ctx.reset()} />
            )}
          </div>
        </div>
        <div
          class={["pnl-body", { "pnl-body--drop": ctx.canUpload.value && dropDir.value === "" }]}
          onDragover={ctx.canUpload.value ? (e: DragEvent) => { e.preventDefault(); dropDir.value = ""; } : undefined}
          onDragleave={ctx.canUpload.value ? () => (dropDir.value = null) : undefined}
          onDrop={ctx.canUpload.value ? (e: DragEvent) => { e.preventDefault(); void onDrop(e, ""); } : undefined}
        >
          {ctx.rows.value.map((row) => {
            const rowDir = row.node.dir ? row.node.path : parentDir(row.node.path);
            return (
              <div
                key={row.node.path}
                class={["pnl-row", {
                  "pnl-row--active": row.node.path === ctx.openPath.value,
                  "pnl-row--drop": ctx.canUpload.value && dropDir.value === rowDir,
                }]}
                style={{ paddingLeft: 6 + row.depth * 14 + "px" }}
                onClick={() => (row.node.dir ? ctx.toggleDir(row.node.path) : ctx.openFile(row.node.path))}
                onDragover={ctx.canUpload.value ? (e: DragEvent) => { e.preventDefault(); e.stopPropagation(); dropDir.value = rowDir; } : undefined}
                onDrop={ctx.canUpload.value ? (e: DragEvent) => { e.preventDefault(); e.stopPropagation(); void onDrop(e, rowDir); } : undefined}
              >
                <VIcon size="16" class="mr-1">
                  {row.node.dir
                    ? ctx.isExpanded(row.node.path)
                      ? "mdi-folder-open-outline"
                      : "mdi-folder-outline"
                    : ctx.iconFor(row.node.path)}
                </VIcon>
                <span class="pnl-ellipsis">{row.node.name}</span>
              </div>
            );
          })}
          {ctx.canUpload.value && dropDir.value !== null && (
            <div class="pnl-drop-hint">Drop into {dropDir.value || "/ (root)"}</div>
          )}
        </div>
        <input ref={fileInput} type="file" multiple style={{ display: "none" }} onChange={onPick} />

        {/* Upload dialog: pick the target folder for the selected files. */}
        <VDialog modelValue={uploadDlg.value} {...{ "onUpdate:modelValue": (v: boolean) => (uploadDlg.value = v) }} maxWidth="480">
          <VCard>
            <VCardTitle>Upload {pending.value.length} file{pending.value.length === 1 ? "" : "s"}</VCardTitle>
            <VCardText>
              <div class="text-caption mb-1">Target folder</div>
              <div class="dirtree-box">
                <DirTree root={ctx.tree.value} selected={uploadDir.value} onSelect={(p: string) => (uploadDir.value = p)} />
              </div>
            </VCardText>
            <VCardActions>
              <VSpacer />
              <VBtn onClick={() => (uploadDlg.value = false)}>Cancel</VBtn>
              <VBtn color="primary" onClick={confirmUpload}>Upload</VBtn>
            </VCardActions>
          </VCard>
        </VDialog>
      </div>
    );
  },
});
