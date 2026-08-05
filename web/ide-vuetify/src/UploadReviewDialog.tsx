// UploadReviewDialog — shown after picking or dropping files, before onUpload
// runs. It reviews the files, lets a single file be renamed and (if it is a
// ZIP/TAR/TAR.GZ) extracted, confirms the target directory, and warns when the
// destination already exists (requiring "Replace existing" to proceed).
import { computed, defineComponent, inject, ref, watch, type PropType } from "vue";
import {
  VBtn, VCard, VCardActions, VCardText, VCardTitle, VCheckbox, VDialog, VSpacer, VSwitch, VTextField,
} from "./vuetify";
import DirTree from "./DirTree";
import { IdeControllerKey } from "./controller";
import { readBase64, type RawFile } from "./upload";
import type { UploadedFile } from "./api";

const baseName = (p: string) => p.slice(p.lastIndexOf("/") + 1);
const join = (dir: string, rel: string) => (dir ? dir + "/" + rel : rel);

export default defineComponent({
  name: "UploadReviewDialog",
  props: {
    modelValue: { type: Boolean, required: true },
    raw: { type: Array as PropType<RawFile[]>, default: () => [] },
    initialDir: { type: String, default: "" },
  },
  emits: {
    "update:modelValue": (_v: boolean) => true,
    confirm: (_files: UploadedFile[], _targetDir: string) => true,
  },
  setup(props, { emit }) {
    const ctx = inject(IdeControllerKey)!;
    const targetDir = ref("");
    const name = ref("");
    const extract = ref(true);
    const replace = ref(false);
    const pickDir = ref(false);
    const busy = ref(false);

    const single = computed(() => props.raw.length === 1);
    const archiveOf = computed(() => (single.value ? ctx.archiveKind(name.value) : undefined));

    // Reset the form each time the dialog opens with a new file set.
    watch(
      () => props.modelValue,
      (open) => {
        if (!open) return;
        targetDir.value = props.initialDir;
        name.value = single.value ? baseName(props.raw[0].path) : "";
        extract.value = true;
        replace.value = false;
        pickDir.value = false;
      },
    );

    // Final destination paths (for the existence check). When extracting, the
    // archive expands into a folder the host manages, so per-entry paths are
    // unknown here — collision detection is skipped for that case.
    const finalPaths = computed<string[]>(() => {
      if (single.value) {
        if (archiveOf.value && extract.value) return [];
        return [join(targetDir.value, name.value)];
      }
      return props.raw.map((r) => join(targetDir.value, r.path));
    });
    const collisions = computed(() => finalPaths.value.filter((p) => ctx.pathExists(p)));
    const blocked = computed(() => collisions.value.length > 0 && !replace.value);

    async function confirm() {
      busy.value = true;
      try {
        let files: UploadedFile[];
        if (single.value && archiveOf.value && extract.value) {
          files = [{ path: name.value, content: "", archive: archiveOf.value, bytes: await readBase64(props.raw[0].file) }];
        } else if (single.value) {
          files = [{ path: name.value, content: await props.raw[0].file.text() }];
        } else {
          files = await Promise.all(props.raw.map(async (r) => ({ path: r.path, content: await r.file.text() })));
        }
        emit("confirm", files, targetDir.value);
        emit("update:modelValue", false);
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
          <VCardTitle>
            {single.value ? "Upload file" : `Upload ${props.raw.length} files`}
          </VCardTitle>
          <VCardText>
            {single.value ? (
              <VTextField
                modelValue={name.value}
                {...{ "onUpdate:modelValue": (v: string) => (name.value = v) }}
                label="Destination name"
                density="compact"
                variant="outlined"
                hideDetails
                class="mb-2"
              />
            ) : (
              <div class="text-caption mb-2">
                {props.raw.slice(0, 6).map((r) => <div class="pnl-ellipsis" key={r.path}>{r.path}</div>)}
                {props.raw.length > 6 && <div class="text-medium-emphasis">…and {props.raw.length - 6} more</div>}
              </div>
            )}

            {single.value && archiveOf.value && (
              <VSwitch
                modelValue={extract.value}
                {...{ "onUpdate:modelValue": (v: boolean | null) => (extract.value = !!v) }}
                label={`Extract archive (${archiveOf.value.toUpperCase()})`}
                density="compact"
                hideDetails
                color="primary"
                class="mb-1"
              />
            )}

            {/* Target directory. */}
            <div class="d-flex align-center" style={{ gap: "4px" }}>
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
                <DirTree root={ctx.tree.value} selected={targetDir.value} onSelect={(p: string) => { targetDir.value = p; pickDir.value = false; }} />
              </div>
            )}

            {collisions.value.length > 0 && (
              <VCheckbox
                modelValue={replace.value}
                {...{ "onUpdate:modelValue": (v: boolean | null) => (replace.value = !!v) }}
                label={collisions.value.length === 1
                  ? `"${collisions.value[0]}" already exists — replace it`
                  : `${collisions.value.length} files already exist — replace them`}
                density="compact"
                hideDetails
                color="error"
                class="mt-2"
              />
            )}
          </VCardText>
          <VCardActions>
            <VSpacer />
            <VBtn onClick={() => emit("update:modelValue", false)}>Cancel</VBtn>
            <VBtn color="primary" loading={busy.value} disabled={blocked.value || (single.value && !name.value.trim())} onClick={confirm}>
              Upload
            </VBtn>
          </VCardActions>
        </VCard>
      </VDialog>
    );
  },
});
