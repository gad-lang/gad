package main

import (
	"encoding/json"

	"github.com/gad-lang/gad/langsym"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
	cc "github.com/moisespsena-go/command-context"
)

func init() { registerCommand("def", defCommand) }

// defCommand is `gad def --offset N [PATH]`: it prints, as JSON, the declaration
// location (offset/line/column) of the identifier at the caret, or null when
// none is found. Scope-aware (blocks, functions, shadowing). Editor plugins call
// it for precise go-to-declaration.
func defCommand() *cc.Command {
	var offset int
	return &cc.Command{
		Name:  "def",
		Usage: "--offset N [PATH]",
		Description: "Print the declaration location of the identifier at a caret offset.\n" +
			"\nPATH is a .gad file or - (stdin); --offset is the 0-based byte offset of\n" +
			"the caret. Output is JSON {offset, line, column} or null.",
		New: func(ctx *cc.CommandContext) error {
			ctx.Flags().IntVar(&offset, "offset", -1, "0-based caret byte offset")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			data, name, err := astReadInput(ctx.Args)
			if err != nil {
				return err
			}
			fs := source.NewFileSet()
			sf := fs.AddFileData(name, -1, data)
			po := &parser.ParserOptions{Mode: parser.ParseComments}
			file, err := parser.NewParserWithOptions(sf, po, nil).ParseFile()
			if err != nil {
				return err
			}

			off, ok := langsym.Definition(file, sf, offset)
			if !ok {
				_, err = ctx.Out.Write([]byte("null\n"))
				return err
			}
			fp := sf.SafePosition(source.Pos(sf.Base + off))
			out, _ := json.Marshal(map[string]int{"offset": off, "line": fp.Line, "column": fp.Column})
			_, err = ctx.Out.Write(append(out, '\n'))
			return err
		},
	}
}
