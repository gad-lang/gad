#!/usr/bin/env bash
# Build the Gad WASM module (gad.wasm, debugger-enabled) and copy Go's
# wasm_exec.js into the React app's public/. Delegates to the repo-level
# scripts/build-wasm.sh.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # web/app
repo="$(cd "$here/../.." && pwd)"                          # repo root

bash "$repo/scripts/build-wasm.sh" "$here/public"
