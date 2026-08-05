//go:build js && wasm && gadwasmdebug

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/gad-lang/gad/web/gadbridge"
)

// dbg is the single debug-session manager for this WASM instance. Running one
// session at a time in a Web Worker keeps the model simple: a blocking VM run
// stays off the UI thread. Compiled only into the `_debug` WASM.
var dbg = gadbridge.NewDebugManager()

// registerDebug installs the debugger stepping protocol (gadDebug*, mirroring
// /api/debug/*) and replaces gadInspect with a session-aware version that can
// inspect the paused frame. It is compiled only into the `_debug` WASM build
// (build tag gadwasmdebug); the normal build ships a no-op (debug_off.go).
func registerDebug() {
	// gadDebugStart(source, path, breakpointsJSON, stopOnEntry, argsJSON[, specsJSON])
	// specsJSON is a JSON array of { line, disabled, condition } — conditional
	// breakpoints that take precedence over breakpointsJSON when present.
	js.Global().Set("gadDebugStart", jsonFuncN(func(args []js.Value) any {
		var specs []gadbridge.BreakpointSpec
		if s := argStr(args, 5); s != "" {
			_ = json.Unmarshal([]byte(s), &specs)
		}
		return dbg.Start(gadbridge.DebugStartRequest{
			Source:          argStr(args, 0),
			Path:            argStr(args, 1),
			Breakpoints:     argInts(args, 2),
			StopOnEntry:     argBool(args, 3),
			Args:            argStrs(args, 4),
			BreakpointSpecs: specs,
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

	// gadInspect(session, expr, source) -> { ok, inspect } | { ok:false, error }
	// tree-navigator description of expr's value: in the paused frame when session
	// is set, else evaluated fresh with source's top-level definitions in scope.
	js.Global().Set("gadInspect", jsonFuncN(func(args []js.Value) any {
		session, expr := argStr(args, 0), argStr(args, 1)
		var (
			insp gadbridge.InspectResult
			err  error
		)
		if session != "" {
			insp, err = dbg.Inspect(session, expr)
		} else {
			insp, err = gadbridge.InspectSource(argStr(args, 2), expr)
		}
		if err != nil {
			return map[string]any{"ok": false, "error": err.Error()}
		}
		return map[string]any{"ok": true, "inspect": insp}
	}))

	// Advertise the debugger so the JS side can gate debug UI on it.
	js.Global().Set("gadHasDebugger", js.ValueOf(true))
}
