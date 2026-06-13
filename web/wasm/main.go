//go:build js && wasm

// Command wasm compiles the Gad bridge to WebAssembly. It installs three
// functions on the JS global object — gadFormat, gadRun and gadDiagnose — each
// taking a source string and returning a JSON string with the same shape as the
// HTTP server's responses, so the React app can use either backend
// interchangeably.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gad-lang/gad/web/gadbridge"
)

func main() {
	js.Global().Set("gadFormat", jsonFunc(func(src string) any { return gadbridge.Format(src) }))
	js.Global().Set("gadRun", jsonFunc(func(src string) any { return gadbridge.Run(src) }))
	js.Global().Set("gadDiagnose", jsonFunc(func(src string) any {
		return map[string]any{"diagnostics": gadbridge.Diagnose(src)}
	}))

	// Signal readiness, then block forever so the exported functions stay live.
	js.Global().Set("gadReady", js.ValueOf(true))
	if cb := js.Global().Get("onGadReady"); cb.Type() == js.TypeFunction {
		cb.Invoke()
	}
	select {}
}

// jsonFunc wraps fn as a JS function: (source string) => json string.
func jsonFunc(fn func(src string) any) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		src := ""
		if len(args) > 0 {
			src = args[0].String()
		}
		data, err := json.Marshal(fn(src))
		if err != nil {
			return `{"error":` + jsString(err.Error()) + `}`
		}
		return string(data)
	})
}

func jsString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
