#!/usr/bin/env bash
# Build the Gad WebAssembly module(s) and copy Go's wasm_exec.js into an output
# directory. Two flavours are produced, selected by the `gadwasmdebug` build tag
# on ./web/wasm:
#   - gad.wasm       : normal build, no debugger (smaller)
#   - gad_debug.wasm : includes the gadDebug* stepping protocol
#
# Usage: scripts/build-wasm.sh <out-dir> [variant]
#   variant: normal | debug | both   (default: both)
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="${1:?usage: build-wasm.sh <out-dir> [normal|debug|both]}"
variant="${2:-both}"

mkdir -p "$out"

build_normal() {
  echo "building gad.wasm (no debugger) ..."
  ( cd "$repo" && GOOS=js GOARCH=wasm go build -o "$out/gad.wasm" ./web/wasm )
}
build_debug() {
  echo "building gad_debug.wasm (with debugger) ..."
  ( cd "$repo" && GOOS=js GOARCH=wasm go build -tags gadwasmdebug -o "$out/gad_debug.wasm" ./web/wasm )
}

case "$variant" in
  normal) build_normal ;;
  debug)  build_debug ;;
  both)   build_normal; build_debug ;;
  *) echo "unknown variant: $variant (want normal|debug|both)" >&2; exit 2 ;;
esac

# Copy Go's wasm_exec.js runtime loader next to the modules.
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

echo "wrote WASM assets to $out"
