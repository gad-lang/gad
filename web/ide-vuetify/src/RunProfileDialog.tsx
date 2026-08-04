// RunProfileDialog — create a named run/debug profile (JetBrains-style): a name,
// the file to execute (prefilled with the open file) and command-line arguments
// (chips). On save it emits the new profile.
import { defineComponent, ref, watch, type PropType } from "vue";
import { VBtn, VCard, VCardActions, VCardText, VCardTitle, VCombobox, VDialog, VSpacer, VTextField } from "./vuetify";
import type { RunProfile } from "./api";

export default defineComponent({
  name: "RunProfileDialog",
  props: {
    modelValue: { type: Boolean, required: true },
    /** Default file path for a new profile (the open file). */
    defaultPath: { type: String, default: "" },
  },
  emits: {
    "update:modelValue": (_v: boolean) => true,
    create: (_p: RunProfile) => true,
  },
  setup(props, { emit }) {
    const name = ref("");
    const path = ref(props.defaultPath);
    const args = ref<string[]>([]);

    // Re-seed when the dialog opens (the open file may have changed).
    watch(
      () => props.modelValue,
      (open) => {
        if (open) {
          path.value = props.defaultPath;
          if (!name.value) name.value = props.defaultPath.split("/").pop() ?? "profile";
        }
      },
    );

    function save() {
      const p: RunProfile = { name: name.value.trim() || (path.value.split("/").pop() ?? "profile"), path: path.value, args: [...args.value] };
      emit("create", p);
      emit("update:modelValue", false);
      name.value = "";
      args.value = [];
    }

    return () => (
      <VDialog
        modelValue={props.modelValue}
        {...{ "onUpdate:modelValue": (v: boolean) => emit("update:modelValue", v) }}
        maxWidth="520"
      >
        <VCard>
          <VCardTitle>New run profile</VCardTitle>
          <VCardText>
            <VTextField
              modelValue={name.value}
              {...{ "onUpdate:modelValue": (v: string) => (name.value = v) }}
              label="Name"
              density="compact"
              variant="outlined"
              class="mb-2"
              hideDetails
            />
            <VTextField
              modelValue={path.value}
              {...{ "onUpdate:modelValue": (v: string) => (path.value = v) }}
              label="File"
              density="compact"
              variant="outlined"
              class="mb-2"
              hideDetails
            />
            <VCombobox
              modelValue={args.value}
              {...{ "onUpdate:modelValue": (v: string[]) => (args.value = v) }}
              label="Arguments"
              hint="Press Enter after each argument (e.g. --count=5, a value, --flag)"
              persistentHint
              multiple
              chips
              closableChips
              density="compact"
              variant="outlined"
            />
          </VCardText>
          <VCardActions>
            <VSpacer />
            <VBtn onClick={() => emit("update:modelValue", false)}>Cancel</VBtn>
            <VBtn color="primary" disabled={!path.value.trim()} onClick={save}>Create</VBtn>
          </VCardActions>
        </VCard>
      </VDialog>
    );
  },
});
