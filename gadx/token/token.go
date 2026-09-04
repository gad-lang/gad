package token

import "github.com/gad-lang/gad/token"

// Gadx-specific token kinds, mapped to token.Token values starting from 1000
// to avoid collision with gad's built-in tokens (max is token.NumTokens = 142).
const (
	EOF token.Token = iota + 1000
	Doctype
	Comment
	Indent
	Outdent
	Blank
	ID
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
	Param
	Var
	Const
	Enum
	HTML
	TextBlock // @text — raw literal-text block
	RawText   // @raw_text — verbatim block, `#{= … }#` interpolation
	Para      // @p — paragraph block
	Md        // @md — Markdown block
	Test      // @test — test block (lowers to Gad `test NAME { … }`)
	Call      // `! recv.method arg1 arg2` — fluent call statement
	Repeat    // `(N)` right after a tag head — the tag is written N times
	tokMax
)

var tokNames = [...]string{
	EOF:          "EOF",
	Doctype:      "DOCTYPE",
	Comment:      "COMENT",
	Indent:       "INDENT",
	Outdent:      "OUTDENT",
	Blank:        "BLANK",
	ID:           "ID",
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
	Param:        "PARAM",
	Var:          "VAR",
	Const:        "CONST",
	Enum:         "ENUM",
	HTML:         "HTML",
	TextBlock:    "TEXT_BLOCK",
	RawText:      "RAW_TEXT",
	Para:         "PARA",
	Md:           "MD",
	Test:         "TEST",
	Call:         "CALL",
	Repeat:       "REPEAT",
}

// String returns a human-readable name for a gadx token.
func String(tok token.Token) string {
	if tok >= EOF && tok < tokMax {
		return tokNames[tok]
	}
	return tok.String()
}

// IsGadxToken reports whether the token is a gadx-specific token.
func IsGadxToken(tok token.Token) bool {
	return tok >= EOF && tok < tokMax
}

// Scanner states.
const (
	ScnNewLine = iota
	ScnLine
	ScnEOF
)
