// Command update-prism-plugin keeps plugins/js/prism-gad in sync with the Gad language:
// it adds any keywords/atoms/builtins the grammar is missing and reports the
// language commits since the plugin was last updated.
//
// Usage:
//
//	go run ./cmd/update-prism-plugin      # dry run (report only)
//	go run ./cmd/update-prism-plugin -w   # apply the additions
package main

import "github.com/gad-lang/gad/cmd/internal/pluginsync"

func main() {
	pluginsync.RunCLI(pluginsync.Target{
		Name:   "prism-gad",
		Dir:    "plugins/js/prism-gad",
		File:   "plugins/js/prism-gad/src/index.ts",
		Arrays: []string{"keywords", "atoms", "builtins"},
	})
}
