package source

// ScannerErrorHandler is an error handler for the scanner.
type ScannerErrorHandler func(pos SourceFilePos, msg string)
