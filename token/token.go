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
	MixedValueStart
	MixedValueEnd
	MixedCodeStart
	MixedCodeEnd
	LiteralBegin_
	MixedText
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
	LiteralEnd_
	OperatorBegin_
	BinaryOperatorBegin_
	Add         // +
	Sub         // -
	Mul         // *
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
	BinaryOperatorEnd_
	DefaultOperatorsBegin_
	NullichCoalesce // ??
	LOr             // ||
	DefaultOperatorsEnd_
	AssignOperatorBegin_
	Define        // :=
	Assign        // =
	AddAssign     // +=
	SubAssign     // -=
	MulAssign     // *=
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
	AssignOperatorEnd_
	UnaryOperatorBegin_
	Inc // ++
	Dec // --
	UnaryOperatorEnd_
	Lambda          // =>
	Not             // !
	Null            // a == nil || nil == a
	NotNull         // a != nil || nil != a
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
	DotName
	DotFile
	IsModule
	KeywordEnd_
)

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
	case Pipe:
		return 7
	case Tilde, DoubleTilde, TripleTilde:
		return 8
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
		NotEqual,
		Tilde, DoubleTilde, TripleTilde:
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

func init() {
	keywords = make(map[string]Token)
	for i := KeyworkBegin_ + 1; i < KeywordEnd_; i++ {
		keywords[tokens[i]] = i
	}
}
