package token

import "github.com/gad-lang/gad/token"

// Giom-specific token kinds, mapped to token.Token values starting from 1000
// to avoid collision with gad's built-in tokens (max is token.NumTokens = 142).
const (
	EOF token.Token = iota + 1000
	Doctype
	Comment
	Indent
	Outdent
	Blank
	Id
	ClassName
	Tag
	Text
	Attribute
	If
	ElseIf
	Wrap
	Else
	For
	Assignment
	Code
	ImportModule
	Func
	Slot
	SlotPass
	Comp
	CompCall
	Match
	Case
	Export
	Global
	Var
	Const
	Enum
	Html
	tokMax
)

var tokNames = [...]string{
	EOF:          "EOF",
	Doctype:      "DOCTYPE",
	Comment:      "COMENT",
	Indent:       "INDENT",
	Outdent:      "OUTDENT",
	Blank:        "BLANK",
	Id:           "ID",
	ClassName:    "CLASS_NAME",
	Tag:          "TAG",
	Text:         "TEXT",
	Attribute:    "ATTRIBUTE",
	If:           "IF",
	ElseIf:       "ELSE_IF",
	Wrap:         "WRAP",
	Else:         "ELSE",
	For:          "FOR",
	Assignment:   "ASSIGNMENT",
	Code:         "CODE",
	ImportModule: "IMPORT_MODULE",
	Func:         "FUNC",
	Slot:         "SLOT",
	SlotPass:     "SLOT_PASS",
	Comp:         "COMP",
	CompCall:     "COMP_CALL",
	Match:        "MATCH",
	Case:         "CASE",
	Export:       "EXPORT",
	Global:       "GLOBAL",
	Var:          "VAR",
	Const:        "CONST",
	Enum:         "ENUM",
	Html:         "HTML",
}

// String returns a human-readable name for a giom token.
func String(tok token.Token) string {
	if tok >= EOF && tok < tokMax {
		return tokNames[tok]
	}
	return tok.String()
}

// IsGiomToken reports whether the token is a giom-specific token.
func IsGiomToken(tok token.Token) bool {
	return tok >= EOF && tok < tokMax
}

// Scanner states.
const (
	ScnNewLine = iota
	ScnLine
	ScnEOF
)
