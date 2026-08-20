package main

import (
	"encoding/json"
	"strings"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/cmd/internal/pluginsync"
	"github.com/gad-lang/gad/langsym"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
	cc "github.com/moisespsena-go/command-context"
)

func init() { registerCommand("complete", completeCommand) }

// completeCommand is `gad complete --offset N [--prefix P] [PATH]`: it prints, as
// a JSON array, the completion candidates at the caret — the identifiers in scope
// (with their lead doc comments), plus the language keywords, constants and
// global builtin functions (with the builtins' own documentation). Editor plugins
// call it for precise, doc-carrying auto-completion.
func completeCommand() *cc.Command {
	var (
		offset int
		prefix string
	)
	return &cc.Command{
		Name:  "complete",
		Usage: "--offset N [--prefix P] [PATH]",
		Description: "Print completion candidates at a caret offset as a JSON array.\n" +
			"\nPATH is a .gad file or - (stdin); --offset is the 0-based byte offset of\n" +
			"the caret. Each candidate is {label, kind, doc}. --prefix filters the\n" +
			"result to labels with that (case-insensitive) prefix.",
		New: func(ctx *cc.CommandContext) error {
			fs := ctx.Flags()
			fs.IntVar(&offset, "offset", -1, "0-based caret byte offset")
			fs.StringVar(&prefix, "prefix", "", "only labels starting with this prefix (case-insensitive)")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			data, name, err := astReadInput(ctx.Args)
			if err != nil {
				return err
			}

			var items []langsym.Symbol
			// Member access (`x.` / `x[`) is resolved first by runtime
			// introspection: the file need not parse (it is mid-edit), and a
			// member context suppresses the in-scope/keyword candidates.
			if member, ok := memberCompletions(string(data), offset); ok {
				items = member
			} else {
				fs := source.NewFileSet()
				sf := fs.AddFileData(name, -1, data)
				po := &parser.ParserOptions{Mode: parser.ParseComments}
				file, perr := parser.NewParserWithOptions(sf, po, nil).ParseFile()
				if perr != nil {
					return perr
				}
				items = completionItems(file, sf, offset)
			}
			items = filterByPrefix(items, prefix)

			out, _ := json.Marshal(items)
			_, err = ctx.Out.Write(append(out, '\n'))
			return err
		},
	}
}

// completionItems assembles the static candidate set: in-scope identifiers first
// (nearest scope wins), then keywords, constants and global builtins with docs.
func completionItems(file *parser.File, sf *source.File, offset int) []langsym.Symbol {
	var items []langsym.Symbol
	seen := map[string]bool{}
	add := func(s langsym.Symbol) {
		if s.Label == "" || seen[s.Label] {
			return
		}
		seen[s.Label] = true
		items = append(items, s)
	}

	for _, s := range langsym.Completions(file, sf, offset) {
		add(s)
	}

	lang := pluginsync.Extract()
	for _, kw := range lang.Keywords {
		add(langsym.Symbol{Label: kw, Kind: "keyword"})
	}
	for _, c := range lang.Constants {
		add(langsym.Symbol{Label: c, Kind: "constant"})
	}
	for _, b := range lang.Builtins {
		add(langsym.Symbol{Label: b, Kind: "function", Doc: builtinDoc(b)})
	}
	return items
}

// builtinDoc returns the documentation of a global builtin function, if any.
func builtinDoc(name string) string {
	bt, ok := gad.BuiltinsMap[name]
	if !ok {
		return ""
	}
	if bf, ok := gad.BuiltinObjects[bt].(*gad.BuiltinFunction); ok {
		return strings.TrimSpace(bf.Doc())
	}
	return ""
}

func filterByPrefix(items []langsym.Symbol, prefix string) []langsym.Symbol {
	if prefix == "" {
		return items
	}
	p := strings.ToLower(prefix)
	var out []langsym.Symbol
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it.Label), p) {
			out = append(out, it)
		}
	}
	return out
}
