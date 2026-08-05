// GadEditor — a Vue (TSX) wrapper around a CodeMirror 6 editor wired for Gad
// (syntax + diagnostics, breakpoint gutter, debug line highlight). It owns the
// EditorView imperatively and reacts to prop changes via compartments.
import { defineComponent, onBeforeUnmount, onMounted, ref, watch, type PropType } from "vue";
import type { DiagnoseFn } from "@gad-lang/codemirror-gad";
import { GadEditorView, langOf, type EditorLanguage, type LocalVar } from "./codemirror";

export default defineComponent({
  name: "GadEditor",
  props: {
    path: { type: String, default: "" },
    modelValue: { type: String, required: true },
    language: { type: String as PropType<EditorLanguage>, default: undefined },
    dark: { type: Boolean, default: false },
    diagnose: { type: Function as PropType<DiagnoseFn>, default: undefined },
    breakpoints: { type: Array as PropType<number[]>, default: () => [] },
    debugLine: { type: Number, default: 0 },
    debugColumn: { type: Number, default: 1 },
    getLocals: { type: Function as PropType<() => Map<string, LocalVar>>, default: undefined },
    /** Navigation request: scroll to `gotoLine`; bump `gotoSeq` to re-trigger. */
    gotoLine: { type: Number, default: 0 },
    gotoSeq: { type: Number, default: 0 },
    /** Right-click on a breakpoint line: (line) => void (e.g. edit condition). */
    onBreakpointContext: { type: Function as PropType<(line: number) => void>, default: undefined },
  },
  emits: {
    "update:modelValue": (_v: string) => true,
    "update:breakpoints": (_lines: number[]) => true,
    ready: (_view: GadEditorView) => true,
  },
  setup(props, { emit }) {
    const host = ref<HTMLDivElement>();
    let editor: GadEditorView | null = null;
    // Guard so echoing our own change back into modelValue does not re-set the doc.
    let selfEdit = false;

    const currentLang = (): EditorLanguage => props.language ?? langOf(props.path);

    onMounted(() => {
      editor = new GadEditorView({
        parent: host.value!,
        doc: props.modelValue,
        language: currentLang(),
        dark: props.dark,
        diagnose: props.diagnose,
        getLocals: props.getLocals,
        onChange: (value) => {
          selfEdit = true;
          emit("update:modelValue", value);
          selfEdit = false;
        },
        onBreakpointsChange: (lines) => emit("update:breakpoints", lines),
        onBreakpointContext: (line) => props.onBreakpointContext?.(line),
      });
      if (props.breakpoints.length) editor.setBreakpoints(props.breakpoints);
      if (props.debugLine) editor.setDebugLine(props.debugLine, props.debugColumn);
      emit("ready", editor);
    });

    onBeforeUnmount(() => editor?.destroy());

    watch(
      () => props.modelValue,
      (v) => {
        if (editor && !selfEdit && v !== editor.getValue()) editor.setValue(v);
      },
    );
    watch([() => props.path, () => props.language], () => editor?.setLanguage(currentLang(), props.diagnose));
    watch(() => props.dark, (d) => editor?.setDark(d));
    watch(() => props.breakpoints, (b) => editor?.setBreakpoints(b ?? []));
    watch([() => props.debugLine, () => props.debugColumn], ([l, c]) => editor?.setDebugLine(l ?? 0, c ?? 1));
    // Navigate on each new goto request (seq bumps even to the same line).
    watch(() => props.gotoSeq, () => { if (props.gotoLine) editor?.gotoLine(props.gotoLine); });

    return () => <div ref={host} class="gad-editor" />;
  },
});
