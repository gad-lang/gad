// UrlImportDialog — import a file from a URL. When the URL points to a
// ZIP/TAR/TAR.GZ, an "Extract archive" switch appears; if on, the download is
// passed to onImport as an archive for the host to extract. A target-folder
// picker (DirTree) chooses where to place the import. Reusable MUI component.
import { useMemo, useState } from "react";
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  IconButton,
  LinearProgress,
  Switch,
  TextField,
  Typography,
} from "@mui/material";
import FolderOpenIcon from "@mui/icons-material/FolderOpen";
import { DirTree } from "./DirTree";
import type { TreeNode } from "./api";

export interface UrlImportDialogProps {
  open: boolean;
  onClose: () => void;
  /** Download progress percent (0–100), or -1 for indeterminate. */
  progress?: number;
  /** Workspace tree, for the target-folder picker. */
  tree: TreeNode | null;
  onImport: (url: string, extract: boolean, targetDir: string) => Promise<void> | void;
}

/** UrlImportDialog renders the URL import dialog. */
export function UrlImportDialog({ open, onClose, progress = 0, tree, onImport }: UrlImportDialogProps) {
  const [url, setUrl] = useState("");
  const [extract, setExtract] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [targetDir, setTargetDir] = useState("");
  const [pickDir, setPickDir] = useState(false);
  const isArchive = useMemo(() => /\.(zip|tar|tar\.gz|tgz)(\?|#|$)/i.test(url), [url]);

  async function doImport() {
    if (!url.trim()) return;
    setBusy(true);
    setError("");
    try {
      await onImport(url.trim(), isArchive && extract, targetDir);
      setUrl("");
      onClose();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Import from URL</DialogTitle>
      <DialogContent>
        <TextField
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          onKeyUp={(e) => e.key === "Enter" && doImport()}
          label="URL"
          placeholder="https://example.com/file.gad"
          size="small"
          fullWidth
          autoFocus
          margin="dense"
        />
        {isArchive && (
          <FormControlLabel
            control={<Switch checked={extract} onChange={(e) => setExtract(e.target.checked)} />}
            label="Extract archive (ZIP / TAR / TAR.GZ)"
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
        {busy && (
          <Box sx={{ mt: 2 }}>
            <LinearProgress variant={progress < 0 ? "indeterminate" : "determinate"} value={progress < 0 ? undefined : progress} />
            {progress >= 0 && <Typography variant="caption" sx={{ display: "block", textAlign: "center", mt: 0.5 }}>{progress}%</Typography>}
          </Box>
        )}
        {error && <Typography color="error" variant="body2" sx={{ mt: 1 }}>{error}</Typography>}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" disabled={busy || !url.trim()} onClick={doImport}>Import</Button>
      </DialogActions>
    </Dialog>
  );
}
