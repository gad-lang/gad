package parser

import (
	"github.com/gad-lang/gad/parser/source"
)

const MainName = "(main)"

// ParseTemplateString parses a template string as mixed content using { }
// delimiters and returns the parsed statements. It is used by TemplateLit
// compilation to process template interpolation expressions.
//
// pos is the position of the string literal's opening delimiter (the quote)
// in the original source. The template content itself begins one byte after
// it, so the parsed file is based at pos+1 to keep interpolation expression
// positions mapped back to their location in the original source.
func ParseTemplateString(tmpl string, pos source.Pos) (f *File, err error) {
	base := int(pos) + 1
	fileSet := source.NewFileSet()
	fileSet.Base = base
	srcFile := fileSet.AddFileData("template", base, []byte(tmpl))
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
