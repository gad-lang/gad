// GadPlayground — a two-pane playground (output | source): format, format &
// apply, or run the Gad source, executed through the injected GadRunner. The
// Vuetify/TSX counterpart of the React playground.
import { defineComponent, ref, type PropType } from "vue";
import GadEditor from "./GadEditor";
import { VBtn } from "./vuetify";
import type { FormatResult, GadRunner, RunResult } from "./types";

const SAMPLE = `// edit me — errors are underlined as you type
param *args

greet := func(name; greeting="Hello") {
  return greeting + ", " + name
}

squares := [n*n for n in [1,2,3,4] if n>1]
println(greet("Gad"), squares)
return squares
`;

export default defineComponent({
  name: "GadPlayground",
  props: {
    runner: { type: Object as PropType<GadRunner>, required: true },
    dark: { type: Boolean, default: false },
    initialDoc: { type: String, default: SAMPLE },
  },
  setup(props) {
    const source = ref(props.initialDoc);
    const busy = ref(false);
    const left = ref<{ kind: "format"; fmt: FormatResult } | { kind: "run"; run: RunResult } | null>(null);
    const diagnose = props.runner.diagnose;

    async function doFormat(apply: boolean) {
      busy.value = true;
      try {
        const fmt = await props.runner.format(source.value);
        left.value = { kind: "format", fmt };
        if (apply && fmt.ok) source.value = fmt.source;
      } finally {
        busy.value = false;
      }
    }
    async function doRun() {
      busy.value = true;
      try {
        left.value = { kind: "run", run: await props.runner.run(source.value) };
      } finally {
        busy.value = false;
      }
    }

    return () => (
      <div class="gp-split">
        <section class="gp-pane">
          <div class="gp-pane-head">Output</div>
          <div class="gp-pane-body">
            {!left.value && <p class="text-medium-emphasis">Format or run the source on the right.</p>}
            {left.value?.kind === "format" && <FormatView fmt={left.value.fmt} />}
            {left.value?.kind === "run" && <RunView run={left.value.run} />}
          </div>
        </section>
        <section class="gp-pane">
          <div class="gp-pane-head">
            <span>Source</span>
            <span class="gp-actions">
              <VBtn size="small" variant="tonal" loading={busy.value} onClick={() => doFormat(false)}>Format</VBtn>
              <VBtn size="small" variant="tonal" loading={busy.value} onClick={() => doFormat(true)}>Format &amp; apply</VBtn>
              <VBtn size="small" color="primary" prependIcon="mdi-play" loading={busy.value} onClick={doRun}>Run</VBtn>
            </span>
          </div>
          <div class="gp-editor">
            <GadEditor
              modelValue={source.value}
              {...{ "onUpdate:modelValue": (v: string) => (source.value = v) }}
              language="gad"
              dark={props.dark}
              diagnose={diagnose}
            />
          </div>
        </section>
      </div>
    );
  },
});

const FormatView = defineComponent({
  props: { fmt: { type: Object as PropType<FormatResult>, required: true } },
  setup(props) {
    return () =>
      props.fmt.ok ? (
        <pre class="gp-out">{props.fmt.source}</pre>
      ) : (
        <div>
          {(props.fmt.diagnostics ?? []).map((d, i) => (
            <div class="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
          ))}
        </div>
      );
  },
});

const RunView = defineComponent({
  props: { run: { type: Object as PropType<RunResult>, required: true } },
  setup(props) {
    return () => (
      <div class={props.run.ok ? "" : "gp-error"}>
        {props.run.stdout && <pre class="gp-out">{props.run.stdout}</pre>}
        {props.run.stderr && <pre class="gp-out gad-ide__diag">{props.run.stderr}</pre>}
        {props.run.ok && props.run.result && <div class="pnl-return">⇦ {props.run.result}</div>}
        {(props.run.diagnostics ?? []).map((d, i) => (
          <div class="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
        ))}
      </div>
    );
  },
});
