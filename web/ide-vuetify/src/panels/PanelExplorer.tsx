// Explorer dockview panel: the workspace file tree with create/upload/delete/
// reset. When the host provides an onUpload handler, files (and directories) can
// be uploaded via a dialog (with a target-folder picker) or dragged and dropped
// onto a folder row (or the panel background = root). Without onUpload, the
// upload button, URL import and drag-drop are all omitted.
import { defineComponent, inject, ref } from "vue";
import { VBtn, VIcon } from "../vuetify";
import { IdeControllerKey } from "../controller";
import UploadReviewDialog from "../UploadReviewDialog";
import { rawFromDataTransfer, rawFromInput, type RawFile } from "../upload";
import { gadFileIconFor } from "../gadFileIcons";
import type { UploadedFile } from "../api";

export default defineComponent({
  name: "PanelExplorer",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const fileInput = ref<HTMLInputElement>();
    // The path of the directory row currently under a drag (null = none).
    const dropDir = ref<string | null>(null);
    // Review dialog: the picked/dropped raw files awaiting confirmation. onUpload
    // runs only after the user confirms (target dir / rename / extract / replace).
    const reviewOpen = ref(false);
    const rawFiles = ref<RawFile[]>([]);
    const reviewDir = ref("");

    const parentDir = (path: string) => {
      const s = path.lastIndexOf("/");
      return s === -1 ? "" : path.slice(0, s);
    };

    function openReview(raw: RawFile[], targetDir: string) {
      if (!raw.length) return;
      rawFiles.value = raw;
      reviewDir.value = targetDir;
      reviewOpen.value = true;
    }
    async function onDrop(e: DragEvent, targetDir: string) {
      dropDir.value = null;
      if (!ctx.canUpload.value || !e.dataTransfer) return;
      openReview(await rawFromDataTransfer(e.dataTransfer), targetDir);
    }
    async function onPick(e: Event) {
      const input = e.target as HTMLInputElement;
      const raw = rawFromInput(input.files);
      input.value = "";
      openReview(raw, "");
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
                {!row.node.dir && gadFileIconFor(row.node.path) ? (
                  <img
                    src={gadFileIconFor(row.node.path)}
                    alt=""
                    width="16"
                    height="16"
                    class="mr-1"
                    style={{ imageRendering: "pixelated", verticalAlign: "-3px" }}
                  />
                ) : (
                  <VIcon size="16" class="mr-1">
                    {row.node.dir
                      ? ctx.isExpanded(row.node.path)
                        ? "mdi-folder-open-outline"
                        : "mdi-folder-outline"
                      : ctx.iconFor(row.node.path)}
                  </VIcon>
                )}
                <span class="pnl-ellipsis">{row.node.name}</span>
              </div>
            );
          })}
          {ctx.canUpload.value && dropDir.value !== null && (
            <div class="pnl-drop-hint">Drop into {dropDir.value || "/ (root)"}</div>
          )}
        </div>
        <input ref={fileInput} type="file" multiple style={{ display: "none" }} onChange={onPick} />

        <UploadReviewDialog
          modelValue={reviewOpen.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (reviewOpen.value = v) }}
          raw={rawFiles.value}
          initialDir={reviewDir.value}
          onConfirm={(files: UploadedFile[], targetDir: string) => ctx.upload(files, targetDir)}
        />
      </div>
    );
  },
});
