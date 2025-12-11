package node

import (
	"fmt"
	"strconv"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

func SExpr(x Expr) *ExprStmt {
	return &ExprStmt{Expr: x}
}

func SDecl(decl Decl) *DeclStmt {
	return &DeclStmt{Decl: decl}
}

func NewGenDecl(
	tok token.Token,
	tokPos, lparen, rparen source.Pos,
	specs ...Spec,
) Decl {
	return &GenDecl{
		Tok:    tok,
		TokPos: tokPos,
		Lparen: lparen,
		Specs:  specs,
		Rparen: rparen,
	}
}

func NewParamSpec(variadic bool, ident *TypedIdentExpr) Spec {
	return &ParamSpec{
		Ident:    ident,
		Variadic: variadic,
	}
}

func NewNamedParamSpec(ident *TypedIdentExpr, value Expr) Spec {
	return &NamedParamSpec{
		Ident: ident,
		Value: value,
	}
}

func NewValueSpec(idents []*IdentExpr, values []Expr) Spec {
	return &ValueSpec{
		Idents: idents,
		Values: values,
	}
}

func SAssign(
	lhs, rhs []Expr,
	token token.Token,
	pos source.Pos,
) *AssignStmt {
	return &AssignStmt{LHS: lhs, RHS: rhs, Token: token, TokenPos: pos}
}

func SReturn(pos source.Pos, result Expr) *ReturnStmt {
	return &ReturnStmt{Return: Return{Result: result, ReturnPos: pos}}
}

func EReturnExpr(pos source.Pos, result Expr) *ReturnExpr {
	return &ReturnExpr{Return: Return{Result: result, ReturnPos: pos}}
}

func SFor(
	init Stmt,
	cond Expr,
	post Stmt,
	body *BlockStmt,
	pos source.Pos,
) *ForStmt {
	return &ForStmt{
		Cond: cond, Init: init, Post: post, Body: body, ForPos: pos,
	}
}

func SForIn(
	key, value *IdentExpr,
	seq Expr,
	body *BlockStmt,
	pos source.Pos,
	elseb ...*BlockStmt,
) *ForInStmt {
	f := &ForInStmt{
		Key: key, Value: value, Iterable: seq, Body: body, ForPos: pos,
	}
	for _, f.Else = range elseb {
	}
	return f
}

func SBreak(pos source.Pos) *BranchStmt {
	return &BranchStmt{
		Token:    token.Break,
		TokenPos: pos,
	}
}

func SContinue(pos source.Pos) *BranchStmt {
	return &BranchStmt{
		Token:    token.Continue,
		TokenPos: pos,
	}
}

func SIf(
	init Stmt,
	cond Expr,
	body *BlockStmt,
	elseStmt Stmt,
	pos source.Pos,
) *IfStmt {
	return &IfStmt{
		Init: init, Cond: cond, Body: body, Else: elseStmt, IfPos: pos,
	}
}

func STry(
	tryPos source.Pos,
	body *BlockStmt,
	catch *CatchStmt,
	finally *FinallyStmt,
) *TryStmt {
	return &TryStmt{TryPos: tryPos, Body: body, Catch: catch, Finally: finally}
}

func SCatch(
	catchPos source.Pos,
	ident *IdentExpr,
	body *BlockStmt,
) *CatchStmt {
	return &CatchStmt{CatchPos: catchPos, Ident: ident, Body: body}
}

func SFinally(
	finallyPos source.Pos,
	body *BlockStmt,
) *FinallyStmt {
	return &FinallyStmt{FinallyPos: finallyPos, Body: body}
}

func SThrow(
	throwPos source.Pos,
	expr Expr,
) *ThrowStmt {
	return &ThrowStmt{ThrowPos: throwPos, Expr: expr}
}

func SIncDec(
	expr Expr,
	tok token.Token,
	pos source.Pos,
) *IncDecStmt {
	return &IncDecStmt{Expr: expr, Token: tok, TokenPos: pos}
}

func NewFuncType(pos, lparen, rparen source.Pos, v ...any) *FuncType {
	f := &FuncType{Params: FuncParams{LParen: lparen, RParen: rparen}, FuncPos: pos}
	for _, v := range v {
		switch t := v.(type) {
		case ArgsList:
			f.Params.Args = t
		case NamedArgsList:
			f.Params.NamedArgs = t
		case *IdentExpr:
			f.Ident = t
		}
	}
	return f
}

func ProxyFuncType() *FuncType {
	return &FuncType{
		Params: FuncParams{
			Args: ArgsList{
				Var: &TypedIdentExpr{
					Ident: &IdentExpr{
						Name: "args",
					},
				},
			},
			NamedArgs: NamedArgsList{
				Var: &TypedIdentExpr{
					Ident: &IdentExpr{
						Name: "kwargs",
					},
				},
			},
		},
	}
}

func Args(vari *TypedIdentExpr, names ...Expr) ArgsList {
	l := ArgsList{Var: vari}
	for _, name := range names {
		switch t := name.(type) {
		case *IdentExpr:
			l.Values = append(l.Values, ETypedIdent(t))
		case *TypedIdentExpr:
			l.Values = append(l.Values, t)
		}
	}
	return l
}

func NamedArgs(vari *TypedIdentExpr, names []*TypedIdentExpr, values []Expr) NamedArgsList {
	return NamedArgsList{Names: names, Var: vari, Values: values}
}

func SBlock(lbrace, rbrace source.Pos, list ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: list, LBrace: Lit("{", lbrace), RBrace: Lit("}", rbrace)}
}

