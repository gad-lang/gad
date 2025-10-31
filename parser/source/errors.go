package source

import "errors"

// ScannerErrorHandler is an error handler for the scanner.
type ScannerErrorHandler func(pos SourceFilePos, msg string)

var (
	ErrIllegalMinimalLineNumber = errors.New("illegal line number (line numbering starts at 1)")
	ErrIllegalLineNumber        = errors.New("illegal line number")
	ErrIllegalFileOffset        = errors.New("illegal file offset")
	ErrIllegalPosition          = errors.New("illegal position")
)
