package gad

//go:generate go run ./cmd/mkoptypes

import "github.com/gad-lang/gad/token"

var (
	TOperator = &Type{TypeName: "Operator", Parent: TBase}
)

var (
	_ ObjectType = (BinaryOperatorType)(0)
)

type BinaryOperatorType token.Token

func (b BinaryOperatorType) Token() token.Token {
	return token.Token(b)
}

func (b BinaryOperatorType) String() string {
	return b.ToString()
}

func (b BinaryOperatorType) IsFalsy() bool {
	t := b.Token()
	return t > token.GroupBinaryOperatorBegin && t < token.GroupBinaryOperatorEnd
}

func (b BinaryOperatorType) Type() ObjectType {
	return TOperator
}

func (b BinaryOperatorType) ToString() string {
	t := b.Token()
	return t.Name() + ReprQuote(t.String())
}

func (b BinaryOperatorType) Equal(right Object) bool {
	if ob, ok := right.(BinaryOperatorType); ok {
		return ob == b
	}
	return false
}

func (BinaryOperatorType) Call(Call) (Object, error) {
	return nil, ErrNotCallable
}

func (b BinaryOperatorType) Name() string {
	return "TBinaryOperator" + b.Token().Name()
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

type SelfAssignOperatorType token.Token

func (b SelfAssignOperatorType) Token() token.Token {
	return token.Token(b)
}

func (b SelfAssignOperatorType) String() string {
	return b.ToString()
}

func (b SelfAssignOperatorType) IsFalsy() bool {
	t := b.Token()
	return t > token.GroupBinaryOperatorBegin && t < token.GroupBinaryOperatorEnd
}

func (b SelfAssignOperatorType) Type() ObjectType {
	return TOperator
}

func (b SelfAssignOperatorType) ToString() string {
	t := b.Token()
	return t.Name() + ReprQuote(t.String())
}

func (b SelfAssignOperatorType) Equal(right Object) bool {
	if ob, ok := right.(SelfAssignOperatorType); ok {
		return ob == b
	}
	return false
}

func (SelfAssignOperatorType) Call(Call) (Object, error) {
	return nil, ErrNotCallable
}

func (b SelfAssignOperatorType) Name() string {
	return "TSelfAssignOperator" + b.Token().Name()
}

func (SelfAssignOperatorType) Getters() Dict {
	return nil
}

func (SelfAssignOperatorType) Setters() Dict {
	return nil
}

func (SelfAssignOperatorType) Methods() Dict {
	return nil
}

func (SelfAssignOperatorType) Fields() Dict {
	return nil
}

func (SelfAssignOperatorType) New(*VM, Dict) (Object, error) {
	return nil, ErrNotInitializable
}

func (SelfAssignOperatorType) IsChildOf(t ObjectType) bool {
	return t == TOperator
}

func (SelfAssignOperatorType) MethodsDisabled() bool {
	return true
}
