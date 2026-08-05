// DirTree — a compact, directory-only tree for picking an upload target folder.
// It renders the workspace root as "/" plus every directory node; clicking a row
// selects it (selected path "" = the workspace root). Reusable MUI component.
import { Box } from "@mui/material";
import FolderOutlinedIcon from "@mui/icons-material/FolderOutlined";
import HomeOutlinedIcon from "@mui/icons-material/HomeOutlined";
import type { TreeNode } from "./api";

export interface DirTreeProps {
  root: TreeNode | null;
  selected: string;
  onSelect: (path: string) => void;
}

const rowSx = (active: boolean) => ({
  display: "flex",
  alignItems: "center",
  gap: 0.5,
  py: 0.25,
  px: 0.75,
  cursor: "pointer",
  borderRadius: 1,
  whiteSpace: "nowrap",
  bgcolor: active ? "primary.main" : "transparent",
  color: active ? "primary.contrastText" : "inherit",
  "&:hover": { bgcolor: active ? "primary.main" : "action.hover" },
});

function DirRow({ node, selected, depth, onSelect }: { node: TreeNode; selected: string; depth: number; onSelect: (p: string) => void }) {
  const dirs = (node.children ?? []).filter((c) => c.dir);
  return (
    <>
      <Box sx={{ ...rowSx(selected === node.path), pl: `${6 + depth * 14}px` }} onClick={() => onSelect(node.path)}>
        <FolderOutlinedIcon sx={{ fontSize: 14 }} />
        <Box component="span" sx={{ overflow: "hidden", textOverflow: "ellipsis" }}>{node.name}</Box>
      </Box>
      {dirs.map((c) => (
        <DirRow key={c.path} node={c} selected={selected} depth={depth + 1} onSelect={onSelect} />
      ))}
    </>
  );
}

/** DirTree renders the directory-only picker rooted at the workspace root. */
export function DirTree({ root, selected, onSelect }: DirTreeProps) {
  return (
    <Box sx={{ fontSize: 13 }}>
      <Box sx={rowSx(selected === "")} onClick={() => onSelect("")}>
        <HomeOutlinedIcon sx={{ fontSize: 14 }} />
        <Box component="span">/ (root)</Box>
      </Box>
      {(root?.children ?? [])
        .filter((c) => c.dir)
        .map((c) => (
          <DirRow key={c.path} node={c} selected={selected} depth={1} onSelect={onSelect} />
        ))}
    </Box>
  );
}
