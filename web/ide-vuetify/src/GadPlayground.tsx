// GadPlayground — a two-pane playground (source | output): format, format &
// apply, or run the source, executed through the injected GadRunner. A dialect
// selector switches the example between plain Gad, a Gad template (.gadt) and
// Gadx (.gadx); each keeps its own buffer. The Vuetify/TSX counterpart of the
// React playground.
import { computed, defineComponent, reactive, ref, type PropType } from "vue";
import GadEditor from "./GadEditor";
import DocPanel from "./DocPanel";
import { VBtn, VBtnToggle, VSelect } from "./vuetify";
import type { EditorLanguage } from "./codemirror";
import type { FormatResult, GadRunner, RunResult } from "./types";

type Dialect = "gad" | "gadt" | "gadx";

const SAMPLES: Record<Dialect, string> = {
  gad: `// edit me — errors are underlined as you type
param *args

greet := func(name; greeting="Hello") {
  return greeting + ", " + name
}

squares := [n*n for n in [1,2,3,4] if n>1]
println(greet("Gad"), squares)
return squares
`,
  gadt: `{% var (name = "Gad", items = [1, 2, 3]) %}
<h1>Hello, {%= name %}!</h1>
<ul>
{% for i in items %}  <li>item {%= i %}</li>
{% end %}</ul>
`,
  gadx: `@main
    h1 Hello Gadx
    ul
        @for i in [1, 2, 3]
            li item {= i }
`,
};

// Editor language + backend sourceType per dialect.
const LANG: Record<Dialect, EditorLanguage> = { gad: "gad", gadt: "gadt", gadx: "gadx" };
const SOURCE_TYPE: Record<Dialect, string> = { gad: "", gadt: "gadTemplate", gadx: "gadx" };

export default defineComponent({
  name: "GadPlayground",
  props: {
    runner: { type: Object as PropType<GadRunner>, required: true },
    dark: { type: Boolean, default: false },
  },
  setup(props) {
    const dialect = ref<Dialect>("gad");
    // Each dialect keeps its own buffer so switching examples preserves edits.
    const buffers = reactive<Record<Dialect, string>>({ ...SAMPLES });
    const source = computed<string>({
      get: () => buffers[dialect.value],
      set: (v) => (buffers[dialect.value] = v),
    });
    const sourceType = computed(() => SOURCE_TYPE[dialect.value]);

    const busy = ref(false);
    // GADX only: encode the returned tag as JSON/YAML instead of rendering HTML.
    const tagEncode = ref<"" | "json" | "yaml">("");
    // The Doc panel on the right is hidden by default.
    const showDoc = ref(false);
    const left = ref<{ kind: "format"; fmt: FormatResult } | { kind: "run"; run: RunResult } | null>(null);
    const diagnose = props.runner.diagnose
      ? (src: string) => props.runner.diagnose!(src, sourceType.value)
      : undefined;

    async function doFormat(apply: boolean) {
      busy.value = true;
      try {
        const fmt = await props.runner.format(source.value, sourceType.value);
        left.value = { kind: "format", fmt };
        if (apply && fmt.ok) source.value = fmt.source;
      } finally {
        busy.value = false;
      }
    }
    async function doRun() {
      busy.value = true;
      try {
        const te = dialect.value === "gadx" ? tagEncode.value : "";
        left.value = { kind: "run", run: await props.runner.run(source.value, sourceType.value, te) };
      } finally {
        busy.value = false;
      }
    }

    return () => (
      <div class="gp-split">
        <section class="gp-pane">
          <div class="gp-pane-head">
            <VBtnToggle
              modelValue={dialect.value}
              {...{ "onUpdate:modelValue": (v: unknown) => { if (v) dialect.value = v as Dialect; } }}
              density="compact"
              variant="outlined"
              mandatory
            >
              <VBtn size="small" value="gad">GAD</VBtn>
              <VBtn size="small" value="gadt">GAD Template</VBtn>
              <VBtn size="small" value="gadx">GADx</VBtn>
            </VBtnToggle>
            <span class="gp-actions">
              {dialect.value === "gadx" && (
                <VSelect
                  modelValue={tagEncode.value}
                  {...{ "onUpdate:modelValue": (v: unknown) => (tagEncode.value = (v as "" | "json" | "yaml") ?? "") }}
                  items={[{ title: "Render", value: "" }, { title: "JSON", value: "json" }, { title: "YAML", value: "yaml" }]}
                  label="Tag"
                  density="compact"
                  variant="outlined"
                  hideDetails
                  style={{ maxWidth: "120px" }}
                />
              )}
              <VBtn size="small" variant="tonal" loading={busy.value} onClick={() => doFormat(false)}>Format</VBtn>
              <VBtn size="small" variant="tonal" loading={busy.value} onClick={() => doFormat(true)}>Format &amp; apply</VBtn>
              <VBtn size="small" color="primary" prependIcon="mdi-play" loading={busy.value} onClick={doRun}>Run</VBtn>
              {props.runner.doc && (
                <VBtn
                  size="small"
                  variant={showDoc.value ? "flat" : "tonal"}
                  color={showDoc.value ? "primary" : undefined}
                  onClick={() => (showDoc.value = !showDoc.value)}
                  title="Toggle the documentation panel"
                >
                  Doc
                </VBtn>
              )}
            </span>
          </div>
          <div class="gp-editor">
            <GadEditor
              key={dialect.value}
              modelValue={source.value}
              {...{ "onUpdate:modelValue": (v: string) => (source.value = v) }}
              language={LANG[dialect.value]}
              dark={props.dark}
              diagnose={diagnose}
            />
          </div>
        </section>
        <section class="gp-pane">
          <div class="gp-pane-head">Output</div>
          <div class="gp-pane-body">
            {!left.value && <p class="text-medium-emphasis">Format or run the source on the left.</p>}
            {left.value?.kind === "format" && <FormatView fmt={left.value.fmt} />}
            {left.value?.kind === "run" && <RunView run={left.value.run} />}
          </div>
        </section>
        {props.runner.doc && showDoc.value && (
          <section class="gp-pane gp-pane--doc">
            <DocPanel doc={props.runner.doc!} source={() => source.value} sourceType={sourceType.value}>
              <VBtn size="small" variant="text" title="Hide the doc panel" onClick={() => (showDoc.value = false)}>✕</VBtn>
            </DocPanel>
          </section>
        )}
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
        {props.run.ok && props.run.result && <div class="pnl-return">↩ {props.run.result}</div>}
        {(props.run.diagnostics ?? []).map((d, i) => (
          <div class="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
        ))}
      </div>
    );
  },
});
