// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package token

import (
	"strconv"
)

var keywords map[string]Token

// Token represents a token.
type Token int

func (tok Token) String() string {
	s := ""

	if 0 <= tok && int(tok) < NumTokens {
		s = tokens[tok]
	} else {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}

	return s
}

func (tok Token) Name() string {
	s := ""

	if 0 <= tok && int(tok) < NumTokens {
		s = tokenNames[tok]
	} else {
		s = "token<" + strconv.Itoa(int(tok)) + ">"
	}

	return s
}

// List of tokens
const (
	Illegal Token = iota
	EOF
	Comment
	ConfigStart
	ConfigEnd
	MixedValueStart
	MixedValueEnd
	MixedCodeStart
	MixedCodeEnd
	MixedText
	GroupLiteralBegin
	Ident
	Int
	Uint
	Float
	Decimal
	Char
	String
	RawString
	RawHeredoc
	Template
	GroupLiteralEnd
	GroupOperatorBegin
	GroupBinaryOperatorBegin
	Add         // +
	Sub         // -
	Mul         // *
	Pow         // **
	Quo         // /
	Rem         // %
	And         // &
	Or          // |
	Xor         // ^
	Shl         // <<
	Shr         // >>
	AndNot      // &^
	LAnd        // &&
	Equal       // ==
	NotEqual    // !=
	Less        // <
	Greater     // >
	LessEq      // <=
	GreaterEq   // >=
	Tilde       // ~
	DoubleTilde // ~~
	TripleTilde // ~~~
	GroupBinaryOperatorEnd
	GroupDefaultOperatorBegin
	Nullich // ??
	LOr     // ||
	GroupDefaultOperatorEnd
	GroupAssignOperatorBegin
	Define // :=
	Assign // =
	GroupSelfAssignOperatorBegin
	AddAssign     // +=
	IncAssign     // ++=
	SubAssign     // -=
	DecAssign     // --=
	MulAssign     // *=
	PowAssign     // **=
	QuoAssign     // /=
	RemAssign     // %=
	AndAssign     // &=
	OrAssign      // |=
	XorAssign     // ^=
	ShlAssign     // <<=
	ShrAssign     // >>=
	AndNotAssign  // &^=
	LOrAssign     // ||=
	NullichAssign // ??=
	GroupSelfAssignOperatorEnd
	GroupAssignOperatorEnd
	GroupUnaryOperatorBegin
	Inc // ++
	Dec // --
	GroupUnaryOperatorEnd
	Lambda          // =>
	Not             // !
	Null            // a == nil || nil == a
	NotNull         // a != nil || nil != a
	Pipe            // .|
	Question        // ?
	NullishSelector // ?.
	GroupOperatorEnd
	LParen    // (
	RParen    // )
	LBrack    // [
	RBrack    // ]
	LBrace    // {
	RBrace    // }
	Semicolon // ;
	Colon     // :
	Comma     // ,
	Period    // .
	GroupKeywordBegin
	Break
	Continue
	Else
	For
	Func
	If
	Return
	True
	False
	Yes
	No
	In
	Nil
	Import
	Embed
	Param
	Global
	Var
	Const
	Try
	Catch
	Finally
	Throw
	Callee
	NamedArgs
	Args
	StdIn
	StdOut
	StdErr
	DotName
	DotFile
	IsModule
	GroupKeywordEnd
)

const NumTokens = int(GroupKeywordEnd)