func SBlockLit(lbrace, rbrace ast.Literal, list ...Stmt) *BlockStmt {
	return &BlockStmt{Stmts: list, LBrace: lbrace, RBrace: rbrace}
}

func EBlock(lbrace, rbrace source.Pos, list ...Stmt) *BlockExpr {
	return &BlockExpr{BlockStmt: SBlock(lbrace, rbrace, list...)}
}

func EIdent(name string, pos source.Pos) *IdentExpr {
	return &IdentExpr{Name: name, NamePos: pos}
}

func EEmptyIdent(pos source.Pos) *IdentExpr {
	return &IdentExpr{Name: "_", NamePos: pos, Empty: true}
}

func ETypedIdent(ident *IdentExpr, typ ...*IdentExpr) *TypedIdentExpr {
	return &TypedIdentExpr{Ident: ident, Type: typ}
}

func SMixedText(pos source.Pos, vlit string, flags ...MixedTextStmtFlag) *MixedTextStmt {
	var f MixedTextStmtFlag
	for _, f = range flags {
	}
	return &MixedTextStmt{Lit: Lit(vlit, pos), Flags: f}
}
func SCodeBegin(lit ast.Literal, removeSpace bool) *CodeBeginStmt {
	return &CodeBeginStmt{Lit: lit, RemoveSpace: removeSpace}
}

func SCodeEnd(lit ast.Literal, removeSpace bool) *CodeEndStmt {
	return &CodeEndStmt{Lit: lit, RemoveSpace: removeSpace}
}

func SMixedValue(start, end ast.Literal, expr Expr) *MixedValueStmt {
	return &MixedValueStmt{Expr: expr, StartLit: start, EndLit: end}
}

func Lit(value string, pos source.Pos) ast.Literal {
	return ast.Literal{Value: value, Pos: pos}
}

func KV(key Expr, value ...Expr) *KeyValueLit {
	kv := &KeyValueLit{Key: key}
	for _, expr := range value {
		kv.Value = expr
	}
	return kv
}

func KVp(key Expr, value ...Expr) *KeyValuePairLit {
	kv := &KeyValuePairLit{Key: key}
	for _, expr := range value {
		kv.Value = expr
	}
	return kv
}

func SConfig(start source.Pos, opts ...*KeyValuePairLit) *ConfigStmt {
	c := &ConfigStmt{ConfigPos: start, Elements: opts}
	c.ParseElements()
	return c
}

func ENullish(
	sel,
	expr Expr,
) *NullishSelectorExpr {
	return &NullishSelectorExpr{Expr: sel, Sel: expr}
}

func EBinary(
	x, y Expr,
	op token.Token,
	pos source.Pos,
) *BinaryExpr {
	return &BinaryExpr{LHS: x, RHS: y, Token: op, TokenPos: pos}
}

func ECond(
	cond, trueExpr, falseExpr Expr,
	questionPos, colonPos source.Pos,
) *CondExpr {
	return &CondExpr{
		Cond: cond, True: trueExpr, False: falseExpr,
		QuestionPos: questionPos, ColonPos: colonPos,
	}
}

func EUnary(x Expr, op token.Token, pos source.Pos) *UnaryExpr {
	return &UnaryExpr{Expr: x, Token: op, TokenPos: pos}
}

func EImport(moduleName string, pos source.Pos) *ImportExpr {
	return &ImportExpr{ModuleName: moduleName, Token: token.Import, TokenPos: pos}
}

func EEmbed(path string, pos source.Pos) *EmbedExpr {
	return &EmbedExpr{Path: path, Token: token.Embed, TokenPos: pos}
}

func Int(value int64, pos source.Pos) *IntLit {
	return &IntLit{Value: value, ValuePos: pos}
}

func Float(value float64, pos source.Pos) *FloatLit {
	return &FloatLit{Value: value, ValuePos: pos}
}

func Decimal(value string, pos source.Pos) *DecimalLit {
	v, _ := decimal.NewFromString(value)
	return &DecimalLit{Value: v, ValuePos: pos}
}

func String(value string, pos source.Pos) *StringLit {
	return &StringLit{Literal: strconv.Quote(value), ValuePos: pos}
}

