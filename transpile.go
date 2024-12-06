package gad

import "github.com/gad-lang/gad/parser/node"

func TranspileOptions() *node.TranspileOptions {
	return &node.TranspileOptions{
		RawStrFuncStart: BuiltinRawStr.String() + "(",
		RawStrFuncEnd:   ";cast)",
		WriteFunc:       "write",
	}
}