var tokens = [...]string{
	Illegal:         "ILLEGAL",
	EOF:             "EOF",
	ConfigStart:     "CONFIGSTART",
	ConfigEnd:       "CONFIGEND",
	Comment:         "COMMENT",
	Ident:           "IDENT",
	Int:             "INT",
	Uint:            "UINT",
	Float:           "FLOAT",
	Decimal:         "DECIMAL",
	Char:            "CHAR",
	String:          "STR",
	RawString:       "RAWSTR",
	RawHeredoc:      "RAWHEREDOC",
	Template:        "TMPL",
	Null:            "NULL",
	NotNull:         "NOTNULL",
	StdIn:           "STDIN",
	StdOut:          "STDOUT",
	StdErr:          "STDERR",
	MixedCodeStart:  "MIXEDCODESTART",
	MixedCodeEnd:    "MIXEDCODEEND",
	MixedValueStart: "MIXEDVALUESTART",
	MixedValueEnd:   "MIXEDVALUEEND",
	MixedText:       "MIXEDTEXT",
	Add:             "+",
	Sub:             "-",
	Mul:             "*",
	Pow:             "**",
	Quo:             "/",
	Rem:             "%",
	And:             "&",
	Or:              "|",
	Xor:             "^",
	Shl:             "<<",
	Shr:             ">>",
	AndNot:          "&^",
	AddAssign:       "+=",
	IncAssign:       "++=",
	SubAssign:       "-=",
	DecAssign:       "--=",
	MulAssign:       "*=",
	PowAssign:       "**=",
	QuoAssign:       "/=",
	RemAssign:       "%=",
	AndAssign:       "&=",
	OrAssign:        "|=",
	XorAssign:       "^=",
	ShlAssign:       "<<=",
	ShrAssign:       ">>=",
	AndNotAssign:    "&^=",
	LOrAssign:       "||=",
	NullichAssign:   "??=",
	LAnd:            "&&",
	LOr:             "||",
	Nullich:         "??",
	Inc:             "++",
	Dec:             "--",
	Equal:           "==",
	Less:            "<",
	Greater:         ">",
	Assign:          "=",
	Lambda:          "=>",
	Not:             "!",
	NotEqual:        "!=",
	LessEq:          "<=",
	GreaterEq:       ">=",
	Tilde:           "~",
	DoubleTilde:     "~~",
	TripleTilde:     "~~~",
	Define:          ":=",
	Pipe:            ".|",
	LParen:          "(",
	LBrack:          "[",
	LBrace:          "{",
	Comma:           ",",
	Period:          ".",
	RParen:          ")",
	RBrack:          "]",
	RBrace:          "}",
	Semicolon:       ";",
	Colon:           ":",
	Question:        "?",
	NullishSelector: "?.",
	Break:           "break",
	Continue:        "continue",
	Else:            "else",
	For:             "for",
	Func:            "func",
	If:              "if",
	Return:          "return",
	True:            "true",
	False:           "false",
	Yes:             "yes",
	No:              "no",
	In:              "in",
	Nil:             "nil",
	Import:          "import",
	Embed:           "embed",
	Param:           "param",
	Global:          "global",
	Var:             "var",
	Const:           "const",
	Try:             "try",
	Catch:           "catch",
	Finally:         "finally",
	Throw:           "throw",
	Callee:          "__callee__",
	Args:            "__args__",
	NamedArgs:       "__named_args__",
	DotName:         "__name__",
	DotFile:         "__file__",
	IsModule:        "__is_module__",
}

