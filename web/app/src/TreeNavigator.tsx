import { useEffect, useState } from "react";
import { Dialog, DialogActions, DialogContent, DialogTitle, Button } from "@mui/material";
import type { InspectEntry, InspectResult } from "./backends/ide";

/** Fetches the inspection of a Gad expression, or null on error. */
export type InspectFn = (expr: string) => Promise<InspectResult | null>;

// Extract the first workspace-relative .gad file path from a repr value string.
// Matches patterns like `‹module: mathx.gad›` or `‹compiledFunction: lib.gad.fn()›`.
function extractGadFile(value: string): string | null {
  const m = value.match(/([\w./\-]+\.gad)(?=[\W]|$)/i);
  return m ? m[1] : null;
}

// One lazily-expanding node. Children are fetched on first expand by appending
// the entry accessor to this node's expression.
function TreeNode({
  label,
  expr,
  type,
  value,
  expandable,
  inspect,
  onGotoSource,
  depth,
}: {
  label: string;
  expr: string;
  type: string;
  value: string;
  expandable: boolean;
  inspect: InspectFn;
  onGotoSource?: (file: string) => void;
  depth: number;
}) {
  const [open, setOpen] = useState(false);
  const [entries, setEntries] = useState<InspectEntry[] | null>(null);
  const [loading, setLoading] = useState(false);

  const toggle = async () => {
    if (!expandable) return;
    if (!open && entries === null) {
      setLoading(true);
      const r = await inspect(expr);
      setEntries(r?.entries ?? []);
      setLoading(false);
    }
    setOpen((o) => !o);
  };

  const gadFile = onGotoSource ? extractGadFile(value) : null;

  return (
    <div className="tn-node">
      <div className="tn-row" style={{ paddingLeft: depth * 14 }} onClick={toggle}>
        <span className="tn-twist">{expandable ? (open ? "▾" : "▸") : "·"}</span>
        <span className="tn-key">{label}</span>
        <span className="tn-type">{type}</span>
        <span className="tn-val">{value}</span>
        {gadFile && (
          <button
            className="tn-goto"
            title={`Go to source: ${gadFile}`}
            onClick={(e) => {
              e.stopPropagation();
              onGotoSource!(gadFile);
            }}
          >
            ↗
          </button>
        )}
      </div>
      {open && loading && (
        <div className="tn-loading" style={{ paddingLeft: (depth + 1) * 14 }}>
          …
        </div>
      )}
      {open &&
        entries?.map((e, i) => (
          <TreeNode
            key={i}
            label={e.key}
            // A child with no accessor (unsupported key) is shown but not drillable.
            expr={expr + e.accessor}
            type={e.type}
            value={e.value}
            expandable={e.expandable && e.accessor !== ""}
            inspect={inspect}
            onGotoSource={onGotoSource}
            depth={depth + 1}
          />
        ))}
    </div>
  );
}

/** TreeNavigator renders a drill-in tree for the value of rootExpr. */
export function TreeNavigator({
  rootExpr,
  rootLabel,
  inspect,
  onGotoSource,
}: {
  rootExpr: string;
  rootLabel: string;
  inspect: InspectFn;
  onGotoSource?: (file: string) => void;
}) {
  const [root, setRoot] = useState<InspectResult | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let alive = true;
    setError("");
    setRoot(null);
    inspect(rootExpr).then((r) => {
      if (!alive) return;
      if (r) setRoot(r);
      else setError("could not evaluate " + rootExpr);
    });
    return () => {
      alive = false;
    };
  }, [rootExpr, inspect]);

  if (error) return <div className="muted">{error}</div>;
  if (!root) return <div className="muted">…</div>;
  return (
    <div className="tree-nav">
      <TreeNode
        label={rootLabel}
        expr={rootExpr}
        type={root.type}
        value={root.value}
        expandable={root.expandable}
        inspect={inspect}
        onGotoSource={onGotoSource}
        depth={0}
      />
    </div>
  );
}

/** InspectDialog hosts a TreeNavigator in a modal. */
export function InspectDialog({
  title,
  rootExpr,
  inspect,
  onClose,
  onGotoSource,
}: {
  title: string;
  rootExpr: string;
  inspect: InspectFn;
  onClose: () => void;
  onGotoSource?: (file: string) => void;
}) {
  return (
    <Dialog open onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Inspect — {title}</DialogTitle>
      <DialogContent dividers>
        <TreeNavigator
          rootExpr={rootExpr}
          rootLabel={title}
          inspect={inspect}
          onGotoSource={onGotoSource}
        />
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={onClose}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  );
}
