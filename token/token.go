// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package token

import "strconv"

var keywords map[string]Token

// Token represents a token.
type Token int

// List of tokens
const (
	Illegal Token = iota
	EOF
	Comment
	ConfigStart
	ConfigEnd
	ToTextBegin
	ToTextEnd
	CodeBegin
	CodeEnd
	LiteralBegin_
	Ident
	Int
	Uint
	Float
	Decimal
	Char
	String
	RawString
	LiteralEnd_
	OperatorBegin_
	Add             // +
	Sub             // -
	Mul             // *
	Quo             // /
	Rem             // %
	And             // &
	Or              // |
	Xor             // ^
	Shl             // <<
	Shr             // >>
	AndNot          // &^
	AddAssign       // +=
	SubAssign       // -=
	MulAssign       // *=
	QuoAssign       // /=
	RemAssign       // %=
	AndAssign       // &=
	OrAssign        // |=
	XorAssign       // ^=
	ShlAssign       // <<=
	ShrAssign       // >>=
	AndNotAssign    // &^=
	LOrAssign       // ||=
	NullichAssign   // ??=
	NullichCoalesce // ??
	LAnd            // &&
	LOr             // ||
	Inc             // ++
	Dec             // --
	Equal           // ==
	Less            // <
	Greater         // >
	Assign          // =
	Lambda          // =>
	Not             // !
	NotEqual        // !=
	Null            // a == nil || nil == a
	NotNull         // a != nil || nil != a
	LessEq          // <=
	GreaterEq       // >=
	Define          // :=
	Pipe            // .|
	LParen          // (
	RParen          // )
	LBrack          // [
	RBrack          // ]
	Comma           // ,
	Period          // .
	RBrace          // }
	LBrace          // {
	Semicolon       // ;
	Colon           // :
	Question        // ?
	NullishSelector // ?.
	OperatorEnd_
	KeyworkBegin_
	Then
	Do
	Begin
	End
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
	KeywordEnd_
)

var tokens = [...]string{
	Illegal:         "ILLEGAL",
	EOF:             "EOF",
	ConfigStart:     "CONFIG",
	Comment:         "COMMENT",
	Ident:           "IDENT",
	Int:             "INT",
	Uint:            "UINT",
	Float:           "FLOAT",
	Decimal:         "DECIMAL",
	Char:            "CHAR",
	String:          "STRING",
	RawString:       "RAWSTRING",
	Null:            "NULL",
	NotNull:         "NOTNULL",
	StdIn:           "STDIN",
	StdOut:          "STDOUT",
	StdErr:          "STDERR",
	CodeBegin:       "CODEBEGIN",
	CodeEnd:         "CODEEND",
	ToTextBegin:     "TOTEXTBEGIN",
	ToTextEnd:       "TOTEXTEND",
	Add:             "+",
	Sub:             "-",
	Mul:             "*",
	Quo:             "/",
	Rem:             "%",
	And:             "&",
	Or:              "|",
	Xor:             "^",
	Shl:             "<<",
	Shr:             ">>",
	AndNot:          "&^",
	AddAssign:       "+=",
	SubAssign:       "-=",
	MulAssign:       "*=",
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
	NullichCoalesce: "??",
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
	Param:           "param",
	Global:          "global",
	Var:             "var",
	Const:           "const",
	Try:             "try",
	Catch:           "catch",
	Finally:         "finally",
	Throw:           "throw",
	Do:              "do",
	Then:            "then",
	Begin:           "begin",
	End:             "end",
	Callee:          "__callee__",
	Args:            "__args__",
	NamedArgs:       "__named_args__",
}

func (tok Token) String() string {
	s := ""

	if 0 <= tok && tok < Token(len(tokens)) {
		s = tokens[tok]
	}

	if s == "" {
		s = "token(" + strconv.Itoa(int(tok)) + ")"
	}

	return s
}

// LowestPrec represents lowest operator precedence.
const LowestPrec = 0

// Precedence returns the precedence for the operator token.
func (tok Token) Precedence() int {
	switch tok {
	case Pipe:
		return 1
	case LOr, NullichCoalesce:
		return 2
	case LAnd:
		return 3
	case Equal, NotEqual, Less, LessEq, Greater, GreaterEq, Null, NotNull:
		return 4
	case Add, Sub, Or, Xor:
		return 5
	case Mul, Quo, Rem, Shl, Shr, And, AndNot:
		return 6
	}
	return LowestPrec
}

// IsLiteral returns true if the token is a literal.
func (tok Token) IsLiteral() bool {
	return LiteralBegin_ < tok && tok < LiteralEnd_
}

// IsOperator returns true if the token is an operator.
func (tok Token) IsOperator() bool {
	return OperatorBegin_ < tok && tok < OperatorEnd_
}

// IsBinaryOperator reports whether token is a binary operator.
func (tok Token) IsBinaryOperator() bool {
	switch tok {
	case Add,
		Sub,
		Mul,
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
		NotEqual:
		return true
	}
	return false
}

// IsKeyword returns true if the token is a keyword.
func (tok Token) IsKeyword() bool {
	return KeyworkBegin_ < tok && tok < KeywordEnd_
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
	case LBrace, Then, Begin, Do:
		return true
	}
	return false
}

func (tok Token) IsBlockEnd() bool {
	switch tok {
	case RBrace, End:
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

func init() {
	keywords = make(map[string]Token)
	for i := KeyworkBegin_ + 1; i < KeywordEnd_; i++ {
		keywords[tokens[i]] = i
	}
}
