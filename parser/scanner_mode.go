package parser

// ScanMode represents a scanner mode.
type ScanMode uint8

func (b *ScanMode) Set(flag ScanMode) *ScanMode    { *b = *b | flag; return b }
func (b *ScanMode) Clear(flag ScanMode) *ScanMode  { *b = *b &^ flag; return b }
func (b *ScanMode) Toggle(flag ScanMode) *ScanMode { *b = *b ^ flag; return b }
func (b ScanMode) Has(flag ScanMode) bool          { return b&flag != 0 }

// List of scanner modes.
const (
	ScanComments ScanMode = 1 << iota
	DontInsertSemis
	ScanMixed
	ScanConfigDisabled
	ScanMixedExprAsValue
	ScanFloatAsDecimal
	ScanCharAsString
)
