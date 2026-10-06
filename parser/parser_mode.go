package parser

// Mode value is a set of flags for parser.
type Mode int

func (b *Mode) Set(flag Mode) *Mode    { *b = *b | flag; return b }
func (b *Mode) Clear(flag Mode) *Mode  { *b = *b &^ flag; return b }
func (b *Mode) Toggle(flag Mode) *Mode { *b = *b ^ flag; return b }
func (b Mode) Has(flag Mode) bool      { return b&flag != 0 }

const (
	// ParseComments parses comments and add them to AST
	ParseComments Mode = 1 << iota
	ParseMixed
	ParseConfigDisabled
	ParseMixedExprAsValue
	ParseFloatAsDecimal
	ParseCharAsString
	// ParseImportMain lets `@import name from "m"` bind the module's main —
	// `@import { main: name } from "m"` —: a template's component (gadx).
	ParseImportMain
)
