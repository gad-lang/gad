// SettingsDialog — mirrors `gad ide`'s Settings: Panels (show/hide dock panels),
// Formatter, Transpile and Template. It edits a draft of the config document and
// emits `save` with the next config; panel visibility is toggled live via
// callbacks (it is layout, not config).
import { defineComponent, reactive, ref, type PropType } from "vue";
import {
  VBtn, VCard, VCardActions, VCardText, VCardTitle, VCheckbox, VDialog,
  VSpacer, VTab, VTabs, VTextField, VWindow, VWindowItem,
} from "./vuetify";

// Opt-in force flags: checked forces that construct onto separate lines. Without
// them a construct wraps only when it overflows the column budget.
const FORCE_FLAGS: [string, string][] = [
  ["array-item-in-new-line", "Each array item on its own line"],
  ["dict-item-in-new-line", "Each dict item on its own line"],
  ["call-params-in-new-line", "Each call argument on its own line"],
];

export interface PanelToggle {
  id: string;
  label: string;
  visible: boolean;
}

export default defineComponent({
  name: "SettingsDialog",
  props: {
    modelValue: { type: Boolean, required: true },
    config: { type: Object as PropType<Record<string, unknown>>, required: true },
    panels: { type: Array as PropType<PanelToggle[]>, default: () => [] },
    onTogglePanel: { type: Function as PropType<(id: string, visible: boolean) => void>, required: true },
  },
  emits: {
    "update:modelValue": (_v: boolean) => true,
    save: (_next: Record<string, unknown>) => true,
  },
  setup(props, { emit }) {
    const tab = ref("panels");
    const rec = (k: string): Record<string, unknown> => (props.config[k] as Record<string, unknown>) || {};

    const fmt = rec("fmt");
    const transpile = rec("transpile");
    const template = rec("template");

    // Formatter checkboxes are opt-in: checked == force that construct on new lines.
    const force = reactive<Record<string, boolean>>(
      Object.fromEntries(FORCE_FLAGS.map(([k]) => [k, fmt[k] === true])),
    );
    const forceAll = ref(fmt.format === true);
    const maxColumns = ref(String(fmt["max-columns"] ?? ""));
    const backup = ref(fmt.backup === true);
    const writeFunc = ref(String(transpile.writeFunc ?? ""));
    const rawStart = ref(String(transpile.rawStrFuncStart ?? ""));
    const rawEnd = ref(String(transpile.rawStrFuncEnd ?? ""));
    const startDelim = ref(String(template.start_delimiter ?? ""));
    const endDelim = ref(String(template.end_delimiter ?? ""));

    function save() {
      const fmtObj: Record<string, unknown> = { ...fmt };
      for (const [k] of FORCE_FLAGS) {
        if (force[k]) fmtObj[k] = true;
        else delete fmtObj[k];
      }
      if (forceAll.value) fmtObj.format = true;
      else delete fmtObj.format;
      const cols = parseInt(maxColumns.value, 10);
      if (Number.isFinite(cols) && cols > 0) fmtObj["max-columns"] = cols;
      else delete fmtObj["max-columns"];
      if (backup.value) fmtObj.backup = true;
      else delete fmtObj.backup;

      const trObj: Record<string, unknown> = { ...transpile };
      const setOrDel = (o: Record<string, unknown>, k: string, v: string) => {
        if (v.trim() === "") delete o[k];
        else o[k] = v;
      };
      setOrDel(trObj, "writeFunc", writeFunc.value);
      setOrDel(trObj, "rawStrFuncStart", rawStart.value);
      setOrDel(trObj, "rawStrFuncEnd", rawEnd.value);

      const tplObj: Record<string, unknown> = { ...template };
      if (startDelim.value === "") delete tplObj.start_delimiter;
      else tplObj.start_delimiter = startDelim.value;
      if (endDelim.value === "") delete tplObj.end_delimiter;
      else tplObj.end_delimiter = endDelim.value;

      const next: Record<string, unknown> = { ...props.config, fmt: fmtObj };
      if (Object.keys(trObj).length > 0) next.transpile = trObj;
      else delete next.transpile;
      if (Object.keys(tplObj).length > 0) next.template = tplObj;
      else delete next.template;

      emit("save", next);
      emit("update:modelValue", false);
    }

    const bind = (r: { value: string }) => ({
      modelValue: r.value,
      "onUpdate:modelValue": (v: string) => (r.value = v),
    });

    return () => (
      <VDialog
        modelValue={props.modelValue}
        {...{ "onUpdate:modelValue": (v: boolean) => emit("update:modelValue", v) }}
        maxWidth="560"
      >
        <VCard>
          <VCardTitle>Settings</VCardTitle>
          <VTabs modelValue={tab.value} {...{ "onUpdate:modelValue": (v: unknown) => (tab.value = v as string) }}>
            <VTab value="panels">Panels</VTab>
            <VTab value="formatter">Formatter</VTab>
            <VTab value="transpile">Transpile</VTab>
            <VTab value="template">Template</VTab>
          </VTabs>
          <VCardText>
            <VWindow modelValue={tab.value} {...{ "onUpdate:modelValue": (v: unknown) => (tab.value = v as string) }}>
              <VWindowItem value="panels">
                {props.panels.map((p) => (
                  <VCheckbox
                    key={p.id}
                    modelValue={p.visible}
                    {...{ "onUpdate:modelValue": (v: boolean | null) => props.onTogglePanel(p.id, !!v) }}
                    label={p.label}
                    density="compact"
                    hideDetails
                  />
                ))}
              </VWindowItem>

              <VWindowItem value="formatter">
                <VCheckbox
                  modelValue={forceAll.value}
                  {...{ "onUpdate:modelValue": (v: boolean | null) => (forceAll.value = !!v) }}
                  label="Force the full multi-line layout"
                  density="compact"
                  hideDetails
                />
                <VTextField
                  modelValue={maxColumns.value}
                  {...{ "onUpdate:modelValue": (v: string) => (maxColumns.value = v) }}
                  label="Max columns (blank uses the default)"
                  type="number"
                  density="compact"
                  hideDetails
                />
                {FORCE_FLAGS.map(([k, label]) => (
                  <VCheckbox
                    key={k}
                    modelValue={force[k]}
                    {...{ "onUpdate:modelValue": (v: boolean | null) => (force[k] = !!v) }}
                    label={label}
                    density="compact"
                    hideDetails
                  />
                ))}
                <VCheckbox
                  modelValue={backup.value}
                  {...{ "onUpdate:modelValue": (v: boolean | null) => (backup.value = !!v) }}
                  label="Keep .backup on format"
                  density="compact"
                  hideDetails
                />
              </VWindowItem>

              <VWindowItem value="transpile">
                <VTextField {...bind(writeFunc)} label="writeFunc" density="compact" variant="outlined" hideDetails class="mb-2" />
                <VTextField {...bind(rawStart)} label="rawStrFuncStart" density="compact" variant="outlined" hideDetails class="mb-2" />
                <VTextField {...bind(rawEnd)} label="rawStrFuncEnd" density="compact" variant="outlined" hideDetails />
              </VWindowItem>

              <VWindowItem value="template">
                <VTextField {...bind(startDelim)} label="start_delimiter" density="compact" variant="outlined" hideDetails class="mb-2" />
                <VTextField {...bind(endDelim)} label="end_delimiter" density="compact" variant="outlined" hideDetails />
              </VWindowItem>
            </VWindow>
          </VCardText>
          <VCardActions>
            <VSpacer />
            <VBtn onClick={() => emit("update:modelValue", false)}>Cancel</VBtn>
            <VBtn color="primary" onClick={save}>Save</VBtn>
          </VCardActions>
        </VCard>
      </VDialog>
    );
  },
});
