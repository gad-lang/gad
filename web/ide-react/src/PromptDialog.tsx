// PromptDialog / ConfirmDialog — in-app replacements for window.prompt /
// window.confirm. Each is driven by a request object carrying a resolve callback;
// pass `null` to keep it closed. Reusable MUI components.
import { useEffect, useState } from "react";
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
} from "@mui/material";

/** PromptRequest asks the user for a string; resolve(null) on cancel. */
export interface PromptRequest {
  title: string;
  label?: string;
  initial: string;
  resolve: (value: string | null) => void;
}

/** ConfirmRequest asks a yes/no question; resolve(false) on cancel. */
export interface ConfirmRequest {
  title: string;
  message: string;
  resolve: (ok: boolean) => void;
}

/** PromptDialog renders the in-app prompt. */
export function PromptDialog({ request, onDone }: { request: PromptRequest | null; onDone: () => void }) {
  const [value, setValue] = useState("");
  useEffect(() => {
    if (request) setValue(request.initial);
  }, [request]);

  const submit = (ok: boolean) => {
    request?.resolve(ok ? value.trim() : null);
    onDone();
  };

  return (
    <Dialog open={!!request} onClose={() => submit(false)} maxWidth="xs" fullWidth>
      <DialogTitle>{request?.title}</DialogTitle>
      <DialogContent>
        <TextField
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyUp={(e) => e.key === "Enter" && submit(true)}
          label={request?.label}
          size="small"
          fullWidth
          autoFocus
          margin="dense"
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={() => submit(false)}>Cancel</Button>
        <Button variant="contained" onClick={() => submit(true)}>OK</Button>
      </DialogActions>
    </Dialog>
  );
}

/** ConfirmDialog renders the in-app confirm. */
export function ConfirmDialog({ request, onDone }: { request: ConfirmRequest | null; onDone: () => void }) {
  const answer = (ok: boolean) => {
    request?.resolve(ok);
    onDone();
  };
  return (
    <Dialog open={!!request} onClose={() => answer(false)} maxWidth="xs" fullWidth>
      <DialogTitle>{request?.title}</DialogTitle>
      <DialogContent>
        <DialogContentText>{request?.message}</DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => answer(false)}>Cancel</Button>
        <Button variant="contained" onClick={() => answer(true)}>OK</Button>
      </DialogActions>
    </Dialog>
  );
}