var tokenNames = [...]string{
	Illegal:                      "Illegal",
	EOF:                          "EOF",
	Comment:                      "Comment",
	ConfigStart:                  "ConfigStart",
	ConfigEnd:                    "ConfigEnd",
	MixedValueStart:              "MixedValueStart",
	MixedValueEnd:                "MixedValueEnd",
	MixedCodeStart:               "MixedCodeStart",
	MixedCodeEnd:                 "MixedCodeEnd",
	MixedText:                    "MixedText",
	GroupLiteralBegin:            "GroupLiteralBegin",
	Ident:                        "Ident",
	Int:                          "Int",
	Uint:                         "Uint",
	Float:                        "Float",
	Decimal:                      "Decimal",
	Char:                         "Char",
	String:                       "String",
	RawString:                    "RawString",
	RawHeredoc:                   "RawHeredoc",
	Template:                     "Template",
	GroupLiteralEnd:              "GroupLiteralEnd",
	GroupOperatorBegin:           "GroupOperatorBegin",
	GroupBinaryOperatorBegin:     "GroupBinaryOperatorBegin",
	Add:                          "Add",
	Sub:                          "Sub",
	Mul:                          "Mul",
	Pow:                          "Pow",
	Quo:                          "Quo",
	Rem:                          "Rem",
	And:                          "And",
	Or:                           "Or",
	Xor:                          "Xor",
	Shl:                          "Shl",
	Shr:                          "Shr",
	AndNot:                       "AndNot",
	LAnd:                         "LAnd",
	Equal:                        "Equal",
	NotEqual:                     "NotEqual",
	Less:                         "Less",
	Greater:                      "Greater",
	LessEq:                       "LessEq",
	GreaterEq:                    "GreaterEq",
	Tilde:                        "Tilde",
	DoubleTilde:                  "DoubleTilde",
	TripleTilde:                  "TripleTilde",
	GroupBinaryOperatorEnd:       "GroupBinaryOperatorEnd",
	GroupDefaultOperatorBegin:    "GroupDefaultOperatorBegin",
	Nullich:                      "Nullich",
	LOr:                          "LOr",
	GroupDefaultOperatorEnd:      "GroupDefaultOperatorEnd",
	GroupAssignOperatorBegin:     "GroupAssignOperatorBegin",
	GroupSelfAssignOperatorBegin: "GroupSelfAssignOperatorBegin",
	Define:                       "Define",
	Assign:                       "Assign",
	AddAssign:                    "AddAssign",
	IncAssign:                    "IncAssign",
	SubAssign:                    "SubAssign",
	DecAssign:                    "DecAssign",
	MulAssign:                    "MulAssign",
	PowAssign:                    "PowAssign",
	QuoAssign:                    "QuoAssign",
	RemAssign:                    "RemAssign",
	AndAssign:                    "AndAssign",
	OrAssign:                     "OrAssign",
	XorAssign:                    "XorAssign",
	ShlAssign:                    "ShlAssign",
	ShrAssign:                    "ShrAssign",
	AndNotAssign:                 "AndNotAssign",
	LOrAssign:                    "LOrAssign",
	NullichAssign:                "NullichAssign",
	GroupSelfAssignOperatorEnd:   "GroupSelfAssignOperatorEnd",
	GroupAssignOperatorEnd:       "GroupAssignOperatorEnd",
	GroupUnaryOperatorBegin:      "GroupUnaryOperatorBegin",
	Inc:                          "Inc",
	Dec:                          "Dec",
	GroupUnaryOperatorEnd:        "GroupUnaryOperatorEnd",
	Lambda:                       "Lambda",
	Not:                          "Not",
	Null:                         "Null",
	NotNull:                      "NotNull",
	Pipe:                         "Pipe",
	Question:                     "Question",
	NullishSelector:              "NullishSelector",
	GroupOperatorEnd:             "GroupOperatorEnd",
	LParen:                       "LParen",
	RParen:                       "RParen",
	LBrack:                       "LBrack",
	RBrack:                       "RBrack",
	LBrace:                       "LBrace",
	RBrace:                       "RBrace",
	Semicolon:                    "Semicolon",
	Colon:                        "Colon",
	Comma:                        "Comma",
	Period:                       "Period",
	GroupKeywordBegin:            "GroupKeywordBegin",
	Break:                        "Break",
	Continue:                     "Continue",
	Else:                         "Else",
	For:                          "For",
	Func:                         "Func",
	If:                           "If",
	Return:                       "Return",
	True:                         "True",
	False:                        "False",
	Yes:                          "Yes",
	No:                           "No",
	In:                           "In",
	Nil:                          "Nil",
	Import:                       "Import",
	Embed:                        "Embed",
	Param:                        "Param",
	Global:                       "Global",
	Var:                          "Var",
	Const:                        "Const",
	Try:                          "Try",
	Catch:                        "Catch",
	Finally:                      "Finally",
	Throw:                        "Throw",
	Callee:                       "Callee",
	NamedArgs:                    "NamedArgs",
	Args:                         "Args",
	StdIn:                        "StdIn",
	StdOut:                       "StdOut",
	StdErr:                       "StdErr",
	DotName:                      "DotName",
	DotFile:                      "DotFile",
	IsModule:                     "IsModule",
	GroupKeywordEnd:              "GroupKeywordEnd",
}

