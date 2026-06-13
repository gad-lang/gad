#!/usr/bin/env bash
# Build the Gad WASM module and copy Go's wasm_exec.js into public/.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo="$(cd "$here/../.." && pwd)"
out="$here/public"

mkdir -p "$out"

echo "building gad.wasm ..."
( cd "$repo" && GOOS=js GOARCH=wasm go build -o "$out/gad.wasm" ./web/wasm )

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
