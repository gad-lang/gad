package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gad-lang/gad/astio"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	cc "github.com/moisespsena-go/command-context"
)

func init() { registerCommand("ast", astCommand) }

// astCommand exposes the Gad AST as JSON/YAML (and back). `gad ast file.gad`
// prints the parsed AST; `gad ast --render < ast.json` reconstructs it and prints
// the Gad source. The serialization (github.com/gad-lang/gad/astio) is public and
// works for the built-in, gadx and user-defined node types.
func astCommand() *cc.Command {
	var (
		asYAML   bool
		doRender bool
		outPath  string
	)
	return &cc.Command{
		Name:  "ast",
		Usage: "[flags] [PATH]",
		Description: "Print a Gad file's AST as JSON (default) or YAML.\n" +
			"\nPATH is a .gad file or - (stdin). With --render the input is an AST\n" +
			"(JSON, or YAML with --yaml) and the Gad source is printed instead.",
		New: func(ctx *cc.CommandContext) error {
			fs := ctx.Flags()
			fs.BoolVar(&asYAML, "yaml", false, "use YAML instead of JSON")
			fs.BoolVar(&doRender, "render", false, "read an AST on the input and print Gad source")
			fs.StringVar(&outPath, "out", "", "write to this file instead of stdout")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			astio.Register((*parser.File)(nil)) // the file node round-trips both ways
			data, name, err := astReadInput(ctx.Args)
			if err != nil {
				return err
			}

			var result []byte
			if doRender {
				n, err := astUnmarshal(data, asYAML)
				if err != nil {
					return err
				}
				c, ok := n.(node.Coder)
				if !ok {
					return fmt.Errorf("ast: input is not a renderable node")
				}
				result = []byte(node.Code(c) + "\n")
			} else {
				file, err := astParse(name, data)
				if err != nil {
					return err
				}
				if asYAML {
					result, err = astio.MarshalYAML(file)
				} else {
					result, err = astio.MarshalJSON(file)
					result = append(result, '\n')
				}
				if err != nil {
					return err
				}
			}

			if outPath == "" || outPath == "-" {
				_, err = ctx.Out.Write(result)
				return err
			}
			return os.WriteFile(outPath, result, 0o644)
		},
	}
}

// astReadInput returns the input bytes and a display name from the first arg (a
// file, or - / none for stdin).
func astReadInput(args []string) (data []byte, name string, err error) {
	if len(args) > 0 && args[0] != "-" {
		data, err = os.ReadFile(args[0])
		return data, args[0], err
	}
	data, err = io.ReadAll(os.Stdin)
	return data, "<stdin>", err
}

// astParse parses Gad source into a File (with comments, so they survive export).
func astParse(name string, src []byte) (*parser.File, error) {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AddFileData(name, -1, src)
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	return parser.NewParserWithOptions(srcFile, po, nil).ParseFile()
}

// astUnmarshal reconstructs a node from JSON (default) or YAML.
func astUnmarshal(data []byte, asYAML bool) (node.Node, error) {
	var (
		n   ast.Node
		err error
	)
	if asYAML {
		n, err = astio.UnmarshalYAML(data)
	} else {
		n, err = astio.UnmarshalJSON(data)
	}
	if err != nil {
		return nil, err
	}
	nn, _ := n.(node.Node)
	return nn, nil
}
