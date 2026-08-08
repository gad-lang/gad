// GadNotebook — a column of independently runnable Gad cells, each with its own
// editor, dialect selector (GAD / GAD Template / GADx) and stdout/stderr/return
// output, executed through the injected GadRunner. The Vuetify/TSX counterpart of
// the React notebook.
import { defineComponent, ref, type PropType } from "vue";
import GadEditor from "./GadEditor";
import { VBtn, VBtnToggle, VSelect } from "./vuetify";
import type { EditorLanguage } from "./codemirror";
import type { GadRunner, RunResult } from "./types";

type Dialect = "gad" | "gadt" | "gadx";
const LANG: Record<Dialect, EditorLanguage> = { gad: "gad", gadt: "gadt", gadx: "gadx" };
const SOURCE_TYPE: Record<Dialect, string> = { gad: "", gadt: "gadTemplate", gadx: "gadx" };

interface Cell {
  id: number;
  source: string;
  dialect: Dialect;
  tagEncode: "" | "json" | "yaml";
  result: RunResult | null;
  running: boolean;
}

let nextId = 1;
const newCell = (source = "", dialect: Dialect = "gad"): Cell => ({
  id: nextId++,
  source,
  dialect,
  tagEncode: "",
  result: null,
  running: false,
});

// One sample cell per source type (GAD / GAD Template / GADx), so the notebook
// opens with a runnable example of each dialect.
const SAMPLES: { dialect: Dialect; source: string }[] = [
  { dialect: "gad", source: `// GAD — plain script\nsquares := [n * n for n in [1, 2, 3, 4, 5]]\nprintln(squares)\nreturn squares` },
  { dialect: "gadt", source: `{% /* GAD Template: literal text plus code islands and value output */ %}\n{% var (name = "Gad", items = [1, 2, 3]) %}\n<h1>Hello, {%= name %}!</h1>\n<ul>\n{% for i in items %}  <li>item {%= i %}</li>\n{% end %}</ul>\n` },
  { dialect: "gadx", source: `//- GADx — indentation-based HTML template\n@main\n    h1 Hello Gadx\n    ul\n        @for i in [1, 2, 3]\n            li item {= i }\n` },
];

export default defineComponent({
  name: "GadNotebook",
  props: {
    runner: { type: Object as PropType<GadRunner>, required: true },
    dark: { type: Boolean, default: false },
  },
  setup(props) {
    const cells = ref<Cell[]>(SAMPLES.map((s) => newCell(s.source, s.dialect)));

    async function runCell(cell: Cell) {
      cell.running = true;
      try {
        const te = cell.dialect === "gadx" ? cell.tagEncode : "";
        cell.result = await props.runner.run(cell.source, SOURCE_TYPE[cell.dialect], te);
      } catch (e) {
        cell.result = { ok: false, stdout: "", stderr: String(e), result: "", diagnostics: [] };
      } finally {
        cell.running = false;
      }
    }
    const addCell = () => cells.value.push(newCell());
    const removeCell = (id: number) => {
      if (cells.value.length > 1) cells.value = cells.value.filter((c) => c.id !== id);
    };
    const diagnoseFor = (cell: Cell) =>
      props.runner.diagnose ? (src: string) => props.runner.diagnose!(src, SOURCE_TYPE[cell.dialect]) : undefined;

    return () => (
      <div class="gnb">
        <p class="text-medium-emphasis">Each cell runs independently.</p>
        {cells.value.map((cell) => (
          <div class="gnb-cell" key={cell.id}>
            <div class="gnb-editor">
              <GadEditor
                key={cell.id + ":" + cell.dialect}
                modelValue={cell.source}
                {...{ "onUpdate:modelValue": (v: string) => (cell.source = v) }}
                language={LANG[cell.dialect]}
                dark={props.dark}
                diagnose={diagnoseFor(cell)}
              />
            </div>
            <div class="gnb-bar">
              <VBtnToggle
                modelValue={cell.dialect}
                {...{ "onUpdate:modelValue": (v: unknown) => { if (v) cell.dialect = v as Dialect; } }}
                density="compact"
                variant="outlined"
                mandatory
              >
                <VBtn size="small" value="gad">GAD</VBtn>
                <VBtn size="small" value="gadt">GAD Template</VBtn>
                <VBtn size="small" value="gadx">GADx</VBtn>
              </VBtnToggle>
              {cell.dialect === "gadx" && (
                <VSelect
                  modelValue={cell.tagEncode}
                  {...{ "onUpdate:modelValue": (v: unknown) => (cell.tagEncode = (v as "" | "json" | "yaml") ?? "") }}
                  items={[{ title: "Render", value: "" }, { title: "JSON", value: "json" }, { title: "YAML", value: "yaml" }]}
                  label="Tag"
                  density="compact"
                  variant="outlined"
                  hideDetails
                  style={{ maxWidth: "120px" }}
                />
              )}
              <VBtn size="small" color="primary" prependIcon="mdi-play" loading={cell.running} onClick={() => runCell(cell)}>Run</VBtn>
              <VBtn size="small" variant="text" onClick={() => removeCell(cell.id)}>Remove</VBtn>
            </div>
            {cell.result && (
              <div class={"gnb-out" + (cell.result.ok ? "" : " gp-error")}>
                {cell.result.stdout && <pre class="gp-out">{cell.result.stdout}</pre>}
                {cell.result.stderr && <pre class="gp-out gad-ide__diag">{cell.result.stderr}</pre>}
                {cell.result.ok && cell.result.result && <div class="pnl-return">↩ {cell.result.result}</div>}
                {(cell.result.diagnostics ?? []).map((d, i) => (
                  <div class="gad-ide__diag" key={i}>{d.line}:{d.column} {d.message}</div>
                ))}
              </div>
            )}
          </div>
        ))}
        <VBtn size="small" variant="tonal" prependIcon="mdi-plus" onClick={addCell}>Add cell</VBtn>
      </div>
    );
  },
});
