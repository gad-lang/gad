package main

import (
	"fmt"
	"io"

	"github.com/gad-lang/gad"
)

// printCompileWarnings writes each compiler warning to w (STDERR), one per line,
// including its source position and detail (CompilerWarning.Error format).
func printCompileWarnings(w io.Writer, warnings []*gad.CompilerWarning) {
	for _, warn := range warnings {
		fmt.Fprintln(w, warn.Error())
	}
}
