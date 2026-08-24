package main

import (
	"strings"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	gadxparser "github.com/gad-lang/gad/gadx/parser"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
)

// langsymParse parses source into a gad *parser.File for the language service
// (`gad def` / `gad complete`), choosing the front-end by the file name:
//
//   - `.gadx` is parsed with the Gadx front-end and lowered to Gad statements
//     (positions are preserved, so caret offsets map back to the .gadx source);
//   - `.gadt` is parsed in mixed/template mode so the `{% … %}` / `{%= … %}`
//     code islands resolve while the literal text is skipped;
//   - everything else is parsed as plain Gad.
//
// name comes from the PATH argument or --stdin-name; data is the buffer.
func langsymParse(name string, data []byte) (*parser.File, *source.File, error) {
	fs := source.NewFileSet()
	sf := fs.AddFileData(name, -1, data)

	if strings.HasSuffix(name, ".gadx") {
		gf, err := gadxparser.NewParser(sf).ParseFile()
		if err != nil {
			return nil, nil, err
		}
		// Lower the Gadx AST to Gad statements (params, locals, interpolations
		// keep their original source positions), then present them as a gad File
		// so the scope resolver works on real identifiers.
		file := &parser.File{InputFile: sf, Stmts: gadxnode.Convert(gf.Stmts)}
		return file, sf, nil
	}

	po := &parser.ParserOptions{Mode: parser.ParseComments}
	var so *parser.ScannerOptions
	if strings.HasSuffix(name, ".gadt") {
		// A `.gadt` template is mixed source: literal text with `{% … %}` code
		// islands. Parse it in mixed mode so the code resolves (plain-Gad parsing
		// chokes on the leading literal text).
		po.Mode |= parser.ParseMixed
		so = &parser.ScannerOptions{
			Mode:           parser.ScanMixed | parser.ScanConfigDisabled,
			MixedDelimiter: parser.DefaultMixedDelimiter,
		}
	}
	file, err := parser.NewParserWithOptions(sf, po, so).ParseFile()
	if err != nil {
		return nil, nil, err
	}
	return file, sf, nil
}
