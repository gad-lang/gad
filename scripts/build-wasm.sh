#!/usr/bin/env bash
# Build the Gad WebAssembly module (gad.wasm, debugger-enabled) and copy Go's
# wasm_exec.js into an output directory.
#
# Usage: scripts/build-wasm.sh <out-dir>
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="${1:?usage: build-wasm.sh <out-dir>}"

mkdir -p "$out"

echo "building gad.wasm ..."
( cd "$repo" && GOOS=js GOARCH=wasm go build -o "$out/gad.wasm" ./web/wasm )

# Copy Go's wasm_exec.js runtime loader next to the module.
goroot="$(go env GOROOT)"
exec_js=""
for cand in "$goroot/lib/wasm/wasm_exec.js" "$goroot/misc/wasm/wasm_exec.js"; do
  if [ -f "$cand" ]; then exec_js="$cand"; break; fi
done
if [ -z "$exec_js" ]; then
  echo "could not find wasm_exec.js under $goroot" >&2
  exit 1
fi
# The source under the Go module cache is read-only; remove any prior copy and
# restore write permission so re-runs don't fail.
rm -f "$out/wasm_exec.js"
cp "$exec_js" "$out/wasm_exec.js"
chmod u+w "$out/wasm_exec.js"

echo "wrote $out/gad.wasm and $out/wasm_exec.js"
