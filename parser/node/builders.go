package node

type NamedArgBuilder struct {
	v NamedArgExpr
}

func ENamedArg() *NamedArgBuilder {
	return &NamedArgBuilder{}
}

func (b *NamedArgBuilder) Ident(v *IdentExpr) *NamedArgBuilder {
	b.v.Ident = v
	return b
}

func (b *NamedArgBuilder) Literal(v *StringLit) *NamedArgBuilder {
	b.v.Lit = v
	return b
}

func (b *NamedArgBuilder) Expr(v Expr) *NamedArgBuilder {
	b.v.Exp = v
	return b
}

func (b *NamedArgBuilder) Var() *NamedArgBuilder {
	b.v.Var = true
	return b
}

func (b *NamedArgBuilder) Build() *NamedArgExpr {
	return &b.v
}
