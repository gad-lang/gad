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

## To do
- [ ] Diagnose why the gad codemirror plugin "isn't working" in the IDE and fix it.
- [ ] Run/Debug settings: one dialog with Run and Debug tabs; surface the split
      stdout/stderr file fields + combine flag in the Run tab.
- [ ] Explorer header: "get file from web" dialog (URL, output name, target dir).
- [ ] Breakpoint dialog: click a gutter breakpoint to edit `disabled` + a gad
      expression `condition` (pause only when `!value.IsFalsy()`); cancel/save.
- [ ] Breakpoints panel: per-entry remove button; click opens the breakpoint
      dialog; double-click navigates to the location.
- [ ] Tooltip for builtin-value identifiers in the gad editor.
- [ ] Evaluate panel: add-expression form (expr, `repr` flag, `+`), list with
      per-row edit / trash / output-dialog / copy; re-evaluate on debug step.
      (needs a backend evaluate endpoint, ideally debug-session aware.)
- [ ] File editor controls: "reload from disk" button.
- [ ] File editor controls: undo / redo buttons.
- [ ] Multi-format editors: JSON, YAML, HTML, CSS, SCSS, JS (TS/JSX); plain-text
      fallback for other types.
- [ ] Error dialog when a backend request fails.
- [ ] Tooltip: copy-to-clipboard button on the gad editor hover tooltip.
- [ ] codemirror plugin: editor features (autocomplete, etc.) inside template strings.
- [ ] Right closable panel rendering the current file's doc comments; reloads 5s
      after an edit (and a manual reload button).

## Editor plugins (separate)
(none — keyword/builtin sync commands shipped via cmd/update-*-plugin.)
