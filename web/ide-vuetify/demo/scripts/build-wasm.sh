#!/usr/bin/env bash
# Build the Gad WASM module for the Vuetify IDE demo and copy Go's wasm_exec.js
# into the demo's public/. The demo is a full in-browser IDE (debug/inspect), so
# it uses the debugger-enabled build under the plain gad.wasm name the demo
# loads. Delegates to the repo-level scripts/build-wasm.sh.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # ide-vuetify/demo
repo="$(cd "$here/../../.." && pwd)"                       # repo root
out="$here/public"

bash "$repo/scripts/build-wasm.sh" "$out" debug
# The demo loads "gad.wasm"; use the debugger build under that name.
mv -f "$out/gad_debug.wasm" "$out/gad.wasm"
echo "wrote $out/gad.wasm (debugger) and $out/wasm_exec.js"
