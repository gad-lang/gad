// Command update-vscode-plugin regenerates the VS Code extension's TextMate
// grammar (editors/vscode-gad/syntaxes/gad.tmLanguage.json) from the current Gad
// language vocabulary, so highlighting stays in sync with the compiler. It also
// reports the language commits since the extension was last updated.
//
// Usage:
//
//	go run ./cmd/update-vscode-plugin      # dry run (report only)
//	go run ./cmd/update-vscode-plugin -w   # write the grammar
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gad-lang/gad/cmd/internal/pluginsync"
)

func main() {
	write := flag.Bool("w", false, "write the generated grammar to the extension")
	flag.Parse()

	const (
		dir     = "editors/vscode-gad"
		grammar = "editors/vscode-gad/syntaxes/gad.tmLanguage.json"
	)

	data, err := pluginsync.TextMateGrammar()
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate grammar:", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	fmt.Println("== vscode-gad (gad.tmLanguage.json) ==")
	old, _ := os.ReadFile(grammar)
	switch {
	case string(old) == string(data):
		fmt.Println("  grammar up to date")
	case *write:
		if err := os.MkdirAll(filepath.Dir(grammar), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(grammar, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}
		fmt.Println("  wrote", grammar)
	default:
		fmt.Println("  grammar out of date (pass -w to write)")
	}

	fmt.Println("  language commits since last extension update:")
	commits := pluginsync.LangCommitsSince(pluginsync.LastCommit(dir))
	if len(commits) == 0 {
		fmt.Println("    (none)")
	}
	for _, c := range commits {
		fmt.Println("    " + c)
	}
}
