package parser

import (
	"github.com/gad-lang/gad/parser/source"
)

const MainName = "(main)"

// ParseInterpolatedString parses an interpolated string as mixed content using
// { } delimiters and returns the parsed statements. It is used by
// InterpolatedStringLit compilation to process the interpolation expressions.
//
// pos is the position of the string literal's opening delimiter (the quote)
// in the original source. The string content itself begins one byte after
// it, so the parsed file is based at pos+1 to keep interpolation expression
// positions mapped back to their location in the original source.
func ParseInterpolatedString(tmpl string, pos source.Pos) (f *File, err error) {
	return ParseInterpolatedStringMode(tmpl, pos, false)
}

// ParseInterpolatedStringMode is ParseInterpolatedString with an explicit raw
// flag. When raw is true the content is verbatim (a raw interpolated string or
// heredoc): the `\{` / `\}` delimiter escape is disabled, so a backslash stays
// literal and `{` always opens interpolation.
func ParseInterpolatedStringMode(tmpl string, pos source.Pos, raw bool) (f *File, err error) {
	base := int(pos) + 1
	fileSet := source.NewFileSet()
	fileSet.Base = base
	srcFile := fileSet.AddFileData("template", base, []byte(tmpl))
	mode := ScanMixed | ScanConfigDisabled
	if raw {
		mode |= ScanRawMixed
	}
	p := NewParserWithOptions(srcFile, &ParserOptions{
		Mode: ParseMixed,
	}, &ScannerOptions{
		Mode:           mode,
		MixedDelimiter: InterpolatedStringDelimiter,
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
