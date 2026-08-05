//go:build js && wasm

// Command wasm compiles the Gad bridge to WebAssembly. It installs functions on
// the JS global object — gadFormat, gadRun, gadDiagnose, gadDoc, … — each
// returning a JSON string with the same shape as the HTTP server's responses,
// so the React app can drive run, format, diagnostics and documentation
// entirely in the browser, with no Go web server.
//
// The module builds in two flavours selected by the `gadwasmdebug` build tag:
// the normal build (no tag) omits the debugger to stay small, while the
// `_debug` build (`-tags gadwasmdebug`) also installs the gadDebug* stepping
// protocol and a session-aware gadInspect (see debug.go / debug_off.go).
package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gad-lang/gad/web/gadbridge"
)

func main() {
	// gadFormat(source[, sourceType]) — formats keeping the dialect: "gadx",
	// "gadTemplate"/"template", or "gad"/"" (the default, plain Gad).
	js.Global().Set("gadFormat", jsonFuncN(func(args []js.Value) any {
		return gadbridge.FormatSource(argStr(args, 0), argStr(args, 1))
	}))
	// gadRun(source[, sourceType[, argsJSON]]) — sourceType "gadTemplate"/"gadx"
	// compiles the respective dialect; empty/"gad" runs plain Gad (the default).
	// argsJSON is a JSON array of strings passed to the script's `param`.
	js.Global().Set("gadRun", jsonFuncN(func(args []js.Value) any {
		return gadbridge.RunSourceArgs(argStr(args, 0), argStr(args, 1), argStrs(args, 2))
	}))
	// gadDiagnose(source[, sourceType]) — same dialect selection as gadRun.
	js.Global().Set("gadDiagnose", jsonFuncN(func(args []js.Value) any {
		return map[string]any{"diagnostics": gadbridge.DiagnoseSource(argStr(args, 0), argStr(args, 1))}
	}))
	// gadDoc(source, sourceType) -> { markdown } : documentation rendered as
	// Markdown. sourceType is "gad" | "gadTemplate" | "gadx".
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

	// gadDocComments(source) -> { docs } : the doc-comment list for the Docs panel.
	js.Global().Set("gadDocComments", jsonFuncN(func(args []js.Value) any {
		return map[string]any{"docs": gadbridge.DocComments(argStr(args, 0))}
	}))

	// gadEval(source, expr, repr) -> { ok, value, error, stdout } : evaluate expr
	// in a fresh VM seeded with source's top-level definitions.
	js.Global().Set("gadEval", jsonFuncN(func(args []js.Value) any {
		return gadbridge.EvalExpr(argStr(args, 0), argStr(args, 1), argBool(args, 2))
	}))

	// gadTranspile(source, mixed) -> FormatResult : rewrite template/mixed source
	// into plain Gad (write(...) calls). mixed=true for `.gadt`.
	js.Global().Set("gadTranspile", jsonFuncN(func(args []js.Value) any {
		return gadbridge.Transpile(argStr(args, 0), argBool(args, 1), nil)
	}))

	// gadInspect(session, expr, source) -> { ok, inspect } | { ok:false, error }
	// tree-navigator description of expr's value, evaluated fresh with source's
	// top-level definitions in scope. The `_debug` build overrides this with a
	// session-aware version that inspects the paused frame (see debug.go).
	js.Global().Set("gadInspect", jsonFuncN(func(args []js.Value) any {
		insp, err := gadbridge.InspectSource(argStr(args, 2), argStr(args, 1))
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		return map[string]any{"ok": true, "inspect": insp}
	}))

	// Install the debugger stepping protocol only in the `_debug` build (build
	// tag gadwasmdebug); a no-op in the normal build (debug_off.go).
	registerDebug()

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
