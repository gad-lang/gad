// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import "strings"

// ParseArgs splits command-line-style arguments into positional and named values
// for a script's `param (…)` declaration, the way the `gad` CLI does — plus
// typed coercion: each value is parsed as a Gad expression, so a positional
// `param (doc dict)` receives a real dict, `--count=5` a real int, `--on` a
// boolean flag (Yes), etc. A value that does not parse as an expression is used
// verbatim as a string.
//
// Rules:
//   - `--name=<expr>` — a named argument whose value is the parsed expression.
//   - `--name`        — a boolean flag (named argument set to Yes).
//   - anything else   — a positional argument (its value parsed as an expression).
//   - a lone `--`     — terminates option parsing; every following token is
//     positional, even one starting with `--`.
//
// The returned Dict/Array plug directly into RunOpts:
//
//	pos, named := gad.ParseArgs(osArgs)
//	vm.RunOpts(&gad.RunOpts{
//	    Args:      gad.Args{pos},
//	    NamedArgs: gad.NewNamedArgs(gad.MustConvertToKeyValueArray(nil, named)),
//	})
func ParseArgs(args []string) (positional Array, named Dict) {
	positional = Array{}
	named = Dict{}
	rest := false
	for _, a := range args {
		if !rest && a == "--" {
			rest = true
			continue
		}
		if !rest && strings.HasPrefix(a, "--") && len(a) > 2 {
			body := a[2:]
			if eq := strings.IndexByte(body, '='); eq >= 0 {
				named[body[:eq]] = parseArgValue(body[eq+1:])
			} else {
				named[body] = Yes
			}
			continue
		}
		positional = append(positional, parseArgValue(a))
	}
	return positional, named
}

// ParseArgsToRunOpts is a convenience over ParseArgs that returns the positional
// arguments and named arguments already shaped for RunOpts.
func ParseArgsToRunOpts(args []string) (Args, *NamedArgs) {
	pos, named := ParseArgs(args)
	return Args{pos}, NewNamedArgs(MustConvertToKeyValueArray(nil, named))
}

// parseArgValue evaluates s as a Gad expression to a typed value (int, dict,
// array, bool, string literal, …); a value that fails to parse or run is used
// verbatim as a string.
func parseArgValue(s string) Object {
	bl := NewBuiltins()
	st := NewSymbolTable(bl.NameSet)
	cr, err := Compile(st, []byte("return ("+s+")"), CompileOptions{})
	if err != nil {
		return Str(s)
	}
	ret, rerr := NewVM(bl.Build(), cr.Bytecode).SetRecover(true).RunOpts(&RunOpts{})
	if rerr != nil || ret == nil {
		return Str(s)
	}
	return ret
}
