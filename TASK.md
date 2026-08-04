# TASK: Vuetify IDE — dockview panels, two v-models, Settings (TSX)

> Created: 2026-08-03 | Updated: 2026-08-04 00:20

## Goal
Rework @gad-lang/ide-vuetify: resizable/movable dockview panels; a Settings
dialog (Panels/Formatter/Transpile/Template); two independent v-models —
`layoutConfig` (dockview layout) and `config` (settings) — persisted to
LocalStorage by the demo; hover-locals like `gad ide`. Apply the layout
save/restore to the React `<Ide>` too. Components authored in TSX (chosen over
.vue/.jsx: keeps full TS typing, matches the React package).

## Log
### 2026-08-04
- TSX toolchain validated — added @vitejs/plugin-vue-jsx + tsconfig
  jsxImportSource:"vue"; vue-tsc canary passed
- Extracted controller.ts (framework-agnostic state/actions), provided via
  inject; 5 dockview panels (Explorer/Editor/CallStack/Locals/Output) inject it
- GadIde.tsx: top toolbar (Format/Doc/Run/Debug/Settings + step controls) +
  DockviewVue; v-model:layoutConfig (toJSON/fromJSON, echo-guarded) and
  v-model:config; Settings + Doc dialogs
- Vuetify+TSX friction (native onClick/title/onKeyup not typed) → contained in
  src/vuetify.ts (permissive re-exports); our own types stay strict
- dockview theme CSS is the consumer's import (like Vuetify) → removed from the
  lib, added to the demo; externalized /^dockview/ and /^vuetify/ in the lib build
- Hover-locals wired (editor getLocals ← paused frame); inspect icons + tree
  dialog retained
- React `<Ide>`: added optional `layoutConfig` + `onLayoutConfigChange`
  (additive, ref-based) restoring/saving the dockview layout
- Verified: ide-vuetify typecheck + lib build; demo typecheck + production build
  (dockview + Vuetify + worker); ide-react + app typecheck — all clean

## Errors & Fixes
| Error | Cause | Fix | Evidence |
|-------|-------|-----|----------|
| JSX.IntrinsicElements missing | no Vue JSX types | tsconfig jsxImportSource:"vue" | canary tc exit 0 |
| Vuetify props reject onClick/title/onKeyup | Vuetify types omit native attrs in TSX | src/vuetify.ts permissive re-exports | pkg tc exit 0 |
| Rollup can't resolve dockview-core css (lib) | lib shouldn't bundle theme css | consumer imports it; externalize dockview | pkg build ok |
| demo can't resolve dockview-core css | not a direct demo dep (bun isolation) | add dockview-core to demo deps | demo build ok |

## Current State
@gad-lang/ide-vuetify is now a TSX package: a `controller.ts` holds all reactive
state/actions and is provided via inject to five dockview-vue panels (Explorer,
Editor, Call Stack, Locals, Output) that are resizable, movable, dockable and
tabbable. GadIde.tsx adds a top toolbar (Run/Debug/Format/Doc/Settings + step
controls) and exposes two independent v-models — `layoutConfig` (dockview
toJSON/fromJSON) and `config` (the Settings document: Panels/Formatter/Transpile/
Template). Hovering an identifier while paused shows its value; Locals/Evaluate
rows have an inspect icon opening the value tree navigator. Vuetify+TSX native-
attribute friction is contained in src/vuetify.ts (permissive re-exports) so our
own code stays fully typed. The demo (App.tsx) binds both v-models to
LocalStorage (and seeds config from the backend). The dockview theme CSS is
imported by the host (demo main.ts), like Vuetify's. The React `<Ide>` gained
additive `layoutConfig`/`onLayoutConfigChange` props that restore/persist its
dockview layout. Everything typechecks and builds (package lib, demo production,
ide-react, app). NOT verified live in a browser — dockview-vue rendering,
provide/inject teleport, and Vuetify interactions need a browser check (run
`make ide-vuetify-demo`). Committing now.
