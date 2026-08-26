#!/usr/bin/env bash
# Provide the Gad WASM module (gad.wasm + Go's wasm_exec.js) in the Vuetify IDE
# demo's public/ directory.
#
# Two modes, chosen automatically:
#   * Inside the gad-lang/gad monorepo (this dir is web/ide-vuetify/demo): build
#     it from source via the repo-level scripts/build-wasm.sh (debugger-enabled).
#   * Standalone (the extracted gad-lang/ide-vuetify repo, e.g. the Pages CI):
#     fetch the published WASM from the docs site — override the source with
#     GAD_WASM_BASE.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # ide-vuetify/demo
public="$here/public"
repo_build="$here/../../../scripts/build-wasm.sh"

if [ -f "$repo_build" ]; then
  echo "build-wasm: building from the Go monorepo"
  bash "$repo_build" "$public"
else
  base="${GAD_WASM_BASE:-https://gad-lang.github.io/latest}"
  echo "build-wasm: fetching published WASM from $base"
  mkdir -p "$public"
  curl -fsSL "$base/gad.wasm" -o "$public/gad.wasm"
  curl -fsSL "$base/wasm_exec.js" -o "$public/wasm_exec.js"
  echo "wrote $public/gad.wasm and $public/wasm_exec.js"
fi