// FromName return a Token from name
func FromName(name string) (t Token) {
	for i, tokenName := range tokenNames {
		if tokenName == name {
			return Token(i)
		}
	}
	return Token(-1)
}

// LowestPrec represents lowest operator precedence.
const LowestPrec = 0

// Precedence returns the precedence for the operator token.
func (tok Token) Precedence() int {
	switch tok {
	case LOr, Nullich:
		return 2
	case LAnd:
		return 3
	case Equal, NotEqual, Less, LessEq, Greater, GreaterEq, Null, NotNull:
		return 4
	case Add, Sub, Or, Xor:
		return 5
	case Mul, Quo, Rem, Shl, Shr, And, AndNot:
		return 6
	case Pow:
		return 7
	case Pipe:
		return 8
	case Tilde, DoubleTilde, TripleTilde:
		return 9
	}
	return LowestPrec
}

// IsLiteral returns true if the token is a literal.
func (tok Token) IsLiteral() bool {
	return GroupLiteralBegin < tok && tok < GroupLiteralEnd
}

// IsOperator returns true if the token is an operator.
func (tok Token) IsOperator() bool {
	return GroupOperatorBegin < tok && tok < GroupOperatorEnd
}

// IsBinaryOperator reports whether token is a binary operator.
func (tok Token) IsBinaryOperator() bool {
	switch tok {
	case Add,
		Sub,
		Mul,
		Pow,
		Quo,
		Rem,
		Less,
		LessEq,
		Greater,
		GreaterEq,
		And,
		Or,
		Xor,
		AndNot,
		Shl,
		Shr,
		Equal,
		NotEqual,
		Tilde,
		DoubleTilde,
		TripleTilde:
		return true
	}
	return false
}

// IsKeyword returns true if the token is a keyword.
func (tok Token) IsKeyword() bool {
	return GroupKeywordBegin < tok && tok < GroupKeywordEnd
}

// Is returns true if then token equals one of args.
func (tok Token) Is(other ...Token) bool {
	for _, o := range other {
		if o == tok {
			return true
		}
	}
	return false
}

func (tok Token) IsBlockStart() bool {
	switch tok {
	case LBrace:
		return true
	}
	return false
}

func (tok Token) IsBlockEnd() bool {
	switch tok {
	case RBrace:
		return true
	}
	return false
}

// Lookup returns corresponding keyword if ident is a keyword.
func Lookup(ident string) Token {
	if tok, isKeyword := keywords[ident]; isKeyword {
		return tok
	}
	return Ident
}

// Unassign convert an assignable Token to your non assignable Token
func Unassign(tok Token) Token {
	switch tok {
	case AddAssign:
		return Add
	case IncAssign:
		return Inc
	case SubAssign:
		return Sub
	case DecAssign:
		return Dec
	case MulAssign:
		return Mul
	case PowAssign:
		return Pow
	case QuoAssign:
		return Quo
	case RemAssign:
		return Rem
	case AndAssign:
		return And
	case OrAssign:
		return Or
	case XorAssign:
		return Xor
	case ShlAssign:
		return Shl
	case ShrAssign:
		return Shr
	case AndNotAssign:
		return AndNot
	case LOrAssign:
		return LOr
	case NullichAssign:
		return Nullich
	default:
		return -1
	}
}

func init() {
	keywords = make(map[string]Token)
	for i := GroupKeywordBegin + 1; i < GroupKeywordEnd; i++ {
		keywords[tokens[i]] = i
	}
}
