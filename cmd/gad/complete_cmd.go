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
		offset    int
		prefix    string
		stdinName string
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
			fs.StringVar(&stdinName, "stdin-name", "",
				"assumed file name for stdin, so its dialect (.gad/.gadx) is detected")
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

			var items []langsym.Symbol
			// Member access (`x.` / `x[`) is resolved first by runtime
			// introspection: the file need not parse (it is mid-edit), and a
			// member context suppresses the in-scope/keyword candidates.
			if member, ok := memberCompletions(string(data), offset); ok {
				items = member
			} else {
				file, sf, perr := langsymParse(name, data)
				if file == nil {
					// The buffer is mid-edit and did not parse — often an empty slot
					// where an expression is expected (`for i, u in ‸ begin`, `x := ‸`),
					// which is exactly where completion matters most. Splice a sentinel
					// identifier at the caret so the expression becomes syntactically
					// valid, then complete at that position.
					if patched, ok := spliceIdent(data, offset); ok {
						if f2, sf2, _ := langsymParse(name, patched); f2 != nil {
							file, sf = f2, sf2
						}
					}
				}
				if file != nil {
					items = completionItems(file, sf, offset)
				} else {
					// Still unparseable: offer the static candidates (keywords,
					// constants, builtins) rather than nothing.
					items = staticCompletions()
					if len(items) == 0 {
						return perr
					}
				}
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
	for _, s := range staticCompletions() {
		add(s)
	}
	return items
}

// staticCompletions is the scope-independent candidate set: the language
// keywords, constants and global builtin functions (with docs). It is the
// fallback when the buffer cannot be parsed at all, so a code context still
// offers something instead of "no suggestions".
func staticCompletions() []langsym.Symbol {
	lang := pluginsync.Extract()
	items := make([]langsym.Symbol, 0, len(lang.Keywords)+len(lang.Constants)+len(lang.Builtins))
	for _, kw := range lang.Keywords {
		items = append(items, langsym.Symbol{Label: kw, Kind: "keyword"})
	}
	for _, c := range lang.Constants {
		items = append(items, langsym.Symbol{Label: c, Kind: "constant"})
	}
	for _, b := range lang.Builtins {
		items = append(items, langsym.Symbol{Label: b, Kind: "function", Doc: builtinDoc(b)})
	}
	return items
}

// spliceIdent inserts a sentinel identifier at the caret byte offset, returning
// the patched buffer. It turns an empty expression slot (where the parser fails
// because a term is expected, e.g. `for x in ‸ begin`) into valid source that
// parses, so completions can resolve the scope at the caret. The sentinel is an
// unlikely name that will not shadow real symbols. Reports false if the offset
// is out of range.
func spliceIdent(data []byte, offset int) ([]byte, bool) {
	if offset < 0 || offset > len(data) {
		return nil, false
	}
	const sentinel = "gadCompletionCaret"
	out := make([]byte, 0, len(data)+len(sentinel))
	out = append(out, data[:offset]...)
	out = append(out, sentinel...)
	out = append(out, data[offset:]...)
	return out, true
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
