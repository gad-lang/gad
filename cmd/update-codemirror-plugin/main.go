// Command update-codemirror-plugin keeps web/codemirror-gad in sync with the Gad
// language: it adds any keywords/atoms/constants/builtins the plugin is missing
// and reports the language commits since the plugin was last updated.
//
// Usage:
//
//	go run ./cmd/update-codemirror-plugin      # dry run (report only)
//	go run ./cmd/update-codemirror-plugin -w   # apply the additions
package main

import "github.com/gad-lang/gad/cmd/internal/pluginsync"

func main() {
	pluginsync.RunCLI(pluginsync.Target{
		Name:   "codemirror-gad",
		Dir:    "web/codemirror-gad",
		File:   "web/codemirror-gad/src/keywords.ts",
		Arrays: []string{"keywords", "atoms", "constants", "builtins"},
	})
}
