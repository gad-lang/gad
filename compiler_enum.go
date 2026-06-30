package gad

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// compileEnumStmt compiles the statement form `enum Name { … }` to
// `const Name = <enum expression>`.
func (c *Compiler) compileEnumStmt(nd *node.EnumStmt) error {
	name, _ := nd.NameExpr.(*node.IdentExpr)
	if name == nil {
		return c.errorf(nd, "enum statement requires a name identifier")
	}
	return c.Compile(&node.DeclStmt{
		Decl: &node.GenDecl{
			Tok: token.Const,
			Specs: []node.Spec{
				&node.ValueSpec{
					Idents: []*node.IdentExpr{name},
					Values: []node.Expr{&nd.EnumExpr},
				},
			},
		},
	})
}

// compileEnumExpr builds the enum value at compile time and pushes it as a
// constant.
func (c *Compiler) compileEnumExpr(nd *node.EnumExpr) error {
	var name string
	if id, _ := nd.NameExpr.(*node.IdentExpr); id != nil {
		name = id.Name
	}
	enum, err := c.buildEnum(nd, name)
	if err != nil {
		return err
	}
	c.emit(nd, OpConstant, c.addConstant(enum))
	return nil
}

// enumNum is a field's computed numeric value: a 64-bit magnitude plus whether
// it is unsigned (uint) or signed (int).
type enumNum struct {
	val      int64
	unsigned bool
}

func (n enumNum) object() Object {
	if n.unsigned && n.val >= 0 {
		return Uint(uint64(n.val))
	}
	return Int(n.val)
}

// buildEnum computes the field values of an enum and returns the *Enum.
//
// Values are assigned left to right. Without an explicit `= expr`, a field's
// value is the previous magnitude + 1 (or 1 for the first field); under `bit`
// it is 1<<n for the n-th bit field. A `+`/`-` prefix makes a field signed and
// sets the running sign (a sign or a signed value propagates to later defaulted
// fields); an explicit value's type (int/uint) likewise propagates. A `_` field
// advances the running value but is not added to the enum. Explicit values may
// reference earlier fields and use integer operators (`All = Read | Write`).
func (c *Compiler) buildEnum(nd *node.EnumExpr, name string) (*Enum, error) {
	enum := NewEnum(name, c.module)

	var (
		hasPrev      bool
		prev         int64
		prevUnsigned = true
		negative     bool
		bitMode      bool
		bitPos       int
	)
	vals := map[string]enumNum{}

	for _, f := range nd.Fields {
		if f.Bit {
			bitMode = true
		}
		signNeg := f.Sign == token.Sub
		signPos := f.Sign == token.Add
		hasSign := signNeg || signPos

		var n enumNum

		if f.Value != nil {
			v, err := c.evalEnumExpr(f.Value, vals)
			if err != nil {
				return nil, err
			}
			n = v
			switch {
			case signNeg:
				n.val, n.unsigned = -absI(n.val), false
			case signPos:
				n.val, n.unsigned = absI(n.val), false
			}
		} else {
			if hasSign {
				negative = signNeg
			}
			var mag int64
			if bitMode {
				mag = int64(1) << uint(bitPos)
				bitPos++
			} else if hasPrev {
				mag = absI(prev) + 1
			} else {
				mag = 1
			}
			if negative {
				n.val, n.unsigned = -mag, false
			} else {
				n.val, n.unsigned = mag, prevUnsigned && !hasSign
			}
		}

		hasPrev = true
		prev = n.val
		prevUnsigned = n.unsigned
		negative = n.val < 0

		if f.Name.Empty || f.Name.Name == "_" {
			continue
		}
		vals[f.Name.Name] = n
		enum.AddValue(f.Name.Name, n.object())
	}

	return enum, nil
}

// evalEnumExpr evaluates a field value expression at compile time. It resolves
// references to earlier fields, integer literals and the integer unary/binary
// operators. The result is unsigned only when both operands are unsigned.
func (c *Compiler) evalEnumExpr(e node.Expr, vals map[string]enumNum) (enumNum, error) {
	switch t := e.(type) {
	case *node.IntLit:
		return enumNum{val: t.Value, unsigned: false}, nil
	case *node.UintLit:
		return enumNum{val: int64(t.Value), unsigned: true}, nil
	case *node.IdentExpr:
		if n, ok := vals[t.Name]; ok {
			return n, nil
		}
		return enumNum{}, c.errorf(e, "enum value references unknown field %q", t.Name)
	case *node.ParenExpr:
		return c.evalEnumExpr(t.Expr, vals)
	case *node.UnaryExpr:
		v, err := c.evalEnumExpr(t.Expr, vals)
		if err != nil {
			return enumNum{}, err
		}
		switch t.Token {
		case token.Sub:
			return enumNum{val: -v.val, unsigned: false}, nil
		case token.Add:
			return enumNum{val: v.val, unsigned: false}, nil
		case token.Xor:
			return enumNum{val: ^v.val, unsigned: v.unsigned}, nil
		}
		return enumNum{}, c.errorf(e, "unsupported enum unary operator %s", t.Token)
	case *node.BinaryExpr:
		l, err := c.evalEnumExpr(t.LHS, vals)
		if err != nil {
			return enumNum{}, err
		}
		r, err := c.evalEnumExpr(t.RHS, vals)
		if err != nil {
			return enumNum{}, err
		}
		u := l.unsigned && r.unsigned
		switch t.Token {
		case token.Add:
			return enumNum{l.val + r.val, u}, nil
		case token.Sub:
			return enumNum{l.val - r.val, u}, nil
		case token.Mul:
			return enumNum{l.val * r.val, u}, nil
		case token.Quo:
			if r.val == 0 {
				return enumNum{}, c.errorf(e, "enum value division by zero")
			}
			return enumNum{l.val / r.val, u}, nil
		case token.Rem:
			if r.val == 0 {
				return enumNum{}, c.errorf(e, "enum value division by zero")
			}
			return enumNum{l.val % r.val, u}, nil
		case token.Or:
			return enumNum{l.val | r.val, u}, nil
		case token.And:
			return enumNum{l.val & r.val, u}, nil
		case token.Xor:
			return enumNum{l.val ^ r.val, u}, nil
		case token.Shl:
			return enumNum{l.val << uint(r.val), u}, nil
		case token.Shr:
			return enumNum{l.val >> uint(r.val), u}, nil
		}
		return enumNum{}, c.errorf(e, "unsupported enum binary operator %s", t.Token)
	}
	return enumNum{}, c.errorf(e, "invalid enum value expression %T", e)
}

func absI(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
