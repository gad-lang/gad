package main

import (
	"encoding/json"

	"github.com/gad-lang/gad/langsym"
	cc "github.com/moisespsena-go/command-context"
)

func init() { registerCommand("symbols", symbolsCommand) }

// symbolsCommand is `gad symbols [PATH]`: it prints a file's structure outline —
// its const/var declarations, functions, classes, mixins, interfaces, enums and
// `met` declarations, with a type's own members nested underneath — as a JSON
// tree. Each node is {name, kind, detail, offset, line, column, children}. Editor
// plugins use it to populate a Structure/Outline view. The `.gad` / `.gadt` /
// `.gadx` dialect is chosen from the file name (--stdin-name for stdin).
func symbolsCommand() *cc.Command {
	var stdinName string
	return &cc.Command{
		Name:  "symbols",
		Usage: "[--stdin-name NAME] [PATH]",
		Description: "Print a file's structure outline as a JSON tree.\n" +
			"\nPATH is a .gad/.gadt/.gadx file or - (stdin). Each node is\n" +
			"{name, kind, offset, line, column, children}; offset is the 0-based byte\n" +
			"offset of the declaration for navigation.",
		New: func(ctx *cc.CommandContext) error {
			ctx.Flags().StringVar(&stdinName, "stdin-name", "",
				"assumed file name for stdin, so its dialect (.gad/.gadt/.gadx) is detected")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			data, name, err := astReadInput(ctx.Args)
			if err != nil {
				return err
			}
			if name == "<stdin>" && stdinName != "" {
				name = stdinName
			}

			// A mid-edit file may not fully parse; langsymParse returns the partial
			// file so the outline still reflects what parsed.
			file, sf, _ := langsymParse(name, data)
			var syms []langsym.OutlineSym
			if file != nil {
				syms = langsym.Outline(file, sf)
			}
			if syms == nil {
				syms = []langsym.OutlineSym{}
			}
			out, _ := json.Marshal(syms)
			_, err = ctx.Out.Write(append(out, '\n'))
			return err
		},
	}
}
