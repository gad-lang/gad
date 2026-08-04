<!-- GadEditor — a Vue wrapper around a CodeMirror 6 editor wired for Gad
     (syntax + diagnostics, breakpoint gutter, debug line highlight). It owns the
     EditorView imperatively and reacts to prop changes via compartments. -->
<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { GadEditorView, langOf, type EditorLanguage, type LocalVar } from "./codemirror";
import type { DiagnoseFn } from "@gad-lang/codemirror-gad";

const props = defineProps<{
  /** File path — drives the language when `language` is not given. */
  path?: string;
  modelValue: string;
  language?: EditorLanguage;
  dark?: boolean;
  diagnose?: DiagnoseFn;
  breakpoints?: number[];
  /** Current paused debug line (1-based, 0 to clear) and column. */
  debugLine?: number;
  debugColumn?: number;
  getLocals?: () => Map<string, LocalVar>;
}>();

const emit = defineEmits<{
  (e: "update:modelValue", value: string): void;
  (e: "update:breakpoints", lines: number[]): void;
}>();

const host = ref<HTMLDivElement>();
let editor: GadEditorView | null = null;
// Guard so echoing our own change back into modelValue does not re-set the doc.
let selfEdit = false;

function currentLang(): EditorLanguage {
  return props.language ?? langOf(props.path ?? "");
}

onMounted(() => {
  editor = new GadEditorView({
    parent: host.value!,
    doc: props.modelValue,
    language: currentLang(),
    dark: !!props.dark,
    diagnose: props.diagnose,
    getLocals: props.getLocals,
    onChange: (value) => {
      selfEdit = true;
      emit("update:modelValue", value);
      selfEdit = false;
    },
    onBreakpointsChange: (lines) => emit("update:breakpoints", lines),
  });
  if (props.breakpoints?.length) editor.setBreakpoints(props.breakpoints);
  if (props.debugLine) editor.setDebugLine(props.debugLine, props.debugColumn ?? 1);
});

onBeforeUnmount(() => editor?.destroy());

watch(
  () => props.modelValue,
  (v) => {
    if (editor && !selfEdit && v !== editor.getValue()) editor.setValue(v);
  },
);
watch([() => props.path, () => props.language], () => editor?.setLanguage(currentLang(), props.diagnose));
watch(() => props.dark, (d) => editor?.setDark(!!d));
watch(() => props.breakpoints, (b) => editor?.setBreakpoints(b ?? []));
watch([() => props.debugLine, () => props.debugColumn], ([l, c]) => editor?.setDebugLine(l ?? 0, c ?? 1));
</script>

<template>
  <div ref="host" class="gad-editor" />
</template>

<style scoped>
.gad-editor {
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
.gad-editor :deep(.cm-editor) {
  height: 100%;
}
</style>
