#!/usr/bin/env bash
# Build the Gad WASM module for the React app / webide and copy Go's
# wasm_exec.js into public/. The app hosts the in-browser IDE (debug/inspect),
# so it uses the debugger-enabled build, published under the plain gad.wasm name
# the app loads. Delegates to the repo-level scripts/build-wasm.sh.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # web/app
repo="$(cd "$here/../.." && pwd)"                          # repo root
out="$here/public"

bash "$repo/scripts/build-wasm.sh" "$out" debug
# The app loads "gad.wasm"; use the debugger build under that name.
mv -f "$out/gad_debug.wasm" "$out/gad.wasm"
echo "wrote $out/gad.wasm (debugger) and $out/wasm_exec.js"
