// UrlImportDialog — import a file from a URL. When the URL points to a
// ZIP/TAR/TAR.GZ, an "Extract archive" switch appears; if on, the download is
// passed to onUpload as an archive for the host to extract.
import { computed, defineComponent, inject, ref } from "vue";
import { VBtn, VCard, VCardActions, VCardText, VCardTitle, VDialog, VProgressLinear, VSpacer, VSwitch, VTextField } from "./vuetify";
import DirTree from "./DirTree";
import { IdeControllerKey } from "./controller";

export default defineComponent({
  name: "UrlImportDialog",
  props: {
    modelValue: { type: Boolean, required: true },
    /** Download progress percent (0–100), or -1 for indeterminate. */
    progress: { type: Number, default: 0 },
  },
  emits: {
    "update:modelValue": (_v: boolean) => true,
    import: (_url: string, _extract: boolean, _targetDir: string) => true,
  },
  setup(props, { emit }) {
    const ctx = inject(IdeControllerKey)!;
    const url = ref("");
    const extract = ref(true);
    const busy = ref(false);
    const error = ref("");
    const targetDir = ref("");
    const pickDir = ref(false);
    const isArchive = computed(() => /\.(zip|tar|tar\.gz|tgz)(\?|#|$)/i.test(url.value));

    async function doImport() {
      if (!url.value.trim()) return;
      busy.value = true;
      error.value = "";
      try {
        await emit("import", url.value.trim(), isArchive.value && extract.value, targetDir.value);
        emit("update:modelValue", false);
        url.value = "";
      } catch (e) {
        error.value = String(e);
      } finally {
        busy.value = false;
      }
    }

    return () => (
      <VDialog
        modelValue={props.modelValue}
        {...{ "onUpdate:modelValue": (v: boolean) => emit("update:modelValue", v) }}
        maxWidth="520"
      >
        <VCard>
          <VCardTitle>Import from URL</VCardTitle>
          <VCardText>
            <VTextField
              modelValue={url.value}
              {...{ "onUpdate:modelValue": (v: string) => (url.value = v) }}
              label="URL"
              placeholder="https://example.com/file.gad"
              density="compact"
              variant="outlined"
              autofocus
              hideDetails
              onKeyup={(e: KeyboardEvent) => e.key === "Enter" && doImport()}
            />
            {isArchive.value && (
              <VSwitch
                modelValue={extract.value}
                {...{ "onUpdate:modelValue": (v: boolean | null) => (extract.value = !!v) }}
                label="Extract archive (ZIP / TAR / TAR.GZ)"
                density="compact"
                hideDetails
                class="mt-2"
                color="primary"
              />
            )}
            {/* Target directory picker. */}
            <div class="d-flex align-center mt-2" style={{ gap: "4px" }}>
              <VTextField
                modelValue={targetDir.value || "/ (root)"}
                label="Target folder"
                density="compact"
                variant="outlined"
                readonly
                hideDetails
                onClick={() => (pickDir.value = !pickDir.value)}
              />
              <VBtn size="small" variant="text" icon="mdi-folder-search-outline" title="Choose folder"
                onClick={() => (pickDir.value = !pickDir.value)} />
            </div>
            {pickDir.value && (
              <div class="dirtree-box mt-1">
                <DirTree
                  root={ctx.tree.value}
                  selected={targetDir.value}
                  onSelect={(p: string) => { targetDir.value = p; pickDir.value = false; }}
                />
              </div>
            )}
            {busy.value && (
              <div class="mt-3">
                <VProgressLinear
                  modelValue={props.progress < 0 ? undefined : props.progress}
                  indeterminate={props.progress < 0}
                  color="primary"
                  height="6"
                  rounded
                />
                {props.progress >= 0 && <div class="text-caption text-center mt-1">{props.progress}%</div>}
              </div>
            )}
            {error.value && <div class="pnl-diag mt-2">{error.value}</div>}
          </VCardText>
          <VCardActions>
            <VSpacer />
            <VBtn onClick={() => emit("update:modelValue", false)}>Cancel</VBtn>
            <VBtn color="primary" loading={busy.value} disabled={!url.value.trim()} onClick={doImport}>Import</VBtn>
          </VCardActions>
        </VCard>
      </VDialog>
    );
  },
});
