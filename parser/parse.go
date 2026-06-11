package parser

import (
	"github.com/gad-lang/gad/parser/source"
)

const MainName = "(main)"

// ParseTemplateString parses a template string as mixed content using { }
// delimiters and returns the parsed statements. It is used by TemplateLit
// compilation to process template interpolation expressions.
func ParseTemplateString(tmpl string, pos source.Pos) (f *File, err error) {
	fileSet := source.NewFileSet()
	fileSet.Base = int(pos)
	srcFile := fileSet.AddFileData("template", int(pos), []byte(tmpl))
	p := NewParserWithOptions(srcFile, &ParserOptions{
		Mode: ParseMixed,
	}, &ScannerOptions{
		Mode:           ScanMixed | ScanConfigDisabled,
		MixedDelimiter: TemplateStrDelimiter,
	})
	return p.ParseFile()
}

func NewSingleParser(input, fileName string, opts *ParserOptions, scannerOpts *ScannerOptions) *Parser {
	fileSet := source.NewFileSet()
	if fileName == "" {
		fileName = MainName
	}

	b := []byte(input)
	srcFile := fileSet.AddFileData(fileName, -1, b)
	return NewParserWithOptions(srcFile, opts, scannerOpts)
}

func Parse(input, fileName string, opts *ParserOptions, scannerOpts *ScannerOptions) (*File, error) {
	return NewSingleParser(input, fileName, opts, scannerOpts).ParseFile()
}
