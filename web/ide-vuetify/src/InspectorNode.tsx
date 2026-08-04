// InspectorNode — one lazily-expanding node of the value tree navigator. Shows a
// value's key, type and rendered value; expandable containers fetch their
// children on first expand via the injected `inspect` function and render child
// InspectorNodes recursively, reaching each child by appending its Gad accessor.
import { defineComponent, onMounted, ref, type PropType } from "vue";
import type { InspectEntry, InspectResult } from "./api";

export type InspectFn = (expr: string) => Promise<InspectResult | null>;

const InspectorNode = defineComponent({
  name: "InspectorNode",
  props: {
    inspect: { type: Function as PropType<InspectFn>, required: true },
    label: { type: String, required: true },
    expr: { type: String, required: true },
    type: { type: String, default: "" },
    value: { type: String, default: "" },
    expandable: { type: Boolean, default: false },
    depth: { type: Number, default: 0 },
    root: { type: Boolean, default: false },
  },
  setup(props) {
    const open = ref(false);
    const loading = ref(false);
    const entries = ref<InspectEntry[] | null>(null);
    const selfType = ref(props.type);
    const selfValue = ref(props.value);
    const selfExpandable = ref(props.expandable);

    async function load() {
      loading.value = true;
      const r = await props.inspect(props.expr);
      loading.value = false;
      if (!r) {
        entries.value = [];
        return;
      }
      selfType.value = r.type;
      selfValue.value = r.value;
      selfExpandable.value = r.expandable;
      entries.value = r.entries ?? [];
    }
    async function toggle() {
      if (!selfExpandable.value) return;
      if (entries.value === null) await load();
      open.value = !open.value;
    }
    onMounted(() => {
      if (props.root) void load();
    });

    return () => (
      <div class="ins-node">
        <div class="ins-row" style={{ paddingLeft: props.depth * 14 + "px" }} onClick={toggle}>
          <span class="ins-twist">{selfExpandable.value ? (open.value ? "▾" : "▸") : "·"}</span>
          <span class="ins-key">{props.label}</span>
          <span class="ins-type">{selfType.value}</span>
          <span class="ins-val">{selfValue.value}</span>
          {loading.value && <span class="ins-load">…</span>}
        </div>
        {open.value && entries.value
          ? entries.value.map((e) => (
              <InspectorNode
                key={e.accessor}
                inspect={props.inspect}
                label={e.key}
                expr={props.expr + e.accessor}
                type={e.type}
                value={e.value}
                expandable={e.expandable}
                depth={props.depth + 1}
              />
            ))
          : null}
      </div>
    );
  },
});

export default InspectorNode;
