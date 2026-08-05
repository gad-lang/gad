//go:build js && wasm && !gadwasmdebug

package main

import "syscall/js"

// registerDebug is a no-op in the normal WASM build: the debugger stepping
// protocol is excluded to keep the module smaller. The `_debug` build (build
// tag gadwasmdebug) installs the real protocol (debug.go).
func registerDebug() {
	js.Global().Set("gadHasDebugger", js.ValueOf(false))
}