func RawString(value string, pos source.Pos) *RawStringLit {
	return &RawStringLit{Literal: value, LiteralPos: pos, Quoted: value[0] == '`'}
}

func RawHeredoc(value string, pos source.Pos) *RawHeredocLit {
	return &RawHeredocLit{Literal: value, LiteralPos: pos}
}

func Char(value rune, pos source.Pos) *CharLit {
	return &CharLit{
		Value: value, ValuePos: pos, Literal: fmt.Sprintf("'%c'", value),
	}
}

func Bool(value bool, pos source.Pos) *BoolLit {
	return &BoolLit{Value: value, ValuePos: pos}
}

func Flag(value bool, pos source.Pos) *FlagLit {
	return &FlagLit{Value: value, ValuePos: pos}
}

func Array(lbracket, rbracket source.Pos, list ...Expr) *ArrayExpr {
	return &ArrayExpr{LBrack: lbracket, RBrack: rbracket, Elements: list}
}

func CaleeKW(pos source.Pos) *CalleeKeywordExpr {
	return &CalleeKeywordExpr{TokenPos: pos, Literal: token.Callee.String()}
}

func ArgsKW(pos source.Pos) *ArgsKeywordExpr {
	return &ArgsKeywordExpr{TokenPos: pos, Literal: token.Args.String()}
}

func NamedArgsKW(pos source.Pos) *NamedArgsKeywordExpr {
	return &NamedArgsKeywordExpr{TokenPos: pos, Literal: token.NamedArgs.String()}
}

func MapElement(
	key string,
	keyPos source.Pos,
	colonPos source.Pos,
	value Expr,
) *DictElementLit {
	return &DictElementLit{
		Key: key, KeyPos: keyPos, ColonPos: colonPos, Value: value,
	}
}

func EDict(lbrace, rbrace source.Pos, list ...*DictElementLit) *DictExpr {
	return &DictExpr{LBrace: lbrace, RBrace: rbrace, Elements: list}
}

func EDictElementClosure(c *ClosureExpr) *DictElementFuncExpr {
	return &DictElementFuncExpr{Expr: c}
}

func EDictElementFunc(f *FuncExpr) *DictElementFuncExpr {
	return &DictElementFuncExpr{Expr: f}
}

func EFunc(funcType *FuncType, body *BlockStmt) *FuncExpr {
	return &FuncExpr{Type: funcType, Body: body}
}

func EFuncBodyE(funcType *FuncType, body Expr) *FuncExpr {
	return &FuncExpr{Type: funcType, BodyExpr: body}
}

func EClosure(funcType *FuncType, body Expr) *ClosureExpr {
	return &ClosureExpr{Body: body}
}

func EParen(x Expr, lparen, rparen source.Pos) *ParenExpr {
	return &ParenExpr{Expr: x, LParen: lparen, RParen: rparen}
}

func ECall(
	f Expr,
	lparen, rparen source.Pos,
	args ...any,
) (ce *CallExpr) {
	ce = &CallExpr{Func: f, CallArgs: CallArgs{LParen: lparen, RParen: rparen}}
	for _, v := range args {
		switch t := v.(type) {
		case CallExprArgs:
			ce.Args = t
		case CallExprNamedArgs:
			ce.NamedArgs = t
		}
	}
	return ce
}

func ECallProxy(efunc Expr) *CallExpr {
	return ECall(efunc, 0, 0,
		Args(ETypedIdent(EIdent("args", 0))),
		NamedArgs(ETypedIdent(EIdent("kwargs", 0)), nil, nil),
	)
}

func ArgVar(pos source.Pos, value Expr) *ArgVarLit {
	return &ArgVarLit{TokenPos: pos, Value: value}
}

func NamedArgVar(pos source.Pos, value Expr) *NamedArgVarLit {
	return &NamedArgVarLit{TokenPos: pos, Value: value}
}

func NewCallExprArgs(
	argVar *ArgVarLit,
	args ...Expr,
) (ce CallExprArgs) {
	return CallExprArgs{Var: argVar, Values: args}
}

func NewCallExprNamedArgs(
	argVar *NamedArgVarLit,
	names []NamedArgExpr, values []Expr,
) (ce CallExprNamedArgs) {
	return CallExprNamedArgs{Var: argVar, Names: names, Values: values}
}

func EIndex(
	x, index Expr,
	lbrack, rbrack source.Pos,
) *IndexExpr {
	return &IndexExpr{
		Expr: x, Index: index, LBrack: lbrack, RBrack: rbrack,
	}
}

func ESlice(
	x, low, high Expr,
	lbrack, rbrack source.Pos,
) *SliceExpr {
	return &SliceExpr{
		Expr: x, Low: low, High: high, LBrack: lbrack, RBrack: rbrack,
	}
}

func ESelector(x, sel Expr) *SelectorExpr {
	return &SelectorExpr{Expr: x, Sel: sel}
}
