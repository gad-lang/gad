package langsym

import (
	"strings"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
)

// ParseFile parses source for the language service, by the dialect its name
// says: a `.gadx` template lowered to Gad statements (their positions the
// source's), a `.gadt` template in mixed mode — its code between delim's, the
// default `{%` / `%}` when nil —, plain Gad else. It is tolerant: a file
// being edited answers what parsed of it, with the error (the file nil only
// when nothing could be read).
func ParseFile(name string, data []byte, delim *parser.MixedDelimiter) (*parser.File, *source.File, error) {
	fs := source.NewFileSet()
	sf := fs.AddFileData(name, -1, data)

	if strings.HasSuffix(name, ".gadx") {
		gf, err := gadxparser.NewParser(sf).ParseFile()
		if gf == nil {
			return nil, sf, err
		}
		// the Gadx AST lowered to Gad statements (params, locals,
		// interpolations keep their source positions), as a gad File so the
		// scope resolver works on real identifiers
		return &parser.File{InputFile: sf, Stmts: gadxnode.Convert(gf.Stmts)}, sf, err
	}

	po := &parser.ParserOptions{Mode: parser.ParseComments}
	var so *parser.ScannerOptions
	if strings.HasSuffix(name, ".gadt") {
		// literal text with code islands: mixed mode, so the code resolves
		d := parser.DefaultMixedDelimiter
		if delim != nil {
			d = *delim
		}
		po.Mode |= parser.ParseMixed
		so = &parser.ScannerOptions{Mode: parser.ScanMixed | parser.ScanConfigDisabled, MixedDelimiter: d}
	}
	file, err := parser.NewParserWithOptions(sf, po, so).ParseFileTolerant()
	if file == nil {
		return nil, sf, err
	}
	return file, sf, err
}
