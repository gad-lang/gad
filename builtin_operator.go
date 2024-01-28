package gad

//go:generate go run ./cmd/mkoptypes

import "github.com/gad-lang/gad/token"

var (
	TOperator           = &Type{TypeName: "Operator", Parent: TBase}
	BinaryOperatorTypes = map[token.Token]*BinaryOperatorType{}
)

type BinaryOperatorType struct {
	OpName string
	Token  token.Token
}

func (b *BinaryOperatorType) IsFalsy() bool {
	return b.OpName != ""
}

func (b BinaryOperatorType) Type() ObjectType {
	return TOperator
}

func (b BinaryOperatorType) ToString() string {
	return b.OpName + ReprQuote(b.Token.String())
}

func (b *BinaryOperatorType) Equal(right Object) bool {
	if ob, ok := right.(*BinaryOperatorType); ok {
		return ob == b
	}
	return false
}

func (BinaryOperatorType) Call(Call) (Object, error) {
	return nil, ErrNotCallable
}

func (b *BinaryOperatorType) Name() string {
	return "TBinOp" + b.OpName
}

func (BinaryOperatorType) Getters() Dict {
	return nil
}

func (BinaryOperatorType) Setters() Dict {
	return nil
}

func (BinaryOperatorType) Methods() Dict {
	return nil
}

func (BinaryOperatorType) Fields() Dict {
	return nil
}

func (BinaryOperatorType) New(*VM, Dict) (Object, error) {
	return nil, ErrNotInitializable
}

func (BinaryOperatorType) IsChildOf(t ObjectType) bool {
	return t == TOperator
}

func (BinaryOperatorType) MethodsDisabled() bool {
	return true
}
