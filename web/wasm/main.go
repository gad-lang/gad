//go:build js && wasm

// Command wasm compiles the Gad bridge to WebAssembly. It installs functions on
// the JS global object — gadFormat, gadRun, gadDiagnose, gadDoc, and the
// gadDebug* stepping protocol — each returning a JSON string with the same shape
// as the HTTP server's responses, so the React app can drive run, format,
// diagnostics, documentation and a full debugger entirely in the browser, with
// no Go web server.
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gad-lang/gad/web/gadbridge"
)

// dbg is the single debug-session manager for this WASM instance. Running one
// session at a time in a Web Worker keeps the model simple: a blocking VM run
// stays off the UI thread.
var dbg = gadbridge.NewDebugManager()

func main() {
	js.Global().Set("gadFormat", jsonFunc(func(src string) any { return gadbridge.Format(src) }))
	js.Global().Set("gadRun", jsonFunc(func(src string) any { return gadbridge.Run(src) }))
	js.Global().Set("gadDiagnose", jsonFunc(func(src string) any {
		return map[string]any{"diagnostics": gadbridge.Diagnose(src)}
	}))
	// gadDoc(source, sourceType) -> { markdown } : documentation rendered as
	// Markdown. sourceType is "gad" | "gadTemplate" | "giom".
	js.Global().Set("gadDoc", jsonFuncN(func(args []js.Value) any {
		md, err := gadbridge.Doc(argStr(args, 0), argStr(args, 1))
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"markdown": md}
	}))

	// gadDocData(source, sourceType) -> { doc } : the structured documentation
	// (JSON: prose + sections[] of symbols) for custom client-side rendering.
	js.Global().Set("gadDocData", jsonFuncN(func(args []js.Value) any {
		d, err := gadbridge.ExtractDoc(argStr(args, 0), argStr(args, 1))
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return map[string]any{"doc": d}
	}))

	// Debug stepping protocol (mirrors /api/debug/*).
	// gadDebugStart(source, path, breakpointsJSON, stopOnEntry, argsJSON)
	js.Global().Set("gadDebugStart", jsonFuncN(func(args []js.Value) any {
		return dbg.Start(gadbridge.DebugStartRequest{
			Source:      argStr(args, 0),
			Path:        argStr(args, 1),
			Breakpoints: argInts(args, 2),
			StopOnEntry: argBool(args, 3),
			Args:        argStrs(args, 4),
		})
	}))
	// gadDebugCommand(session, command)
	js.Global().Set("gadDebugCommand", jsonFuncN(func(args []js.Value) any {
		return dbg.Command(argStr(args, 0), argStr(args, 1))
	}))
	// gadDebugEval(session, expr, repr) -> { ok, value } | { ok:false, error }
	js.Global().Set("gadDebugEval", jsonFuncN(func(args []js.Value) any {
		value, err := dbg.Eval(argStr(args, 0), argStr(args, 1), argBool(args, 2))
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		return map[string]any{"ok": true, "value": value}
	}))
	// gadDebugStop(session)
	js.Global().Set("gadDebugStop", jsonFuncN(func(args []js.Value) any {
		dbg.Stop(argStr(args, 0))
		return map[string]any{"ok": true}
	}))

	// Signal readiness, then block forever so the exported functions stay live.
	js.Global().Set("gadReady", js.ValueOf(true))
	if cb := js.Global().Get("onGadReady"); cb.Type() == js.TypeFunction {
		cb.Invoke()
	}
	select {}
}

// jsonFuncN wraps fn as a JS function taking any number of args and returning a
// JSON string.
func jsonFuncN(fn func(args []js.Value) any) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		data, err := json.Marshal(fn(args))
		if err != nil {
			return `{"error":` + jsString(err.Error()) + `}`
		}
		return string(data)
	})
}

func argStr(args []js.Value, i int) string {
	if i < len(args) && args[i].Type() == js.TypeString {
		return args[i].String()
	}
	return ""
}

func argBool(args []js.Value, i int) bool {
	return i < len(args) && args[i].Type() == js.TypeBoolean && args[i].Bool()
}

// argInts reads a JSON array of ints passed as a string (e.g. "[1,3]").
func argInts(args []js.Value, i int) []int {
	var out []int
	if s := argStr(args, i); s != "" {
		_ = json.Unmarshal([]byte(s), &out)
	}
	return out
}

// argStrs reads a JSON array of strings passed as a string.
func argStrs(args []js.Value, i int) []string {
	var out []string
	if s := argStr(args, i); s != "" {
		_ = json.Unmarshal([]byte(s), &out)
	}
	return out
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
