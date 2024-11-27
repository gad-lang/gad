package parser

import "github.com/gad-lang/gad/parser/source"

const MainName = "(main)"

func Parse(input, fileName string, opts *ParserOptions, scannerOpts *ScannerOptions) (*File, error) {
	fileSet := source.NewFileSet()
	if fileName == "" {
		fileName = MainName
	}

	b := []byte(input)
	srcFile := fileSet.AddFileData(fileName, -1, b)
	p := NewParserWithOptions(srcFile, opts, scannerOpts)
	return p.ParseFile()
}
