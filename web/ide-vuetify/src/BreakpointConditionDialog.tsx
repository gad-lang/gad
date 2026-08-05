// BreakpointConditionDialog — edit a breakpoint's condition (a Gad expression;
// the breakpoint pauses only when it is truthy) and its disabled flag.
import { defineComponent, ref, watch, type PropType } from "vue";
import { VBtn, VCard, VCardActions, VCardText, VCardTitle, VCheckbox, VDialog, VSpacer, VTextField } from "./vuetify";
import type { BpMeta } from "./controller";

export default defineComponent({
  name: "BreakpointConditionDialog",
  props: {
    modelValue: { type: Boolean, required: true },
    line: { type: Number, default: 0 },
    initial: { type: Object as PropType<BpMeta>, default: () => ({}) },
  },
  emits: {
    "update:modelValue": (_v: boolean) => true,
    save: (_m: BpMeta) => true,
  },
  setup(props, { emit }) {
    const condition = ref(props.initial.condition ?? "");
    const disabled = ref(!!props.initial.disabled);
    watch(
      () => props.modelValue,
      (open) => {
        if (open) {
          condition.value = props.initial.condition ?? "";
          disabled.value = !!props.initial.disabled;
        }
      },
    );
    function save() {
      emit("save", { condition: condition.value.trim() || undefined, disabled: disabled.value });
      emit("update:modelValue", false);
    }
    return () => (
      <VDialog
        modelValue={props.modelValue}
        {...{ "onUpdate:modelValue": (v: boolean) => emit("update:modelValue", v) }}
        maxWidth="440"
      >
        <VCard>
          <VCardTitle>Breakpoint · line {props.line}</VCardTitle>
          <VCardText>
            <VTextField
              modelValue={condition.value}
              {...{ "onUpdate:modelValue": (v: string) => (condition.value = v) }}
              label="Condition (Gad expression)"
              hint="Pause only when this is truthy, e.g. i > 10"
              persistentHint
              density="compact"
              variant="outlined"
              autofocus
              onKeyup={(e: KeyboardEvent) => e.key === "Enter" && save()}
            />
            <VCheckbox
              modelValue={disabled.value}
              {...{ "onUpdate:modelValue": (v: boolean | null) => (disabled.value = !!v) }}
              label="Disabled"
              density="compact"
              hideDetails
              class="mt-2"
            />
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
