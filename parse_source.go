package gad

import (
	"io"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
)

// ParseSource parses src — the source file name — with the front-end of its
// dialect, as the compiler does an imported module or an included file: plain
// Gad, a `.gadt` mixed template, or a `.gadx` template lowered to Gad
// statements. The positions of the file's nodes resolve through
// file.InputFile.Set().Position — the line and column in src.
//
// It is what an inspection of the sources a compilation reads works on (see
// importers.FileImporter.Inspect and gadx.Render.Inspect).
func ParseSource(name string, src []byte, kind SourceKind) (*parser.File, error) {
	return parseSourceFile(source.NewFileSet().AppendFileData(name, src), kind, nil)
}

// parseSourceFile parses srcFile with the front-end of kind; trace, when not
// nil, receives the Gad parser's trace.
func parseSourceFile(srcFile *source.File, kind SourceKind, trace io.Writer) (*parser.File, error) {
	if kind == SourceKindGadx {
		return parseGadxFile(srcFile)
	}
	parserOptions := &parser.ParserOptions{Trace: trace}
	var scannerOptions *parser.ScannerOptions
	if kind == SourceKindGadt {
		// A .gadt source is a mixed template: literal text with {% … %} code.
		parserOptions.Mode |= parser.ParseMixed
		scannerOptions = &parser.ScannerOptions{
			Mode:           parser.ScanMixed | parser.ScanConfigDisabled,
			MixedDelimiter: parser.DefaultMixedDelimiter,
		}
	}
	return parser.NewParserWithOptions(srcFile, parserOptions, scannerOptions).ParseFile()
}
