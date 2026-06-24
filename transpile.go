package gad

import "github.com/gad-lang/gad/parser/node"

func TranspileOptions() *node.TranspileOptions {
	return &node.TranspileOptions{
		// `raw "…"` is the raw-string operator, which takes no suffix, so
		// RawStrFuncEnd is empty (matches the CLI's transpile defaults).
		RawStrFuncStart: "raw ",
		RawStrFuncEnd:   "",
		WriteFunc:       "write",
	}
}
