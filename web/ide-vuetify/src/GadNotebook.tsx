// GadNotebook — a column of independently runnable Gad cells, each with its own
// editor and stdout/stderr/return output, executed through the injected
// GadRunner. The Vuetify/TSX counterpart of the React notebook.
import { defineComponent, ref, type PropType } from "vue";
import GadEditor from "./GadEditor";
import { VBtn } from "./vuetify";
import type { GadRunner, RunResult } from "./types";

interface Cell {
  id: number;
  source: string;
  result: RunResult | null;
  running: boolean;
}

let nextId = 1;
const newCell = (source = ""): Cell => ({ id: nextId++, source, result: null, running: false });

const SAMPLE = [
  `total := 0\nfor i in [1, 2, 3, 4] {\n  total += i\n}\nprintln("sum =", total)\nreturn total`,
  `squares := [n * n for n in [1, 2, 3, 4, 5]]\nprintln(squares)\nreturn squares`,
];

export default defineComponent({
  name: "GadNotebook",
  props: {
    runner: { type: Object as PropType<GadRunner>, required: true },
    dark: { type: Boolean, default: false },
  },
  setup(props) {
    const cells = ref<Cell[]>(SAMPLE.map((s) => newCell(s)));
    const diagnose = props.runner.diagnose;

    async function runCell(cell: Cell) {
      cell.running = true;
      try {
        cell.result = await props.runner.run(cell.source);
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

    return () => (
      <div class="gnb">
        <p class="text-medium-emphasis">Each cell runs independently.</p>
        {cells.value.map((cell) => (
          <div class="gnb-cell" key={cell.id}>
            <div class="gnb-editor">
              <GadEditor
                modelValue={cell.source}
                {...{ "onUpdate:modelValue": (v: string) => (cell.source = v) }}
                language="gad"
                dark={props.dark}
                diagnose={diagnose}
              />
            </div>
            <div class="gnb-bar">
              <VBtn size="small" color="primary" prependIcon="mdi-play" loading={cell.running} onClick={() => runCell(cell)}>Run</VBtn>
              <VBtn size="small" variant="text" onClick={() => removeCell(cell.id)}>Remove</VBtn>
            </div>
            {cell.result && (
              <div class={"gnb-out" + (cell.result.ok ? "" : " gp-error")}>
                {cell.result.stdout && <pre class="gp-out">{cell.result.stdout}</pre>}
                {cell.result.stderr && <pre class="gp-out gad-ide__diag">{cell.result.stderr}</pre>}
                {cell.result.ok && cell.result.result && <div class="pnl-return">⇦ {cell.result.result}</div>}
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
