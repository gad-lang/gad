// UploadReviewDialog — shown after picking or dropping files, before onUpload
// runs. It reviews the files, lets a single file be renamed and (if it is a
// ZIP/TAR/TAR.GZ) extracted, confirms the target directory, and warns when the
// destination already exists (requiring "Replace existing" to proceed). Reusable
// MUI component: the host supplies archiveKind/pathExists/tree via props.
import { useEffect, useMemo, useState } from "react";
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  Switch,
  TextField,
  Typography,
} from "@mui/material";
import FolderOpenIcon from "@mui/icons-material/FolderOpen";
import { DirTree } from "./DirTree";
import { readBase64, type RawFile } from "./upload";
import type { TreeNode, UploadedFile } from "./api";

type ArchiveKind = "zip" | "tar" | "tar.gz";

export interface UploadReviewDialogProps {
  open: boolean;
  onClose: () => void;
  raw: RawFile[];
  initialDir?: string;
  tree: TreeNode | null;
  /** Return the archive kind for a name, or undefined when it is not an archive. */
  archiveKind: (name: string) => ArchiveKind | undefined;
  /** Whether a destination path already exists in the workspace. */
  pathExists: (path: string) => boolean;
  onConfirm: (files: UploadedFile[], targetDir: string) => void;
}

const baseName = (p: string) => p.slice(p.lastIndexOf("/") + 1);
const join = (dir: string, rel: string) => (dir ? dir + "/" + rel : rel);

/** UploadReviewDialog renders the pre-upload review dialog. */
export function UploadReviewDialog({ open, onClose, raw, initialDir = "", tree, archiveKind, pathExists, onConfirm }: UploadReviewDialogProps) {
  const single = raw.length === 1;
  const [targetDir, setTargetDir] = useState("");
  const [name, setName] = useState("");
  const [extract, setExtract] = useState(true);
  const [replace, setReplace] = useState(false);
  const [pickDir, setPickDir] = useState(false);
  const [busy, setBusy] = useState(false);

  const archiveOf = single ? archiveKind(name) : undefined;

  // Reset the form each time the dialog opens with a new file set.
  useEffect(() => {
    if (!open) return;
    setTargetDir(initialDir);
    setName(single ? baseName(raw[0].path) : "");
    setExtract(true);
    setReplace(false);
    setPickDir(false);
  }, [open, initialDir, single, raw]);

  // Final destination paths (for the existence check). When extracting, the
  // archive expands into a folder the host manages, so per-entry paths are
  // unknown here — collision detection is skipped for that case.
  const finalPaths = useMemo<string[]>(() => {
    if (single) {
      if (archiveOf && extract) return [];
      return [join(targetDir, name)];
    }
    return raw.map((r) => join(targetDir, r.path));
  }, [single, archiveOf, extract, targetDir, name, raw]);
  const collisions = finalPaths.filter((p) => pathExists(p));
  const blocked = collisions.length > 0 && !replace;

  async function confirm() {
    setBusy(true);
    try {
      let files: UploadedFile[];
      if (single && archiveOf && extract) {
        files = [{ path: name, content: "", archive: archiveOf, bytes: await readBase64(raw[0].file) }];
      } else if (single) {
        files = [{ path: name, content: await raw[0].file.text() }];
      } else {
        files = await Promise.all(raw.map(async (r) => ({ path: r.path, content: await r.file.text() })));
      }
      onConfirm(files, targetDir);
      onClose();
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{single ? "Upload file" : `Upload ${raw.length} files`}</DialogTitle>
      <DialogContent>
        {single ? (
          <TextField
            value={name}
            onChange={(e) => setName(e.target.value)}
            label="Destination name"
            size="small"
            fullWidth
            margin="dense"
          />
        ) : (
          <Box sx={{ fontSize: 12, mb: 1 }}>
            {raw.slice(0, 6).map((r) => (
              <Box key={r.path} sx={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{r.path}</Box>
            ))}
            {raw.length > 6 && <Box sx={{ color: "text.secondary" }}>…and {raw.length - 6} more</Box>}
          </Box>
        )}

        {single && archiveOf && (
          <FormControlLabel
            control={<Switch checked={extract} onChange={(e) => setExtract(e.target.checked)} />}
            label={`Extract archive (${archiveOf.toUpperCase()})`}
          />
        )}

        <Box sx={{ display: "flex", alignItems: "center", gap: 0.5, mt: 1 }}>
          <TextField
            value={targetDir || "/ (root)"}
            label="Target folder"
            size="small"
            fullWidth
            slotProps={{ input: { readOnly: true } }}
            onClick={() => setPickDir((v) => !v)}
          />
          <IconButton size="small" title="Choose folder" onClick={() => setPickDir((v) => !v)}>
            <FolderOpenIcon fontSize="small" />
          </IconButton>
        </Box>
        {pickDir && (
          <Box sx={{ mt: 1, maxHeight: 180, overflow: "auto", border: 1, borderColor: "divider", borderRadius: 1, p: 0.5 }}>
            <DirTree root={tree} selected={targetDir} onSelect={(p) => { setTargetDir(p); setPickDir(false); }} />
          </Box>
        )}

        {collisions.length > 0 && (
          <FormControlLabel
            control={<Checkbox color="error" checked={replace} onChange={(e) => setReplace(e.target.checked)} />}
            label={collisions.length === 1
              ? `"${collisions[0]}" already exists — replace it`
              : `${collisions.length} files already exist — replace them`}
            sx={{ mt: 1, color: "error.main" }}
          />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={busy || blocked || (single && !name.trim())} onClick={confirm}>Upload</Button>
      </DialogActions>
    </Dialog>
  );
}
