// DirTree — a compact, directory-only tree for picking an upload target folder.
// It renders the workspace root as "/" plus every directory node; clicking a row
// selects it (selected path "" = the workspace root).
import { defineComponent, type PropType } from "vue";
import { VIcon } from "./vuetify";
import type { TreeNode } from "./api";

const DirRow = defineComponent({
  name: "DirRow",
  props: {
    node: { type: Object as PropType<TreeNode>, required: true },
    selected: { type: String, required: true },
    depth: { type: Number, default: 0 },
    onSelect: { type: Function as PropType<(path: string) => void>, required: true },
  },
  setup(props) {
    return () => {
      const dirs = (props.node.children ?? []).filter((c) => c.dir);
      return (
        <>
          <div
            class={["dirtree-row", { "dirtree-row--active": props.selected === props.node.path }]}
            style={{ paddingLeft: 6 + props.depth * 14 + "px" }}
            onClick={() => props.onSelect(props.node.path)}
          >
            <VIcon size="14" class="mr-1">mdi-folder-outline</VIcon>
            <span class="pnl-ellipsis">{props.node.name}</span>
          </div>
          {dirs.map((c) => (
            <DirRow key={c.path} node={c} selected={props.selected} depth={props.depth + 1} onSelect={props.onSelect} />
          ))}
        </>
      );
    };
  },
});

export default defineComponent({
  name: "DirTree",
  props: {
    root: { type: Object as PropType<TreeNode | null>, default: null },
    selected: { type: String, default: "" },
    onSelect: { type: Function as PropType<(path: string) => void>, required: true },
  },
  setup(props) {
    return () => (
      <div class="dirtree">
        <div
          class={["dirtree-row", { "dirtree-row--active": props.selected === "" }]}
          onClick={() => props.onSelect("")}
        >
          <VIcon size="14" class="mr-1">mdi-folder-home-outline</VIcon>
          <span>/ (root)</span>
        </div>
        {(props.root?.children ?? [])
          .filter((c) => c.dir)
          .map((c) => <DirRow key={c.path} node={c} selected={props.selected} depth={1} onSelect={props.onSelect} />)}
      </div>
    );
  },
});
