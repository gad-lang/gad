# IDE epic (`web/ide` backend + `web/app` React frontend)

Split from the original mega-bullet. Backend changes are unit-tested in
`web/ide`; frontend changes are typechecked (`tsc --noEmit`) — most need browser
verification for full sign-off.

## Done
- [x] Explorer: show/hide hidden files toggle (backend `?hidden=true` + eye button).
- [x] Locals panel: per-row copy-to-clipboard button.
- [x] Explorer: F2 / right-click context menu (open, run, format, transpile, rename, remove).
- [x] Remove confirmation dialog with recursive checkbox for non-empty dirs.
- [x] Run output: split into stdout/stderr files + combine flag (backend + API).
- [x] Explorer: "get file from web" backend (`/api/ide/fetch`, URL → workspace path).
- [x] Transpile: `gadbridge.Transpile` + `/api/ide/transpile`, config-driven `TranspileOptions`.
- [x] Settings dialog: Transpile options (writeFunc, rawStrFunc start/end → `.gad.yaml`).
- [x] Editor controls: reload-from-disk, undo, redo buttons.
- [x] Error dialog when a backend request fails (status line + modal, copyable).
- [x] Evaluate panel: add/edit/remove expressions, repr flag, output dialog, copy,
      re-eval on debug step. Backend `/api/ide/eval` (standalone, file as prelude).

## To do
- [x] gad codemirror plugin not working in the IDE: root cause was the default
      `gad ide` serving the build-free `cmd/gad/ideapp` (plain textarea, no
      CodeMirror) — the plugin only loaded in the React `web/app` (prod/--static
      build). Fixed by loading CodeMirror 6 + the gad language into the build-free
      app from a pinned ESM CDN, with a textarea fallback when offline.
- [ ] Run/Debug settings: one dialog with Run and Debug tabs; surface the split
      stdout/stderr file fields + combine flag in the Run tab.
- [ ] Explorer header: "get file from web" dialog (URL, output name, target dir).
- [x] Breakpoint condition + disabled: debug.Engine honours per-breakpoint
      Disabled flag and a Gad `condition` (evaluated in the paused frame's scope,
      pauses only when truthy); wired through /api/ide/debug/start
      (BreakpointSpecs). Breakpoints panel entry opens a dialog to edit
      disabled/condition (stored in .gad.yaml ide.breakpointMeta).
- [ ] Breakpoints panel: double-click an entry to navigate to its location
      (remove button + click-to-edit-condition done).
- [ ] Tooltip for builtin-value identifiers in the gad editor.
- [ ] Evaluate panel: debug-session-aware eval (run in the paused frame's scope,
      not a fresh VM) — needs a `debug.Engine` eval hook. (panel + standalone
      eval done; this is the frame-scope upgrade.)
- [ ] Multi-format editors: JSON, YAML, HTML, CSS, SCSS, JS (TS/JSX); plain-text
      fallback for other types.
- [ ] Tooltip: copy-to-clipboard button on the gad editor hover tooltip.
- [ ] codemirror plugin: editor features (autocomplete, etc.) inside template strings.
- [ ] Right closable panel rendering the current file's doc comments; reloads 5s
      after an edit (and a manual reload button).

- [x] STEP IN module import now opens the module file: the debugger reported
      imported-module frames as absolute `file:<abs>` paths (and the main file as
      `(main)`), and the UI never followed the stop into another file. Backend now
      normalizes debug file names to workspace-relative paths (DebugResponse.File
      + per-frame File via Server.normalizeDebugFile); the UI opens the stop's
      file and highlights the line there (debug.path follows res.file).
- [x] evaluate `1 + 1` showing the file's value instead of `2`: the file prelude
      ended in a top-level `return` which short-circuited the eval. Fixed with
      gadbridge.EvalSource (strips top-level returns, keeps definitions). repr's
      `‹int: 2›` is the correct repr form.

## Editor plugins (separate)
(none — keyword/builtin sync commands shipped via cmd/update-*-plugin.)
