#!/usr/bin/env bash
# Build the Gad WASM module (gad.wasm, debugger-enabled) and copy Go's
# wasm_exec.js into the Vuetify IDE demo's public/. Delegates to the repo-level
# scripts/build-wasm.sh.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # ide-vuetify/demo
repo="$(cd "$here/../../.." && pwd)"                       # repo root

bash "$repo/scripts/build-wasm.sh" "$here/public"
