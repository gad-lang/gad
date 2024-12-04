package parser

import "github.com/gad-lang/gad/parser/source"

const MainName = "(main)"

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
