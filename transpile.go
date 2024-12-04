package gad

import "github.com/gad-lang/gad/parser/node"

func TranspileOptions() *node.TranspileOptions {
	return &node.TranspileOptions{
		RawStrFunc: BuiltinRawStr.String(),
		WriteFunc:  "write",
	}
}
